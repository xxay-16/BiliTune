package store

import (
	"encoding/json"
	"github.com/xxay-16/BiliTune/internal/model"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLegacyImportAndEncryptedSession(t *testing.T) {
	root := t.TempDir()
	track := model.Track{ID: "BV1234567890", Title: "测试歌", DurationMS: 123000}
	serialize := func(v any) string { b, _ := json.Marshal(v); return string(b) }
	prefs := map[string]any{"flutter.settings.theme_mode": "light", "settings.playback": serialize(map[string]any{"audioQuality": "lossless", "lyricsSourcePreference": "biliOnly", "playbackSpeed": 1.25, "historyLimit": 50}), "playback.history": serialize([]model.Track{track}), "playback.state": serialize(map[string]any{"track": track, "queue": []model.Track{track}, "positionMs": 43000, "volume": .4, "repeat": "all"}), "bili_credential": serialize(map[string]string{"cookieHeader": "SESSDATA=test-secret; bili_jct=csrf"}), "downloads.tasks": serialize([]map[string]any{{"id": "task1", "bvid": track.ID, "title": track.Title, "status": "completed", "savePath": filepath.Join(root, "done.m4a")}, {"id": "task2", "status": "downloading", "downloadedBytes": 123}})}
	original, _ := json.Marshal(prefs)
	legacy := filepath.Join(root, "shared_preferences.json")
	_ = os.WriteFile(legacy, original, 0600)
	st, state, err := Open(filepath.Join(root, "desktop"))
	if err != nil {
		t.Fatal(err)
	}
	if !st.Migrated || state.Settings.Theme != "light" || state.Settings.LyricsSource != "biliOnly" || state.Settings.PlaybackRate != 1.25 || len(state.History) != 1 || state.Player.Position != 43 || state.Player.QueueCount != 1 || len(state.Downloads) != 2 || state.Downloads[1].Status != "paused" {
		t.Fatalf("migration failed: %+v", state)
	}
	cookie, err := st.Cookie()
	if err != nil || !strings.Contains(cookie, "test-secret") {
		t.Fatal(err)
	}
	disk, _ := os.ReadFile(filepath.Join(st.Dir, "state.json"))
	if strings.Contains(string(disk), "test-secret") {
		t.Fatal("session in state")
	}
	unchanged, _ := os.ReadFile(legacy)
	if string(unchanged) != string(original) {
		t.Fatal("legacy modified")
	}
	_, again, err := Open(st.Dir)
	if err != nil || again.Player.Playing || again.Player.Position != 43 {
		t.Fatal(err, again.Player)
	}
}
func TestCorruptStatePreserved(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	_ = os.WriteFile(path, []byte("{broken"), 0600)
	_, _, err := Open(dir)
	if err == nil {
		t.Fatal("expected error")
	}
	data, _ := os.ReadFile(path)
	if string(data) != "{broken" {
		t.Fatal("overwrote corrupt file")
	}
}
func TestSaveAtomicRoundtrip(t *testing.T) {
	st, state, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	state.Settings.Theme = "light"
	for i := 0; i < 5; i++ {
		if err = st.Save(state); err != nil {
			t.Fatal(err)
		}
	}
	_, got, err := Open(st.Dir)
	if err != nil || got.Settings.Theme != "light" {
		t.Fatal(err)
	}
	files, _ := filepath.Glob(filepath.Join(st.Dir, ".write-*"))
	if len(files) != 0 {
		t.Fatal(files)
	}
}

func TestAppearanceUpgradeKeepsLaterThemeChoice(t *testing.T) {
	dir := t.TempDir()
	settings := model.DefaultSettings()
	settings.Theme = "dark"
	original := map[string]any{"settings": settings, "history": []model.Track{{ID: "saved", Title: "保留的音乐"}}, "player": model.Player{Volume: .42, Position: 19}}
	data, _ := json.Marshal(original)
	if err := os.WriteFile(filepath.Join(dir, "state.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	st, state, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if state.AppearanceVersion != 1 || state.Settings.Theme != "light" || len(state.History) != 1 || state.Player.Volume != .42 || state.Player.Position != 19 {
		t.Fatalf("appearance migration changed data: %+v", state)
	}
	state.Settings.Theme = "dark"
	if err = st.Save(state); err != nil {
		t.Fatal(err)
	}
	_, again, err := Open(dir)
	if err != nil || again.Settings.Theme != "dark" {
		t.Fatal("explicit theme choice lost after restart", err)
	}
}
