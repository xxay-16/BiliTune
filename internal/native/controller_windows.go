//go:build windows

package native

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"unsafe"

	"github.com/xxay-16/BiliTune/internal/model"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

var user32 = windows.NewLazySystemDLL("user32.dll")
var registerHotkey = user32.NewProc("RegisterHotKey")
var unregisterHotkey = user32.NewProc("UnregisterHotKey")
var getCursor = user32.NewProc("GetCursorPos")
var getPhysicalCursor = user32.NewProc("GetPhysicalCursorPos")
var postMessage = user32.NewProc("PostMessageW")

const mediaMessage = 0x8000 + 0x350

type Controller struct {
	hwnd                   uintptr
	notify                 func(string)
	bindings               []model.Hotkey
	actions                map[uintptr]string
	mediaOverrides         map[string]string
	media                  *mediaSession
	suspended              bool
	lastTitle, lastArtist  string
	lastPlaying, lastTrack bool
}

func New(hwnd uintptr, notify func(string)) *Controller {
	return &Controller{hwnd: hwnd, notify: notify, actions: map[uintptr]string{}, mediaOverrides: map[string]string{}, media: newMediaSession(hwnd)}
}
func (c *Controller) clear() {
	for id := range c.actions {
		unregisterHotkey.Call(c.hwnd, id)
	}
	c.actions = map[uintptr]string{}
	c.mediaOverrides = map[string]string{}
}
func (c *Controller) Apply(bindings []model.Hotkey) map[string]string {
	c.bindings = append([]model.Hotkey{}, bindings...)
	return c.register()
}
func (c *Controller) register() map[string]string {
	c.clear()
	errors := map[string]string{}
	if c.suspended {
		return errors
	}
	seen := map[string]bool{}
	id := uintptr(12000)
	add := func(binding model.Hotkey, isDefault bool) {
		key := VirtualKey(binding.Key)
		mods := uintptr(0x4000)
		for _, modifier := range binding.Modifiers {
			switch modifier {
			case "alt":
				mods |= 1
			case "control":
				mods |= 2
			case "shift":
				mods |= 4
			case "win":
				mods |= 8
			}
		}
		signature := fmt.Sprintf("%d/%d", mods, key)
		if seen[signature] {
			if !isDefault {
				errors[binding.Action] = "此组合键已被另一个操作使用"
			}
			return
		}
		seen[signature] = true
		if key == 0 {
			errors[binding.Action] = "暂不支持此按键，请重新录制"
			return
		}
		if c.media != nil && mods == 0x4000 && key >= 0xB0 && key <= 0xB3 {
			c.mediaOverrides[binding.Key] = binding.Action
			return
		}
		success, _, err := registerHotkey.Call(c.hwnd, id, mods, key)
		if success == 0 {
			errors[binding.Action] = fmt.Sprintf("快捷键注册失败，可能被其他程序或旧版 BiliTune 占用（%v）", err)
			return
		}
		c.actions[id] = binding.Action
		id++
	}
	for _, binding := range c.bindings {
		add(binding, false)
	}
	if c.media == nil {
		for _, binding := range []model.Hotkey{{Action: "playPause", Key: "MediaPlayPause"}, {Action: "previousTrack", Key: "MediaPreviousTrack"}, {Action: "nextTrack", Key: "MediaNextTrack"}, {Action: "stop", Key: "MediaStop"}} {
			add(binding, true)
		}
	}
	return errors
}
func (c *Controller) Record(recording bool) map[string]string {
	c.suspended = recording
	errors := map[string]string{}
	if recording {
		c.clear()
	} else {
		errors = c.register()
	}
	if c.media != nil {
		c.media.enabled(c.lastTrack && !recording)
	}
	return errors
}
func (c *Controller) dispatch(action string) {
	if c.suspended {
		return
	}
	keys := map[string]string{"play": "MediaPlayPause", "pause": "MediaPlayPause", "playPause": "MediaPlayPause", "previousTrack": "MediaPreviousTrack", "nextTrack": "MediaNextTrack", "stop": "MediaStop"}
	if override := c.mediaOverrides[keys[action]]; override != "" && !(override == "playPause" && (action == "play" || action == "pause")) {
		action = override
	}
	if action != "" {
		go c.notify(action)
	}
}
func (c *Controller) Intercept(hwnd uintptr, msg uint32, wparam, lparam uintptr) (uintptr, bool) {
	if c == nil {
		return 0, false
	}
	if msg == 0x0312 {
		if action := c.actions[wparam]; action != "" {
			c.dispatch(action)
			return 0, true
		}
	}
	command := 0
	if msg == 0x0319 {
		command = int((lparam >> 16) & 0xfff)
	} else if msg == mediaMessage {
		command = int(wparam)
	}
	if command != 0 {
		action := map[int]string{14: "playPause", 46: "play", 47: "pause", 11: "nextTrack", 12: "previousTrack", 13: "stop"}[command]
		if action != "" && !c.suspended {
			c.dispatch(action)
			return 1, true
		}
	}
	return 0, false
}
func (c *Controller) Media(player model.Player) {
	if c == nil || c.media == nil {
		return
	}
	title, artist := "BiliTune", ""
	hasTrack := player.Track != nil
	if hasTrack {
		title = player.Track.Title
		artist = player.Track.Artist
	}
	if title == c.lastTitle && artist == c.lastArtist && player.Playing == c.lastPlaying && hasTrack == c.lastTrack {
		return
	}
	c.lastTitle = title
	c.lastArtist = artist
	c.lastPlaying = player.Playing
	c.lastTrack = hasTrack
	c.media.update(title, artist, player.Playing, hasTrack && !c.suspended)
}
func (c *Controller) Close() {
	if c == nil {
		return
	}
	c.clear()
	if c.media != nil {
		c.media.close()
		c.media = nil
	}
}

func VirtualKey(key string) uintptr {
	if len(key) == 1 {
		char := strings.ToUpper(key)[0]
		if char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' {
			return uintptr(char)
		}
	}
	if len(key) >= 2 && len(key) <= 3 && key[0] == 'F' {
		n := 0
		for _, r := range key[1:] {
			if r < '0' || r > '9' {
				return 0
			}
			n = n*10 + int(r-'0')
		}
		if n >= 1 && n <= 24 {
			return uintptr(0x70 + n - 1)
		}
	}
	if len(key) == 7 && strings.HasPrefix(key, "Numpad") && key[6] >= '0' && key[6] <= '9' {
		return uintptr(0x60 + key[6] - '0')
	}
	return map[string]uintptr{"Space": 0x20, "Enter": 0x0d, "Escape": 0x1b, "Backspace": 8, "Tab": 9, "ArrowLeft": 0x25, "ArrowRight": 0x27, "ArrowUp": 0x26, "ArrowDown": 0x28, "Home": 0x24, "End": 0x23, "PageUp": 0x21, "PageDown": 0x22, "Insert": 0x2d, "Delete": 0x2e, "MediaPlayPause": 0xb3, "MediaStop": 0xb2, "MediaNextTrack": 0xb0, "MediaPreviousTrack": 0xb1, "NumpadAdd": 0x6b, "NumpadSubtract": 0x6d, "NumpadMultiply": 0x6a, "NumpadDivide": 0x6f, "NumpadDecimal": 0x6e, "Semicolon": 0xba, "Equal": 0xbb, "Comma": 0xbc, "Minus": 0xbd, "Period": 0xbe, "Slash": 0xbf, "Backquote": 0xc0, "BracketLeft": 0xdb, "Backslash": 0xdc, "BracketRight": 0xdd, "Quote": 0xde}[key]
}

func Cursor() (int, int) {
	var point struct{ X, Y int32 }
	if ok, _, _ := getPhysicalCursor.Call(uintptr(unsafe.Pointer(&point))); ok == 0 {
		getCursor.Call(uintptr(unsafe.Pointer(&point)))
	}
	return int(point.X), int(point.Y)
}
func Reveal(path string) error {
	if _, err := os.Stat(path); err != nil {
		return err
	}
	command := exec.Command("explorer.exe", "/select,", path)
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return command.Start()
}
func StartupEnabled() bool {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()
	value, _, err := key.GetStringValue("BiliTune")
	return err == nil && value != ""
}
func SetStartup(enabled bool) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	if enabled {
		exe, err := os.Executable()
		if err != nil {
			return err
		}
		return key.SetStringValue("BiliTune", `"`+exe+`" --hidden`)
	}
	err = key.DeleteValue("BiliTune")
	if err == windows.ERROR_FILE_NOT_FOUND {
		return nil
	}
	return err
}
