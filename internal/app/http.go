package app

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/xxay-16/BiliTune/internal/bili"
	"github.com/xxay-16/BiliTune/internal/model"
)

func (s *Service) Handler() http.Handler {
	mux := http.NewServeMux()
	jsonRoute := func(pattern string, handle func(http.ResponseWriter, *http.Request) (any, error)) {
		mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			if origin := r.Header.Get("Origin"); origin != "" {
				u, err := url.Parse(origin)
				if err != nil || !strings.EqualFold(u.Host, r.Host) {
					w.WriteHeader(http.StatusForbidden)
					_ = json.NewEncoder(w).Encode(map[string]string{"error": "请求来源无效"})
					return
				}
			}
			result, err := handle(w, r)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
			if result == nil {
				result = map[string]bool{"ok": true}
			}
			_ = json.NewEncoder(w).Encode(result)
		})
	}
	decode := func(r *http.Request, dest any) error {
		decoder := json.NewDecoder(io.LimitReader(r.Body, 32<<20))
		if err := decoder.Decode(dest); err != nil {
			return fmt.Errorf("请求数据无效：%w", err)
		}
		return nil
	}
	id := func(r *http.Request) int64 { v, _ := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64); return v }
	jsonRoute("GET /api/bootstrap", func(w http.ResponseWriter, r *http.Request) (any, error) { return s.Bootstrap(), nil })
	jsonRoute("GET /api/player", func(w http.ResponseWriter, r *http.Request) (any, error) { return s.Player(), nil })
	jsonRoute("GET /api/queue", func(w http.ResponseWriter, r *http.Request) (any, error) { return s.Queue(), nil })
	jsonRoute("GET /api/account", func(w http.ResponseWriter, r *http.Request) (any, error) {
		account, err := s.Account(r.Context())
		return map[string]any{"account": account}, err
	})
	jsonRoute("GET /api/qr", func(w http.ResponseWriter, r *http.Request) (any, error) { return s.client.QR(r.Context()) })
	jsonRoute("GET /api/qr/poll", func(w http.ResponseWriter, r *http.Request) (any, error) {
		return s.PollQR(r.Context(), r.URL.Query().Get("key"))
	})
	jsonRoute("POST /api/login", func(w http.ResponseWriter, r *http.Request) (any, error) {
		var body struct {
			Cookie string `json:"cookie"`
		}
		if err := decode(r, &body); err != nil {
			return nil, err
		}
		return s.LoginCookie(r.Context(), body.Cookie)
	})
	jsonRoute("POST /api/logout", func(w http.ResponseWriter, r *http.Request) (any, error) { return nil, s.Logout() })
	jsonRoute("GET /api/folders", func(w http.ResponseWriter, r *http.Request) (any, error) { return s.Folders(r.Context()) })
	jsonRoute("GET /api/folder/tracks", func(w http.ResponseWriter, r *http.Request) (any, error) {
		return s.client.FolderTracks(r.Context(), id(r), r.URL.Query().Get("keyword"))
	})
	jsonRoute("POST /api/folder", func(w http.ResponseWriter, r *http.Request) (any, error) {
		var body struct {
			Title   string `json:"title"`
			Intro   string `json:"intro"`
			Private bool   `json:"private"`
		}
		if err := decode(r, &body); err != nil {
			return nil, err
		}
		if strings.TrimSpace(body.Title) == "" {
			return nil, fmt.Errorf("请填写收藏夹标题")
		}
		return nil, s.client.CreateFolder(r.Context(), body.Title, body.Intro, body.Private)
	})
	jsonRoute("POST /api/folder/hidden", func(w http.ResponseWriter, r *http.Request) (any, error) {
		var body struct {
			ID     int64 `json:"id"`
			Hidden bool  `json:"hidden"`
		}
		if err := decode(r, &body); err != nil {
			return nil, err
		}
		s.HideFolder(body.ID, body.Hidden)
		return nil, nil
	})
	jsonRoute("POST /api/favorite", func(w http.ResponseWriter, r *http.Request) (any, error) {
		var body struct {
			Track model.Track `json:"track"`
			From  int64       `json:"from"`
			To    int64       `json:"to"`
		}
		if err := decode(r, &body); err != nil {
			return nil, err
		}
		return nil, s.Favorite(r.Context(), body.Track, body.From, body.To)
	})
	jsonRoute("GET /api/discover", func(w http.ResponseWriter, r *http.Request) (any, error) { return s.client.Discover(r.Context()) })
	jsonRoute("GET /api/search", func(w http.ResponseWriter, r *http.Request) (any, error) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		return s.Search(r.Context(), r.URL.Query().Get("q"), page, r.URL.Query().Get("music") == "true")
	})
	jsonRoute("GET /api/history", func(w http.ResponseWriter, r *http.Request) (any, error) { return s.snapshot().History, nil })
	jsonRoute("POST /api/history/clear", func(w http.ResponseWriter, r *http.Request) (any, error) {
		s.mu.Lock()
		s.state.History = []model.Track{}
		s.mu.Unlock()
		s.persist()
		return nil, nil
	})
	jsonRoute("POST /api/play", func(w http.ResponseWriter, r *http.Request) (any, error) {
		var body struct {
			Track model.Track   `json:"track"`
			Queue []model.Track `json:"queue"`
		}
		if err := decode(r, &body); err != nil {
			return nil, err
		}
		return nil, s.Play(body.Track, body.Queue)
	})
	jsonRoute("POST /api/command", func(w http.ResponseWriter, r *http.Request) (any, error) {
		var body struct {
			Action string  `json:"action"`
			Value  float64 `json:"value"`
		}
		if err := decode(r, &body); err != nil {
			return nil, err
		}
		return nil, s.Command(body.Action, body.Value)
	})
	jsonRoute("POST /api/queue", func(w http.ResponseWriter, r *http.Request) (any, error) {
		var body struct {
			Action string      `json:"action"`
			Track  model.Track `json:"track"`
			ID     string      `json:"id"`
		}
		if err := decode(r, &body); err != nil {
			return nil, err
		}
		return nil, s.QueueAction(body.Action, body.Track, body.ID)
	})
	jsonRoute("POST /api/settings", func(w http.ResponseWriter, r *http.Request) (any, error) {
		var settings model.Settings
		if err := decode(r, &settings); err != nil {
			return nil, err
		}
		return nil, s.SetSettings(settings)
	})
	jsonRoute("POST /api/hotkeys/record", func(w http.ResponseWriter, r *http.Request) (any, error) {
		var body struct {
			Recording bool `json:"recording"`
		}
		if err := decode(r, &body); err != nil {
			return nil, err
		}
		s.RecordHotkeys(body.Recording)
		return nil, nil
	})
	jsonRoute("POST /api/lyrics", func(w http.ResponseWriter, r *http.Request) (any, error) {
		var track model.Track
		if err := decode(r, &track); err != nil {
			return nil, err
		}
		return s.client.Lyrics(r.Context(), track, s.snapshot().Settings.LyricsSource), nil
	})
	jsonRoute("GET /api/downloads", func(w http.ResponseWriter, r *http.Request) (any, error) { return s.Downloads(), nil })
	jsonRoute("POST /api/download", func(w http.ResponseWriter, r *http.Request) (any, error) {
		var track model.Track
		if err := decode(r, &track); err != nil {
			return nil, err
		}
		return nil, s.AddDownload(track)
	})
	jsonRoute("POST /api/download/action", func(w http.ResponseWriter, r *http.Request) (any, error) {
		var body struct {
			ID     string `json:"id"`
			Action string `json:"action"`
		}
		if err := decode(r, &body); err != nil {
			return nil, err
		}
		if body.Action == "reveal" {
			return nil, s.RevealDownload(body.ID)
		}
		return nil, s.DownloadAction(body.ID, body.Action)
	})
	jsonRoute("POST /api/window", func(w http.ResponseWriter, r *http.Request) (any, error) {
		var body struct {
			Action string `json:"action"`
		}
		if err := decode(r, &body); err != nil {
			return nil, err
		}
		return nil, s.Window(body.Action)
	})
	jsonRoute("POST /api/dialog/directory", func(w http.ResponseWriter, r *http.Request) (any, error) {
		path, err := s.ChooseDirectory()
		return map[string]string{"path": path}, err
	})
	jsonRoute("POST /api/dialog/files", func(w http.ResponseWriter, r *http.Request) (any, error) { return s.LocalFiles() })
	jsonRoute("POST /api/open", func(w http.ResponseWriter, r *http.Request) (any, error) {
		var body struct {
			URL string `json:"url"`
		}
		if err := decode(r, &body); err != nil {
			return nil, err
		}
		u, err := url.Parse(body.URL)
		if err != nil || u.Scheme != "https" || !(u.Hostname() == "www.bilibili.com" || u.Hostname() == "github.com") {
			return nil, fmt.Errorf("仅支持打开 Bilibili 和 GitHub 链接")
		}
		s.mu.RLock()
		fn := s.desktop.OpenURL
		s.mu.RUnlock()
		if fn != nil {
			return nil, fn(body.URL)
		}
		return nil, nil
	})
	jsonRoute("GET /api/update", func(w http.ResponseWriter, r *http.Request) (any, error) {
		req, _ := http.NewRequestWithContext(r.Context(), "GET", "https://api.github.com/repos/xxay-16/BiliTune/releases/latest", nil)
		req.Header.Set("User-Agent", "BiliTune/"+model.Version)
		response, err := s.client.HTTP.Do(req)
		if err != nil {
			return nil, err
		}
		defer response.Body.Close()
		if response.StatusCode == 404 {
			return map[string]any{"available": false, "version": model.Version}, nil
		}
		if response.StatusCode != 200 {
			return nil, fmt.Errorf("GitHub HTTP %d", response.StatusCode)
		}
		var release struct {
			Tag string `json:"tag_name"`
			URL string `json:"html_url"`
		}
		if err = json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&release); err != nil {
			return nil, err
		}
		return map[string]any{"version": release.Tag, "url": release.URL, "available": versionNewer(release.Tag, model.Version)}, nil
	})
	mux.HandleFunc("GET /api/image", func(w http.ResponseWriter, r *http.Request) {
		raw := r.URL.Query().Get("url")
		u, err := url.Parse(raw)
		if err != nil || u.Scheme != "https" || !allowedImageHost(u.Hostname()) {
			http.Error(w, "Invalid image", 400)
			return
		}
		req, _ := http.NewRequestWithContext(r.Context(), "GET", raw, nil)
		req.Header.Set("Referer", bili.Referer)
		req.Header.Set("User-Agent", bili.UserAgent)
		response, err := s.client.HTTP.Do(req)
		if err != nil {
			http.Error(w, "Image unavailable", 502)
			return
		}
		defer response.Body.Close()
		if response.StatusCode != 200 {
			http.Error(w, "Image unavailable", 502)
			return
		}
		contentType := response.Header.Get("Content-Type")
		if !strings.HasPrefix(contentType, "image/") {
			http.Error(w, "Invalid image", 400)
			return
		}
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Cache-Control", "private, max-age=86400")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		_, _ = io.Copy(w, io.LimitReader(response.Body, 8<<20))
	})
	return mux
}
func allowedImageHost(host string) bool {
	for _, suffix := range []string{"hdslb.com", "biliimg.com", "bilibili.com"} {
		if host == suffix || strings.HasSuffix(host, "."+suffix) {
			return true
		}
	}
	return false
}
func versionNewer(remote, local string) bool {
	parse := func(value string) [3]int {
		var version [3]int
		value = strings.TrimPrefix(value, "v")
		for i, part := range strings.SplitN(strings.SplitN(value, "-", 2)[0], ".", 3) {
			version[i], _ = strconv.Atoi(part)
		}
		return version
	}
	a, b := parse(remote), parse(local)
	for i := range a {
		if a[i] != b[i] {
			return a[i] > b[i]
		}
	}
	return false
}
