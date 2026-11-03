package main

import (
	"context"
	"crypto/sha256"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	core "github.com/xxay-16/BiliTune/internal/app"
	"github.com/xxay-16/BiliTune/internal/bili"
	"github.com/xxay-16/BiliTune/internal/model"
	"github.com/xxay-16/BiliTune/internal/native"
	"github.com/xxay-16/BiliTune/internal/player"
	"github.com/xxay-16/BiliTune/internal/store"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed frontend/public/logo.png
var icon []byte

func main() {
	dataDir := flag.String("data-dir", "", "独立的数据目录")
	hidden := flag.Bool("hidden", false, "启动到系统托盘")
	preview := flag.String("preview", "", "仅供开发：在本机地址预览界面")
	flag.Parse()
	if *preview != "" && *dataDir == "" {
		log.Fatal("界面预览必须指定独立的 --data-dir")
	}
	st, state, err := store.Open(*dataDir)
	if err != nil {
		log.Fatal(err)
	}
	cookie, err := st.Cookie()
	if err != nil {
		log.Fatal("无法读取登录凭据：", err)
	}
	client := bili.New(cookie)
	engine, engineErr := player.New()
	message := ""
	if engineErr != nil {
		message = engineErr.Error()
	}
	service := core.New(st, state, client, engine, message)
	defer service.Close()
	frontend, _ := fs.Sub(assets, "frontend/dist")
	api := service.Handler()
	middleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api/") {
				api.ServeHTTP(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
	if *preview != "" {
		if !strings.HasPrefix(*preview, "127.0.0.1:") {
			log.Fatal("预览仅允许绑定 127.0.0.1")
		}
		server := &http.Server{Addr: *preview, Handler: middleware(http.FileServer(http.FS(frontend))), ReadHeaderTimeout: 5 * time.Second}
		service.Start()
		go func() {
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
			defer stop()
			<-ctx.Done()
			_ = server.Shutdown(context.Background())
		}()
		log.Printf("预览地址 http://%s（独立数据目录）", *preview)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
		return
	}
	service.SetAutoStartInitial(native.StartupEnabled())
	var controller *native.Controller
	var mainWindow, popup *application.WebviewWindow
	var tray *application.SystemTray
	var quitting atomic.Bool
	show := func() {
		if mainWindow != nil {
			mainWindow.UnMinimise()
			mainWindow.Show()
			mainWindow.Focus()
		}
	}
	hash := sha256.Sum256([]byte(st.Dir))
	desktop := application.New(application.Options{
		Name: "BiliTune", Description: "Bilibili 桌面音乐播放器", Icon: icon,
		Assets: application.AssetOptions{Handler: application.AssetFileServerFS(frontend), Middleware: middleware},
		Windows: application.WindowsOptions{DisableQuitOnLastWindowClosed: true, AdditionalBrowserArgs: browserDebugArgs(), WebviewUserDataPath: filepath.Join(st.Dir, "webview"), WndProcInterceptor: func(hwnd uintptr, msg uint32, w, l uintptr) (uintptr, bool) {
			if controller != nil {
				return controller.Intercept(hwnd, msg, w, l)
			}
			return 0, false
		}},
		SingleInstance: &application.SingleInstanceOptions{UniqueID: fmt.Sprintf("BiliTune-Desktop-%x", hash[:8]), OnSecondInstanceLaunch: func(application.SecondInstanceData) { show() }},
		OnShutdown: func() {
			quitting.Store(true)
			service.Close()
			application.InvokeSync(func() {
				if controller != nil {
					controller.Close()
				}
			})
		},
	})
	windowBackground := application.NewRGB(255, 255, 255)
	if state.Settings.Theme == "dark" {
		windowBackground = application.NewRGB(23, 23, 24)
	}
	mainWindow = desktop.Window.NewWithOptions(application.WebviewWindowOptions{Name: "main", Title: "BiliTune", Width: 1380, Height: 900, MinWidth: 1060, MinHeight: 690, Frameless: true, Hidden: *hidden, BackgroundColour: windowBackground, URL: "/", DefaultContextMenuDisabled: true})
	popup = desktop.Window.NewWithOptions(application.WebviewWindowOptions{Name: "tray-menu", Title: "BiliTune 托盘", Width: 180, Height: 302, MinWidth: 180, MaxWidth: 180, MinHeight: 302, MaxHeight: 302, Frameless: true, AlwaysOnTop: true, DisableResize: true, Hidden: true, BackgroundColour: application.NewRGB(23, 23, 28), URL: "/#tray", DefaultContextMenuDisabled: true, Windows: application.WindowsWindow{HiddenOnTaskbar: true}})
	tray = desktop.SystemTray.New()
	tray.SetIcon(icon)
	tray.SetTooltip("BiliTune")
	tray.AttachWindow(popup)
	tray.WindowOffset(6)
	tray.OnClick(show)
	tray.OnRightClick(func() { tray.ShowWindow(); popup.Focus() })
	popup.OnWindowEvent(events.Common.WindowLostFocus, func(*application.WindowEvent) { tray.HideWindow() })
	closeMain := func() {
		switch service.Bootstrap().Settings.CloseBehavior {
		case "exit":
			quitting.Store(true)
			desktop.Quit()
		case "minimize":
			mainWindow.Minimise()
		default:
			mainWindow.Hide()
		}
	}
	mainWindow.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		if !quitting.Load() {
			e.Cancel()
			closeMain()
		}
	})
	service.SetDesktop(core.Desktop{
		Window: func(action string) error {
			switch action {
			case "show":
				show()
			case "hide":
				mainWindow.Hide()
			case "toggleWindow":
				if mainWindow.IsVisible() && !mainWindow.IsMinimised() {
					mainWindow.Hide()
				} else {
					show()
				}
			case "minimize":
				mainWindow.Minimise()
			case "maximize":
				mainWindow.ToggleMaximise()
			case "close":
				closeMain()
			case "tray-hide":
				tray.HideWindow()
			case "exit":
				quitting.Store(true)
				go desktop.Quit()
			default:
				return fmt.Errorf("未知窗口操作")
			}
			return nil
		},
		Directory: func() (string, error) {
			return desktop.Dialog.OpenFile().CanChooseDirectories(true).CanChooseFiles(false).SetTitle("选择下载目录").AttachToWindow(mainWindow).PromptForSingleSelection()
		},
		Files: func() ([]string, error) {
			return desktop.Dialog.OpenFile().AddFilter("音频文件", "*.mp3;*.m4a;*.mp4;*.flac;*.wav;*.ogg;*.opus;*.aac;*.wma").SetTitle("打开本地音频").AttachToWindow(mainWindow).PromptForMultipleSelection()
		},
		Reveal: native.Reveal, OpenURL: desktop.Browser.OpenURL,
		Configure: func(settings model.Settings, autostart bool) map[string]string {
			errors := map[string]string{}
			application.InvokeSync(func() {
				if controller != nil {
					errors = controller.Apply(settings.Hotkeys)
				}
				if autostart {
					if err := native.SetStartup(settings.AutoStart); err != nil {
						errors["autoStart"] = err.Error()
					}
				}
			})
			return errors
		},
		Record: func(record bool) {
			application.InvokeSync(func() {
				if controller != nil {
					service.SetHotkeyErrors(controller.Record(record))
				}
			})
		},
		Media: func(p model.Player) {
			if !quitting.Load() {
				application.InvokeAsync(func() {
					if controller != nil && !quitting.Load() {
						controller.Media(p)
					}
				})
			}
		},
	})
	mainWindow.OnWindowEvent(events.Common.WindowRuntimeReady, func(*application.WindowEvent) {
		application.InvokeSync(func() {
			if controller != nil || mainWindow.NativeWindow() == nil {
				return
			}
			controller = native.New(uintptr(mainWindow.NativeWindow()), func(action string) {
				if err := service.Command(action, 0); err != nil {
					log.Printf("快捷键操作失败：%v", err)
				}
			})
			service.SetHotkeyErrors(controller.Apply(service.Bootstrap().Settings.Hotkeys))
		})
	})
	desktop.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) { service.Start() })
	if err := desktop.Run(); err != nil {
		log.Fatal(err)
	}
}
