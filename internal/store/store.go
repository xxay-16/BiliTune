package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/xxay-16/BiliTune/internal/model"
)

type Store struct {
	mu       sync.Mutex
	Dir      string
	Migrated bool
}

func Open(dir string) (*Store, model.State, error) {
	if dir == "" {
		root, err := os.UserConfigDir()
		if err != nil {
			return nil, model.State{}, err
		}
		dir = filepath.Join(root, "com.wep56", "bilitune", "desktop")
	}
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, model.State{}, err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, model.State{}, err
	}
	s := &Store{Dir: dir}
	state := model.State{Settings: model.DefaultSettings(), Player: model.Player{Volume: .7, Repeat: "off"}, History: []model.Track{}, Downloads: []model.Download{}, Queue: []model.Track{}, SearchHistory: []string{}, HiddenFolders: []int64{}}
	data, err := os.ReadFile(filepath.Join(dir, "state.json"))
	if err == nil {
		if err := json.Unmarshal(data, &state); err != nil {
			return nil, state, fmt.Errorf("读取设置失败，请保留 state.json 以便恢复：%w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, state, err
	} else {
		legacy := filepath.Join(filepath.Dir(dir), "shared_preferences.json")
		if err := s.importLegacy(legacy, &state); err != nil {
			return nil, state, err
		}
		if err := s.Save(state); err != nil {
			return nil, state, err
		}
	}
	for i := range state.Downloads {
		if state.Downloads[i].Status == "downloading" || state.Downloads[i].Status == "queued" {
			state.Downloads[i].Status = "paused"
		}
	}
	state.Player.Playing = false
	state.Player.Buffering = false
	state.Player.Error = ""
	state.Player.QueueCount = len(state.Queue)
	if state.Player.Track != nil {
		state.Player.Duration = float64(state.Player.Track.DurationMS) / 1000
	}
	if state.Settings.Hotkeys == nil {
		state.Settings.Hotkeys = []model.Hotkey{}
	}
	return s, state, nil
}

func (s *Store) atomic(name string, data []byte) error {
	path := filepath.Join(s.Dir, name)
	f, err := os.CreateTemp(s.Dir, ".write-*")
	if err != nil {
		return err
	}
	temp := f.Name()
	defer os.Remove(temp)
	if err := f.Chmod(0600); err != nil {
		f.Close()
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(temp, path)
}

func (s *Store) Save(state model.State) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return s.atomic("state.json", data)
}
func (s *Store) SaveCookie(cookie string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := protect([]byte(cookie))
	if err != nil {
		return err
	}
	return s.atomic("session.dat", data)
}
func (s *Store) Cookie() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(filepath.Join(s.Dir, "session.dat"))
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	plain, err := unprotect(data)
	return string(plain), err
}

func decodeLegacy(data map[string]json.RawMessage, key string, dest any) bool {
	raw, ok := data[key]
	if !ok {
		raw, ok = data["flutter."+key]
	}
	if !ok {
		return false
	}
	var text string
	if json.Unmarshal(raw, &text) == nil {
		if json.Unmarshal([]byte(text), dest) == nil {
			return true
		}
	}
	return json.Unmarshal(raw, dest) == nil
}

func (s *Store) importLegacy(path string, state *model.State) error {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var prefs map[string]json.RawMessage
	if err = json.Unmarshal(data, &prefs); err != nil {
		return fmt.Errorf("旧版设置无法读取，未修改原文件：%w", err)
	}
	decodeLegacy(prefs, "playback.history", &state.History)
	decodeLegacy(prefs, "search.history", &state.SearchHistory)
	decodeLegacy(prefs, "library.hidden_folder_ids", &state.HiddenFolders)
	decodeLegacy(prefs, "settings.windows_hotkeys", &state.Settings.Hotkeys)
	decodeLegacy(prefs, "settings.theme_mode", &state.Settings.Theme)
	decodeLegacy(prefs, "settings.window_close_behavior", &state.Settings.CloseBehavior)
	var playback struct {
		AudioQuality  string  `json:"audioQuality"`
		LyricsSource  string  `json:"lyricsSourcePreference"`
		PlaybackSpeed float64 `json:"playbackSpeed"`
		HistoryLimit  int     `json:"historyLimit"`
	}
	if decodeLegacy(prefs, "settings.playback", &playback) {
		if playback.AudioQuality != "" {
			state.Settings.AudioQuality = playback.AudioQuality
		}
		if playback.LyricsSource != "" {
			state.Settings.LyricsSource = playback.LyricsSource
		}
		if playback.PlaybackSpeed > 0 {
			state.Settings.PlaybackRate = playback.PlaybackSpeed
		}
		if playback.HistoryLimit > 0 {
			state.Settings.HistoryLimit = playback.HistoryLimit
		}
	}
	var download struct {
		DirectoryPath string `json:"directoryPath"`
		MaxConcurrent int    `json:"maxConcurrent"`
	}
	if decodeLegacy(prefs, "settings.download", &download) {
		state.Settings.DownloadDir = download.DirectoryPath
		if download.MaxConcurrent > 0 {
			state.Settings.DownloadConcurrency = download.MaxConcurrent
		}
	}
	var player struct {
		Track      *model.Track  `json:"track"`
		Queue      []model.Track `json:"queue"`
		PositionMS int64         `json:"positionMs"`
		Volume     float64       `json:"volume"`
		Shuffle    bool          `json:"shuffle"`
		Repeat     string        `json:"repeat"`
	}
	if decodeLegacy(prefs, "playback.state", &player) {
		state.Queue = player.Queue
		state.Player.Track = player.Track
		state.Player.Position = float64(player.PositionMS) / 1000
		state.Player.Volume = player.Volume
		state.Player.Shuffle = player.Shuffle
		state.Player.Repeat = player.Repeat
	}
	var credential struct {
		CookieHeader string `json:"cookieHeader"`
		Sessdata     string `json:"sessdata"`
		BiliJct      string `json:"biliJct"`
		DedeUserID   string `json:"dedeUserId"`
	}
	if decodeLegacy(prefs, "bili_credential", &credential) {
		cookie := credential.CookieHeader
		if cookie == "" && credential.Sessdata != "" {
			cookie = "SESSDATA=" + credential.Sessdata + "; bili_jct=" + credential.BiliJct + "; DedeUserID=" + credential.DedeUserID
		}
		if cookie != "" {
			if err := s.SaveCookie(cookie); err != nil {
				return err
			}
		}
	}
	var downloads []struct {
		ID              string `json:"id"`
		Title           string `json:"title"`
		Artist          string `json:"artist"`
		Type            string `json:"type"`
		BVID            string `json:"bvid"`
		AID             int64  `json:"aid"`
		CID             int64  `json:"cid"`
		AudioID         int64  `json:"audioId"`
		SavePath        string `json:"savePath"`
		CoverURL        string `json:"coverUrl"`
		Status          string `json:"status"`
		DownloadedBytes int64  `json:"downloadedBytes"`
		TotalBytes      int64  `json:"totalBytes"`
		ErrorMessage    string `json:"errorMessage"`
	}
	if decodeLegacy(prefs, "downloads.tasks", &downloads) {
		for _, d := range downloads {
			status := d.Status
			if status == "queued" || status == "downloading" || status == "cancelled" {
				status = "paused"
			}
			track := model.Track{ID: d.ID, Title: d.Title, Artist: d.Artist, Type: d.Type, BVID: d.BVID, AID: d.AID, CID: d.CID, AudioID: d.AudioID, CoverURL: d.CoverURL}
			if d.BVID != "" {
				track.ID = d.BVID
			}
			if status == "completed" {
				track.LocalPath = d.SavePath
			}
			state.Downloads = append(state.Downloads, model.Download{ID: d.ID, Track: track, Status: status, Path: d.SavePath, Received: d.DownloadedBytes, Total: d.TotalBytes, Error: d.ErrorMessage})
		}
	}
	s.Migrated = true
	return nil
}
