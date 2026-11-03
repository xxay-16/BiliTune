//go:build windows

package native

import (
	"runtime"
	"sync/atomic"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var combase = windows.NewLazySystemDLL("combase.dll")
var roInitialize = combase.NewProc("RoInitialize")
var roUninitialize = combase.NewProc("RoUninitialize")
var roFactory = combase.NewProc("RoGetActivationFactory")
var createString = combase.NewProc("WindowsCreateString")
var deleteString = combase.NewProc("WindowsDeleteString")
var interopIID = guid("ddb0472d-c911-4a1f-86d9-dc3d71a95f5a")
var controlsIID = guid("99fa3ff4-1742-42a6-902e-087d41f965ec")
var unknownIID = guid("00000000-0000-0000-c000-000000000046")
var agileIID = guid("94ea2b94-e9cc-49e0-c0ff-ee64ca8f5b90")
var delegateIID = guid("0557e996-7b23-5bae-aa81-ea0d671143a4")

func guid(raw string) windows.GUID { value, _ := windows.GUIDFromString("{" + raw + "}"); return value }
func com(object uintptr, index int, args ...uintptr) uintptr {
	if object == 0 {
		return 0x80004003
	}
	table := *(*uintptr)(unsafe.Pointer(object))
	method := *(*uintptr)(unsafe.Pointer(table + uintptr(index)*unsafe.Sizeof(uintptr(0))))
	values := append([]uintptr{object}, args...)
	result, _, _ := syscall.SyscallN(method, values...)
	return result
}
func failed(result uintptr) bool { return int32(result) < 0 }
func hstring(raw string) uintptr {
	chars := windows.StringToUTF16(raw)
	var result uintptr
	code, _, _ := createString.Call(uintptr(unsafe.Pointer(&chars[0])), uintptr(len(chars)-1), uintptr(unsafe.Pointer(&result)))
	runtime.KeepAlive(chars)
	if failed(code) {
		return 0
	}
	return result
}
func release(object uintptr) {
	if object != 0 {
		com(object, 2)
	}
}

type eventDelegate struct {
	Table      *[4]uintptr
	References atomic.Uint32
	Window     uintptr
}
type mediaSession struct {
	controls, display, music uintptr
	token                    int64
	delegate                 *eventDelegate
	apartment                bool
	thread                   uint32
}

func newMediaSession(hwnd uintptr) *mediaSession {
	code, _, _ := roInitialize.Call(0)
	apartment := !failed(code)
	if failed(code) && uint32(code) != 0x80010106 {
		return nil
	}
	s := &mediaSession{apartment: apartment, thread: windows.GetCurrentThreadId()}
	class := hstring("Windows.Media.SystemMediaTransportControls")
	if class == 0 {
		s.close()
		return nil
	}
	defer deleteString.Call(class)
	var factory uintptr
	code, _, _ = roFactory.Call(class, uintptr(unsafe.Pointer(&interopIID)), uintptr(unsafe.Pointer(&factory)))
	if failed(code) {
		s.close()
		return nil
	}
	defer release(factory)
	if failed(com(factory, 6, hwnd, uintptr(unsafe.Pointer(&controlsIID)), uintptr(unsafe.Pointer(&s.controls)))) {
		s.close()
		return nil
	}
	com(s.controls, 11, 0)
	for _, index := range []int{13, 15, 17, 25, 27} {
		com(s.controls, index, 1)
	}
	if failed(com(s.controls, 8, uintptr(unsafe.Pointer(&s.display)))) {
		s.close()
		return nil
	}
	com(s.display, 7, 1)
	if failed(com(s.display, 12, uintptr(unsafe.Pointer(&s.music)))) {
		s.close()
		return nil
	}
	delegate := &eventDelegate{Window: hwnd}
	delegate.References.Store(1)
	table := &[4]uintptr{}
	table[0] = windows.NewCallback(func(self, requested, out uintptr) uintptr {
		if requested == 0 || out == 0 {
			return 0x80004003
		}
		iid := *(*windows.GUID)(unsafe.Pointer(requested))
		if iid == unknownIID || iid == agileIID || iid == delegateIID {
			*(*uintptr)(unsafe.Pointer(out)) = self
			delegate.References.Add(1)
			return 0
		}
		*(*uintptr)(unsafe.Pointer(out)) = 0
		return 0x80004002
	})
	table[1] = windows.NewCallback(func(self uintptr) uintptr { return uintptr(delegate.References.Add(1)) })
	table[2] = windows.NewCallback(func(self uintptr) uintptr { return uintptr(delegate.References.Add(^uint32(0))) })
	table[3] = windows.NewCallback(func(self, sender, args uintptr) uintptr {
		var button int32
		if failed(com(args, 6, uintptr(unsafe.Pointer(&button)))) {
			return 0
		}
		commands := map[int32]uintptr{0: 46, 1: 47, 2: 13, 6: 11, 7: 12}
		if command := commands[button]; command != 0 {
			postMessage.Call(hwnd, mediaMessage, command, 0)
		}
		return 0
	})
	delegate.Table = table
	s.delegate = delegate
	if failed(com(s.controls, 32, uintptr(unsafe.Pointer(delegate)), uintptr(unsafe.Pointer(&s.token)))) {
		s.close()
		return nil
	}
	return s
}
func (s *mediaSession) enabled(value bool) {
	if s == nil || s.controls == 0 {
		return
	}
	v := uintptr(0)
	if value {
		v = 1
	}
	com(s.controls, 11, v)
}
func (s *mediaSession) update(title, artist string, playing, hasTrack bool) {
	if s == nil {
		return
	}
	s.enabled(hasTrack)
	status := uintptr(0)
	if hasTrack {
		status = 4
		if playing {
			status = 3
		}
	}
	com(s.controls, 7, status)
	name := hstring(title)
	defer deleteString.Call(name)
	who := hstring(artist)
	defer deleteString.Call(who)
	com(s.music, 7, name)
	com(s.music, 11, who)
	com(s.display, 17)
	runtime.KeepAlive(s.delegate)
}
func (s *mediaSession) close() {
	if s == nil {
		return
	}
	if s.controls != 0 {
		if s.token != 0 {
			com(s.controls, 33, uintptr(s.token))
		}
		com(s.controls, 11, 0)
	}
	release(s.music)
	release(s.display)
	release(s.controls)
	s.music = 0
	s.display = 0
	s.controls = 0
	if s.apartment && s.thread == windows.GetCurrentThreadId() {
		roUninitialize.Call()
		s.apartment = false
	}
	runtime.KeepAlive(s.delegate)
}
