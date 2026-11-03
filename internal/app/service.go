package app

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/xxay-16/BiliTune/internal/bili"
	"github.com/xxay-16/BiliTune/internal/model"
	"github.com/xxay-16/BiliTune/internal/player"
	"github.com/xxay-16/BiliTune/internal/store"
)

type Desktop struct {
	Window    func(string) error
	Directory func() (string, error)
	Files     func() ([]string, error)
	Reveal    func(string) error
	OpenURL   func(string) error
	Configure func(model.Settings, bool) map[string]string
	Record    func(bool)
	Media     func(model.Player)
}

type Service struct {
	mu              sync.RWMutex
	state           model.State
	store           *store.Store
	client          *bili.Client
	engine          player.Engine
	engineError     string
	account         *model.Account
	desktop         Desktop
	hotkeyErrors    map[string]string
	ctx             context.Context
	cancel          context.CancelFunc
	loadCancel      context.CancelFunc
	generation      uint64
	loaded          bool
	loading         bool
	engineMu        sync.Mutex
	favoriteMu      sync.Mutex
	downloadMu      sync.Mutex
	downloadCancels map[string]context.CancelFunc
	downloadWorkers map[string]bool
	closed          sync.Once
	workers         sync.WaitGroup
	loopWorkers     sync.WaitGroup
	started         sync.Once
	persistMu       sync.Mutex
}

func New(st *store.Store, state model.State, client *bili.Client, engine player.Engine, engineErr string) *Service {
	ctx, cancel := context.WithCancel(context.Background())
	return &Service{store: st, state: state, client: client, engine: engine, engineError: engineErr, ctx: ctx, cancel: cancel, hotkeyErrors: map[string]string{}, downloadCancels: map[string]context.CancelFunc{}, downloadWorkers: map[string]bool{}}
}
func (s *Service) SetDesktop(desktop Desktop) { s.mu.Lock(); s.desktop = desktop; s.mu.Unlock() }
func (s *Service) Start() {
	s.started.Do(func() { s.loopWorkers.Add(1); go func() { defer s.loopWorkers.Done(); s.loop() }() })
}
func (s *Service) Close() {
	s.closed.Do(func() {
		s.cancel()
		s.engineMu.Lock()
		s.mu.Lock()
		if s.loadCancel != nil {
			s.loadCancel()
		}
		s.mu.Unlock()
		s.engineMu.Unlock()
		s.downloadMu.Lock()
		for _, cancel := range s.downloadCancels {
			cancel()
		}
		s.downloadMu.Unlock()
		s.workers.Wait()
		s.loopWorkers.Wait()
		s.persist()
		if s.engine != nil {
			s.engine.Close()
		}
	})
}
func (s *Service) snapshot() model.State {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state := s.state
	state.Queue = append([]model.Track{}, s.state.Queue...)
	state.History = append([]model.Track{}, s.state.History...)
	state.Downloads = append([]model.Download{}, s.state.Downloads...)
	state.HiddenFolders = append([]int64{}, s.state.HiddenFolders...)
	state.SearchHistory = append([]string{}, s.state.SearchHistory...)
	state.Settings.Hotkeys = append([]model.Hotkey{}, s.state.Settings.Hotkeys...)
	return state
}
func (s *Service) persist() {
	s.persistMu.Lock()
	defer s.persistMu.Unlock()
	if err := s.store.Save(s.snapshot()); err != nil {
		s.setError("保存本地数据失败：" + err.Error())
	}
}
func (s *Service) setError(message string) {
	s.mu.Lock()
	s.state.Player.Error = message
	s.mu.Unlock()
}
func (s *Service) Player() model.Player { s.mu.RLock(); defer s.mu.RUnlock(); return s.state.Player }
func (s *Service) Queue() []model.Track { return s.snapshot().Queue }
func (s *Service) Bootstrap() model.Bootstrap {
	state := s.snapshot()
	s.mu.RLock()
	account := s.account
	errors := make(map[string]string, len(s.hotkeyErrors))
	for key, value := range s.hotkeyErrors {
		errors[key] = value
	}
	s.mu.RUnlock()
	return model.Bootstrap{Version: model.Version, Settings: state.Settings, Account: account, Player: state.Player, History: state.History, SearchHistory: state.SearchHistory, Migrated: s.store.Migrated, EngineReady: s.engine != nil, EngineError: s.engineError, HotkeyErrors: errors}
}
func (s *Service) Account(ctx context.Context) (*model.Account, error) {
	if s.client.Cookie() == "" {
		return nil, nil
	}
	account, err := s.client.Account(ctx)
	if err == nil {
		s.mu.Lock()
		s.account = account
		s.mu.Unlock()
	}
	return account, err
}
func (s *Service) LoginCookie(ctx context.Context, cookie string) (*model.Account, error) {
	cookie = strings.TrimSpace(cookie)
	if !strings.Contains(cookie, "SESSDATA=") {
		return nil, fmt.Errorf("请输入包含 SESSDATA 的完整 Cookie")
	}
	previous := s.client.Cookie()
	s.client.SetCookie(cookie)
	account, err := s.Account(ctx)
	if err != nil || account == nil {
		s.client.SetCookie(previous)
		if err == nil {
			err = fmt.Errorf("Cookie 无效或已过期")
		}
		return nil, err
	}
	if err = s.store.SaveCookie(cookie); err != nil {
		return nil, err
	}
	return account, nil
}
func (s *Service) PollQR(ctx context.Context, key string) (map[string]any, error) {
	data, err := s.client.PollQR(ctx, key)
	if err != nil {
		return nil, err
	}
	if bili.Int(data["code"]) == 0 {
		if err = s.store.SaveCookie(s.client.Cookie()); err != nil {
			return nil, err
		}
		account, err := s.Account(ctx)
		if err != nil {
			return nil, err
		}
		data = map[string]any{"code": 0, "account": account}
	}
	return data, nil
}
func (s *Service) Logout() error {
	s.client.SetCookie("")
	s.mu.Lock()
	s.account = nil
	s.mu.Unlock()
	return s.store.SaveCookie("")
}
func (s *Service) Folders(ctx context.Context) ([]model.Folder, error) {
	s.mu.RLock()
	account := s.account
	s.mu.RUnlock()
	if account == nil {
		var err error
		account, err = s.Account(ctx)
		if err != nil {
			return nil, err
		}
	}
	if account == nil {
		return []model.Folder{}, nil
	}
	folders, err := s.client.Folders(ctx, account.MID)
	if err != nil {
		return nil, err
	}
	hidden := s.snapshot().HiddenFolders
	result := []model.Folder{}
	for _, folder := range folders {
		found := false
		for _, id := range hidden {
			if folder.ID == id {
				found = true
				break
			}
		}
		if !found {
			result = append(result, folder)
		}
	}
	return result, nil
}
func (s *Service) HideFolder(id int64, hidden bool) {
	s.mu.Lock()
	next := []int64{}
	for _, item := range s.state.HiddenFolders {
		if item != id {
			next = append(next, item)
		}
	}
	if hidden {
		next = append(next, id)
	}
	s.state.HiddenFolders = next
	s.mu.Unlock()
	s.persist()
}
func (s *Service) Favorite(ctx context.Context, track model.Track, from, to int64) error {
	s.favoriteMu.Lock()
	defer s.favoriteMu.Unlock()
	if from == 0 && to == 0 {
		return fmt.Errorf("请选择收藏夹")
	}
	return s.client.Favorite(ctx, track, from, to)
}
func (s *Service) Search(ctx context.Context, query string, page int, music bool) ([]model.Track, error) {
	if page < 1 {
		page = 1
	}
	tracks, err := s.client.Search(ctx, query, page, music)
	if err == nil && page == 1 && strings.TrimSpace(query) != "" {
		s.mu.Lock()
		history := []string{strings.TrimSpace(query)}
		for _, item := range s.state.SearchHistory {
			if item != query && len(history) < 20 {
				history = append(history, item)
			}
		}
		s.state.SearchHistory = history
		s.mu.Unlock()
		s.persist()
	}
	return tracks, err
}
func (s *Service) SetSettings(settings model.Settings) error {
	if settings.Theme != "dark" && settings.Theme != "light" && settings.Theme != "system" {
		return fmt.Errorf("无效的主题")
	}
	if settings.PlaybackRate < .5 || settings.PlaybackRate > 2 {
		return fmt.Errorf("播放速度应为 0.5 到 2 倍")
	}
	if settings.HistoryLimit < 20 || settings.HistoryLimit > 500 {
		return fmt.Errorf("历史记录数量应为 20 到 500")
	}
	if settings.DownloadConcurrency < 1 || settings.DownloadConcurrency > 4 {
		return fmt.Errorf("同时下载数量应为 1 到 4")
	}
	if settings.CloseBehavior != "tray" && settings.CloseBehavior != "minimize" && settings.CloseBehavior != "exit" {
		return fmt.Errorf("无效的关闭行为")
	}
	if settings.DownloadDir != "" && !filepath.IsAbs(settings.DownloadDir) {
		return fmt.Errorf("请选择绝对路径的下载目录")
	}
	s.mu.Lock()
	old := s.state.Settings
	s.state.Settings = settings
	if len(s.state.History) > settings.HistoryLimit {
		s.state.History = s.state.History[:settings.HistoryLimit]
	}
	desktop := s.desktop
	s.mu.Unlock()
	if s.engine != nil {
		_ = s.engine.SetRate(settings.PlaybackRate)
	}
	if desktop.Configure != nil {
		errors := desktop.Configure(settings, old.AutoStart != settings.AutoStart)
		s.mu.Lock()
		s.hotkeyErrors = errors
		s.mu.Unlock()
	}
	s.persist()
	return nil
}
func (s *Service) SetHotkeyErrors(values map[string]string) {
	s.mu.Lock()
	s.hotkeyErrors = values
	s.mu.Unlock()
}
func (s *Service) SetAutoStartInitial(enabled bool) {
	s.mu.Lock()
	s.state.Settings.AutoStart = enabled
	s.mu.Unlock()
}
func (s *Service) RecordHotkeys(record bool) {
	s.mu.RLock()
	fn := s.desktop.Record
	s.mu.RUnlock()
	if fn != nil {
		fn(record)
	}
}

func (s *Service) Play(track model.Track, queue []model.Track) error {
	return s.playFrom(track, queue, 0)
}
func (s *Service) playFrom(track model.Track, queue []model.Track, start float64) error {
	if s.engine == nil {
		return fmt.Errorf("%s", s.engineError)
	}
	if track.ID == "" {
		return fmt.Errorf("歌曲编号为空")
	}
	s.engineMu.Lock()
	s.mu.Lock()
	if s.ctx.Err() != nil {
		s.mu.Unlock()
		s.engineMu.Unlock()
		return fmt.Errorf("程序正在退出")
	}
	if s.loadCancel != nil {
		s.loadCancel()
	}
	ctx, cancel := context.WithCancel(s.ctx)
	s.loadCancel = cancel
	s.generation++
	gen := s.generation
	s.loaded = false
	s.loading = true
	if len(queue) > 0 {
		s.state.Queue = append([]model.Track{}, queue...)
		s.state.Player.QueueVersion++
	}
	if len(s.state.Queue) == 0 {
		s.state.Queue = []model.Track{track}
		s.state.Player.QueueVersion++
	}
	initialTrack := track
	s.state.Player.Track = &initialTrack
	s.state.Player.Position = start
	s.state.Player.Duration = float64(track.DurationMS) / 1000
	s.state.Player.Buffering = true
	s.state.Player.Playing = true
	s.state.Player.Error = ""
	s.state.Player.QueueCount = len(s.state.Queue)
	settings := s.state.Settings
	s.mu.Unlock()
	_ = s.engine.Pause()
	s.workers.Add(1)
	s.engineMu.Unlock()
	go func() {
		defer s.workers.Done()
		var source model.Source
		var err error
		if track.LocalPath != "" {
			source = model.Source{URL: track.LocalPath, Quality: "本地音频"}
		} else if strings.HasPrefix(track.SourceURL, "file:") {
			u, e := url.Parse(track.SourceURL)
			if e == nil {
				source = model.Source{URL: strings.TrimPrefix(u.Path, "/"), Quality: "本地音频"}
			} else {
				err = e
			}
		} else {
			track, source, err = s.client.Resolve(ctx, track, settings.AudioQuality)
		}
		if ctx.Err() != nil {
			return
		}
		if err == nil {
			s.engineMu.Lock()
			defer s.engineMu.Unlock()
			s.mu.RLock()
			current := gen == s.generation
			s.mu.RUnlock()
			if !current {
				return
			}
			headers := map[string]string{"Referer": bili.Referer, "User-Agent": bili.UserAgent}
			err = s.engine.Load(source.URL, headers, start)
			if err != nil {
				for _, backup := range source.Backups {
					if ctx.Err() != nil {
						break
					}
					if err = s.engine.Load(backup, headers, start); err == nil {
						break
					}
				}
			}
			if err == nil {
				_ = s.engine.SetRate(settings.PlaybackRate)
				s.mu.RLock()
				volume := s.state.Player.Volume
				playing := s.state.Player.Playing
				s.mu.RUnlock()
				_ = s.engine.SetVolume(volume)
				if !playing {
					_ = s.engine.Pause()
				}
			}
		}
		s.mu.Lock()
		if gen != s.generation {
			s.mu.Unlock()
			return
		}
		s.loading = false
		s.state.Player.Buffering = false
		if err != nil {
			s.state.Player.Error = err.Error()
			s.state.Player.Playing = false
			s.mu.Unlock()
			return
		}
		s.loaded = true
		s.state.Player.Track = &track
		s.state.Player.Quality = source.Quality
		history := []model.Track{track}
		for _, item := range s.state.History {
			if item.ID != track.ID && len(history) < settings.HistoryLimit {
				history = append(history, item)
			}
		}
		s.state.History = history
		s.mu.Unlock()
		s.persist()
	}()
	return nil
}

func (s *Service) Command(action string, value float64) error {
	s.mu.RLock()
	p := s.state.Player
	queue := append([]model.Track{}, s.state.Queue...)
	loaded := s.loaded
	loading := s.loading
	settings := s.state.Settings
	desktop := s.desktop
	s.mu.RUnlock()
	if action == "show" || action == "hide" || action == "exit" || action == "toggleWindow" {
		if desktop.Window != nil {
			return desktop.Window(action)
		}
		return nil
	}
	if s.engine == nil {
		return fmt.Errorf("%s", s.engineError)
	}
	switch action {
	case "playPause":
		if p.Playing {
			return s.Command("pause", 0)
		}
		return s.Command("play", 0)
	case "play":
		if p.Track == nil {
			return nil
		}
		s.engineMu.Lock()
		s.mu.Lock()
		loaded, loading = s.loaded, s.loading
		if loading {
			s.state.Player.Playing = true
			s.mu.Unlock()
			s.engineMu.Unlock()
			return nil
		}
		s.state.Player.Playing = true
		s.mu.Unlock()
		if !loaded {
			s.engineMu.Unlock()
			return s.playFrom(*p.Track, queue, p.Position)
		}
		err := s.engine.Play()
		s.engineMu.Unlock()
		return err
	case "pause":
		s.engineMu.Lock()
		defer s.engineMu.Unlock()
		s.mu.Lock()
		s.state.Player.Playing = false
		s.mu.Unlock()
		return s.engine.Pause()
	case "stop":
		s.engineMu.Lock()
		defer s.engineMu.Unlock()
		s.mu.Lock()
		if s.loadCancel != nil {
			s.loadCancel()
		}
		s.generation++
		s.loading = false
		s.loaded = false
		s.state.Player.Playing = false
		s.state.Player.Buffering = false
		s.state.Player.Position = 0
		s.mu.Unlock()
		return s.engine.Stop()
	case "nextTrack", "previousTrack":
		if len(queue) == 0 {
			return nil
		}
		index := -1
		for i, item := range queue {
			if p.Track != nil && item.ID == p.Track.ID {
				index = i
				break
			}
		}
		if index < 0 {
			return s.Play(queue[0], queue)
		}
		next := index + 1
		if action == "previousTrack" {
			next = (index - 1 + len(queue)) % len(queue)
		} else {
			if p.Shuffle && len(queue) > 1 {
				next = rand.IntN(len(queue) - 1)
				if next >= index {
					next++
				}
			} else if next >= len(queue) {
				if p.Repeat == "off" {
					return s.Command("pause", 0)
				}
				next = 0
			}
		}
		return s.Play(queue[next], queue)
	case "seek":
		if value < 0 {
			value = 0
		}
		if p.Duration > 0 && value > p.Duration {
			value = p.Duration
		}
		s.mu.Lock()
		s.state.Player.Position = value
		s.mu.Unlock()
		if loaded {
			return s.engine.Seek(value)
		}
	case "volume":
		if value < 0 {
			value = 0
		}
		if value > 1 {
			value = 1
		}
		if err := s.engine.SetVolume(value); err != nil {
			return err
		}
		s.mu.Lock()
		s.state.Player.Volume = value
		s.mu.Unlock()
		s.persist()
	case "rate":
		settings.PlaybackRate = value
		return s.SetSettings(settings)
	case "shuffle":
		s.mu.Lock()
		s.state.Player.Shuffle = !s.state.Player.Shuffle
		s.mu.Unlock()
		s.persist()
	case "repeat":
		modes := []string{"off", "all", "one"}
		index := 0
		for i, mode := range modes {
			if mode == p.Repeat {
				index = (i + 1) % len(modes)
			}
		}
		s.mu.Lock()
		s.state.Player.Repeat = modes[index]
		s.mu.Unlock()
		s.persist()
	default:
		return fmt.Errorf("未知播放操作：%s", action)
	}
	return nil
}

func (s *Service) QueueAction(action string, track model.Track, id string) error {
	s.mu.Lock()
	queue := s.state.Queue
	switch action {
	case "clear":
		if s.state.Player.Track != nil {
			queue = []model.Track{*s.state.Player.Track}
		} else {
			queue = []model.Track{}
		}
	case "remove":
		next := []model.Track{}
		for _, item := range queue {
			if item.ID != id {
				next = append(next, item)
			}
		}
		queue = next
	case "append":
		queue = append(queue, track)
	case "next":
		index := len(queue)
		for i, item := range queue {
			if s.state.Player.Track != nil && item.ID == s.state.Player.Track.ID {
				index = i + 1
				break
			}
		}
		queue = append(queue, model.Track{})
		copy(queue[index+1:], queue[index:])
		queue[index] = track
	default:
		s.mu.Unlock()
		return fmt.Errorf("无效的队列操作")
	}
	s.state.Queue = queue
	s.state.Player.QueueVersion++
	s.state.Player.QueueCount = len(queue)
	s.mu.Unlock()
	s.persist()
	return nil
}

func (s *Service) loop() {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	ticks := 0
	ended := false
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			ticks++
			s.mu.RLock()
			loaded := s.loaded
			loading := s.loading
			s.mu.RUnlock()
			if loaded && !loading && s.engine != nil {
				s.engineMu.Lock()
				snapshot := s.engine.Snapshot()
				s.mu.Lock()
				if !s.loaded || s.loading {
					s.mu.Unlock()
					s.engineMu.Unlock()
					continue
				}
				s.state.Player.Playing = snapshot.Playing
				s.state.Player.Position = snapshot.Position
				if snapshot.Duration > 0 {
					s.state.Player.Duration = snapshot.Duration
				}
				s.state.Player.Buffered = snapshot.Buffered
				s.state.Player.Buffering = snapshot.Buffering
				if snapshot.Error != "" {
					s.state.Player.Error = snapshot.Error
					s.state.Player.Playing = false
					s.state.Player.Buffering = false
					s.loaded = false
				}
				p := s.state.Player
				s.mu.Unlock()
				s.engineMu.Unlock()
				if snapshot.Ended && !ended {
					ended = true
					if p.Repeat == "one" {
						_ = s.engine.Seek(0)
						_ = s.engine.Play()
					} else {
						_ = s.Command("nextTrack", 0)
					}
				} else if !snapshot.Ended {
					ended = false
				}
			}
			s.mu.RLock()
			media := s.desktop.Media
			p := s.state.Player
			s.mu.RUnlock()
			if media != nil {
				media(p)
			}
			if ticks%5 == 0 {
				s.scheduleDownloads()
			}
			if ticks%25 == 0 {
				s.persist()
			}
		}
	}
}

func (s *Service) Window(action string) error {
	s.mu.RLock()
	fn := s.desktop.Window
	s.mu.RUnlock()
	if fn == nil {
		return fmt.Errorf("此操作仅在桌面窗口中可用")
	}
	return fn(action)
}
func (s *Service) ChooseDirectory() (string, error) {
	s.mu.RLock()
	fn := s.desktop.Directory
	s.mu.RUnlock()
	if fn == nil {
		return "", fmt.Errorf("此操作仅在桌面窗口中可用")
	}
	return fn()
}
func (s *Service) LocalFiles() ([]model.Track, error) {
	s.mu.RLock()
	fn := s.desktop.Files
	s.mu.RUnlock()
	if fn == nil {
		return nil, fmt.Errorf("此操作仅在桌面窗口中可用")
	}
	paths, err := fn()
	if err != nil {
		return nil, err
	}
	tracks := []model.Track{}
	for _, path := range paths {
		ext := strings.ToLower(filepath.Ext(path))
		if !strings.Contains("|.mp3|.m4a|.mp4|.flac|.wav|.ogg|.opus|.aac|.wma|", "|"+ext+"|") || ext == "" {
			continue
		}
		if _, err := os.Stat(path); err != nil {
			continue
		}
		tracks = append(tracks, model.Track{ID: "local:" + path, Title: strings.TrimSuffix(filepath.Base(path), ext), Artist: "本地音频", Type: "audio", LocalPath: path})
	}
	return tracks, nil
}
