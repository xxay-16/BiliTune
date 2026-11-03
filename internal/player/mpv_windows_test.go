//go:build windows

package player

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMPVRealPlayback(t *testing.T) {
	if os.Getenv("BILITUNE_TEST_MPV") == "" {
		t.Skip("set BILITUNE_TEST_MPV=1 to exercise the bundled runtime")
	}
	engine, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	if err = engine.(*MPV).property("ao", "null"); err != nil {
		t.Fatal(err)
	}
	if err = engine.SetVolume(0); err != nil {
		t.Fatal(err)
	}
	// Ten seconds of silence: verifies decoding and seeking without audible output.
	data := make([]byte, 44+16000*10*2)
	copy(data, "RIFF")
	binary.LittleEndian.PutUint32(data[4:], uint32(len(data)-8))
	copy(data[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(data[16:], 16)
	binary.LittleEndian.PutUint16(data[20:], 1)
	binary.LittleEndian.PutUint16(data[22:], 1)
	binary.LittleEndian.PutUint32(data[24:], 16000)
	binary.LittleEndian.PutUint32(data[28:], 32000)
	binary.LittleEndian.PutUint16(data[32:], 2)
	binary.LittleEndian.PutUint16(data[34:], 16)
	copy(data[36:], "data")
	binary.LittleEndian.PutUint32(data[40:], uint32(len(data)-44))
	path := filepath.Join(t.TempDir(), "silence.wav")
	if err = os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	if err = engine.Load(path, nil, 3); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		snap := engine.Snapshot()
		if snap.Error != "" {
			t.Fatal(snap.Error)
		}
		if snap.Duration > 9 && snap.Position >= 3 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("did not load: %+v", snap)
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err = engine.Pause(); err != nil {
		t.Fatal(err)
	}
	if engine.Snapshot().Playing {
		t.Fatal("pause failed")
	}
	if err = engine.Seek(5); err != nil {
		t.Fatal(err)
	}
	if err = engine.Stop(); err != nil {
		t.Fatal(err)
	}
	if err = engine.Load(filepath.Join(t.TempDir(), "missing.wav"), nil, 0); err != nil {
		t.Fatal(err)
	}
	deadline = time.Now().Add(3 * time.Second)
	for engine.Snapshot().Error == "" {
		if time.Now().After(deadline) {
			t.Fatal("missing async error")
		}
		time.Sleep(50 * time.Millisecond)
	}
}
