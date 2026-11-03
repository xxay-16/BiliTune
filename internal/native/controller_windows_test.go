//go:build windows

package native

import (
	"github.com/xxay-16/BiliTune/internal/model"
	"golang.org/x/sys/windows"
	"os"
	"runtime"
	"testing"
	"time"
	"unsafe"
)

func TestVirtualKeys(t *testing.T) {
	for key, want := range map[string]uintptr{"A": 0x41, "z": 0x5a, "F24": 0x87, "Numpad0": 0x60, "MediaPlayPause": 0xb3, "MediaPreviousTrack": 0xb1, "Backquote": 0xc0, "F25": 0, "unsupported": 0} {
		if got := VirtualKey(key); got != want {
			t.Errorf("%s: %d != %d", key, got, want)
		}
	}
}
func TestMediaDispatchKeepsPlayPauseIdempotent(t *testing.T) {
	actions := make(chan string, 3)
	c := &Controller{notify: func(s string) { actions <- s }, mediaOverrides: map[string]string{"MediaPlayPause": "playPause"}}
	c.dispatch("pause")
	select {
	case action := <-actions:
		if action != "pause" {
			t.Fatal(action)
		}
	case <-time.After(time.Second):
		t.Fatal("no event")
	}
	c.suspended = true
	c.dispatch("nextTrack")
	select {
	case <-actions:
		t.Fatal("dispatched while recording")
	case <-time.After(20 * time.Millisecond):
	}
}
func TestWindowsMediaAndHotkeySmoke(t *testing.T) {
	if os.Getenv("BILITUNE_TEST_NATIVE") == "" {
		t.Skip("interactive Windows integration; set BILITUNE_TEST_NATIVE=1")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	class, _ := windows.UTF16PtrFromString("STATIC")
	name, _ := windows.UTF16PtrFromString("BiliTune native test")
	hwnd, _, err := user32.NewProc("CreateWindowExW").Call(0, uintptr(unsafe.Pointer(class)), uintptr(unsafe.Pointer(name)), 0, 0, 0, 200, 100, 0, 0, 0, 0)
	if hwnd == 0 {
		t.Fatal(err)
	}
	defer user32.NewProc("DestroyWindow").Call(hwnd)
	c := New(hwnd, func(string) {})
	defer c.Close()
	if c.media == nil {
		t.Fatal("SMTC unavailable")
	}
	errors := c.Apply([]model.Hotkey{{Action: "nextTrack", Key: "F23", Modifiers: []string{"control", "alt", "shift"}}})
	if len(errors) > 0 {
		t.Fatal(errors)
	}
	c.Media(model.Player{Track: &model.Track{Title: "超长标题测试", Artist: "BiliTune"}, Playing: true})
	var status int32
	if failed(com(c.media.controls, 6, uintptr(unsafe.Pointer(&status)))) || status != 3 {
		t.Fatalf("SMTC status %d", status)
	}
	if len(c.Record(false)) > 0 {
		t.Fatal("hotkeys not restored")
	}
}
