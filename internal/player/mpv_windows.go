//go:build windows

package player

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

type MPV struct {
	mu                               sync.Mutex
	dll                              *windows.DLL
	handle                           uintptr
	command, set, get, free, destroy *windows.Proc
	wait, errorString                *windows.Proc
	loading                          bool
	lastError                        string
}

func New() (Engine, error) {
	exe, _ := os.Executable()
	candidates := []string{os.Getenv("BILITUNE_MPV_PATH"), filepath.Join(filepath.Dir(exe), "libmpv-2.dll"), filepath.Join("bin", "libmpv-2.dll")}
	var dll *windows.DLL
	var err error
	for _, path := range candidates {
		if path == "" {
			continue
		}
		if _, e := os.Stat(path); e != nil {
			continue
		}
		dll, err = windows.LoadDLL(path)
		if err == nil {
			break
		}
	}
	if dll == nil {
		return nil, fmt.Errorf("未找到音频引擎，请保留程序同目录中的 libmpv-2.dll")
	}
	proc := func(name string) (*windows.Proc, error) { return dll.FindProc(name) }
	create, err := proc("mpv_create")
	if err != nil {
		dll.Release()
		return nil, err
	}
	initialize, err := proc("mpv_initialize")
	if err != nil {
		dll.Release()
		return nil, err
	}
	setOption, err := proc("mpv_set_option_string")
	if err != nil {
		dll.Release()
		return nil, err
	}
	m := &MPV{dll: dll}
	if m.command, err = proc("mpv_command_string"); err != nil {
		dll.Release()
		return nil, err
	}
	m.set, _ = proc("mpv_set_property_string")
	m.get, _ = proc("mpv_get_property_string")
	m.free, _ = proc("mpv_free")
	m.destroy, _ = proc("mpv_terminate_destroy")
	m.wait, _ = proc("mpv_wait_event")
	m.errorString, _ = proc("mpv_error_string")
	if m.set == nil || m.get == nil || m.free == nil || m.destroy == nil {
		dll.Release()
		return nil, fmt.Errorf("音频引擎接口不完整")
	}
	m.handle, _, _ = create.Call()
	if m.handle == 0 {
		dll.Release()
		return nil, fmt.Errorf("无法创建音频引擎")
	}
	for key, value := range map[string]string{"video": "no", "audio-display": "no", "terminal": "no", "input-default-bindings": "no", "input-vo-keyboard": "no", "input-media-keys": "no", "keep-open": "yes", "cache": "yes", "cache-secs": "60"} {
		k, _ := windows.BytePtrFromString(key)
		v, _ := windows.BytePtrFromString(value)
		setOption.Call(m.handle, uintptr(unsafe.Pointer(k)), uintptr(unsafe.Pointer(v)))
		runtime.KeepAlive(k)
		runtime.KeepAlive(v)
	}
	code, _, _ := initialize.Call(m.handle)
	if int32(code) < 0 {
		m.Close()
		return nil, fmt.Errorf("无法初始化音频引擎（%d）", int32(code))
	}
	return m, nil
}
func (m *MPV) cmd(value string) error {
	if m.handle == 0 {
		return fmt.Errorf("音频引擎已关闭")
	}
	ptr, err := windows.BytePtrFromString(value)
	if err != nil {
		return err
	}
	code, _, _ := m.command.Call(m.handle, uintptr(unsafe.Pointer(ptr)))
	runtime.KeepAlive(ptr)
	if int32(code) < 0 {
		return fmt.Errorf("音频引擎执行失败（%d）", int32(code))
	}
	return nil
}
func (m *MPV) property(key, value string) error {
	if m.handle == 0 {
		return fmt.Errorf("音频引擎已关闭")
	}
	k, err := windows.BytePtrFromString(key)
	if err != nil {
		return err
	}
	v, err := windows.BytePtrFromString(value)
	if err != nil {
		return err
	}
	code, _, _ := m.set.Call(m.handle, uintptr(unsafe.Pointer(k)), uintptr(unsafe.Pointer(v)))
	runtime.KeepAlive(k)
	runtime.KeepAlive(v)
	if int32(code) < 0 {
		return fmt.Errorf("无法设置 %s（%d）", key, int32(code))
	}
	return nil
}
func (m *MPV) read(key string) string {
	k, _ := windows.BytePtrFromString(key)
	ptr, _, _ := m.get.Call(m.handle, uintptr(unsafe.Pointer(k)))
	runtime.KeepAlive(k)
	if ptr == 0 {
		return ""
	}
	defer m.free.Call(ptr)
	length := 0
	for length < 65536 && *(*byte)(unsafe.Pointer(ptr + uintptr(length))) != 0 {
		length++
	}
	return string(unsafe.Slice((*byte)(unsafe.Pointer(ptr)), length))
}
func (m *MPV) Load(raw string, headers map[string]string, start float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.handle == 0 {
		return fmt.Errorf("音频引擎已关闭")
	}
	fields := []string{}
	for k, v := range headers {
		fields = append(fields, k+": "+strings.ReplaceAll(v, ",", "\\,"))
	}
	if err := m.property("http-header-fields", strings.Join(fields, ",")); err != nil {
		return err
	}
	m.events()
	m.lastError = ""
	if err := m.cmd("loadfile " + strconv.Quote(raw) + " replace start=" + strconv.FormatFloat(start, 'f', 3, 64)); err != nil {
		return err
	}
	m.loading = true
	return m.property("pause", "no")
}
func (m *MPV) Play() error  { m.mu.Lock(); defer m.mu.Unlock(); return m.property("pause", "no") }
func (m *MPV) Pause() error { m.mu.Lock(); defer m.mu.Unlock(); return m.property("pause", "yes") }
func (m *MPV) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.property("pause", "yes"); err != nil {
		return err
	}
	m.loading = false
	return m.cmd("stop")
}
func (m *MPV) Seek(value float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cmd("seek " + strconv.FormatFloat(value, 'f', 3, 64) + " absolute")
}
func (m *MPV) SetVolume(value float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.property("volume", strconv.FormatFloat(value*100, 'f', 2, 64))
}
func (m *MPV) SetRate(value float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.property("speed", strconv.FormatFloat(value, 'f', 2, 64))
}
func (m *MPV) Snapshot() Snapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.handle == 0 {
		return Snapshot{}
	}
	m.events()
	number := func(name string) float64 { n, _ := strconv.ParseFloat(m.read(name), 64); return n }
	return Snapshot{Playing: m.read("pause") == "no" && (m.loading || m.read("idle-active") != "yes"), Position: number("time-pos"), Duration: number("duration"), Buffered: number("demuxer-cache-time"), Buffering: m.loading || m.read("paused-for-cache") == "yes", Ended: !m.loading && m.read("eof-reached") == "yes", Error: m.lastError}
}

// libmpv reports network/decoder failures asynchronously after loadfile returns.
func (m *MPV) events() {
	if m.wait == nil || m.handle == 0 {
		return
	}
	for i := 0; i < 128; i++ {
		ptr, _, _ := m.wait.Call(m.handle, 0)
		if ptr == 0 {
			return
		}
		event := (*struct {
			ID    int32
			Error int32
			Reply uint64
			Data  uintptr
		})(unsafe.Pointer(ptr))
		if event.ID == 0 {
			return
		}
		if event.ID == 8 {
			m.loading = false
		}
		if event.ID == 7 && event.Data != 0 {
			end := (*struct {
				Reason int32
				Error  int32
			})(unsafe.Pointer(event.Data))
			if end.Reason == 4 {
				m.loading = false
				m.lastError = fmt.Sprintf("无法播放此音频（%d），请重试或检查网络与账号权限", end.Error)
			}
		}
	}
}
func (m *MPV) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.handle != 0 {
		m.destroy.Call(m.handle)
		m.handle = 0
	}
	if m.dll != nil {
		m.dll.Release()
		m.dll = nil
	}
}
