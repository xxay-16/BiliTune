package app

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/xxay-16/BiliTune/internal/bili"
	"github.com/xxay-16/BiliTune/internal/model"
	"github.com/xxay-16/BiliTune/internal/player"
	"github.com/xxay-16/BiliTune/internal/store"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeEngine struct {
	mu    sync.Mutex
	snap  player.Snapshot
	loads []string
	start float64
}

func (e *fakeEngine) Load(raw string, _ map[string]string, start float64) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.loads = append(e.loads, raw)
	e.start = start
	e.snap = player.Snapshot{Playing: true, Position: start, Duration: 100}
	return nil
}
func (e *fakeEngine) Play() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.snap.Playing = true
	return nil
}
func (e *fakeEngine) Pause() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.snap.Playing = false
	return nil
}
func (e *fakeEngine) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.snap.Playing = false
	e.snap.Position = 0
	return nil
}
func (e *fakeEngine) Seek(v float64) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.snap.Position = v
	return nil
}
func (e *fakeEngine) SetVolume(float64) error   { return nil }
func (e *fakeEngine) SetRate(float64) error     { return nil }
func (e *fakeEngine) Close()                    {}
func (e *fakeEngine) Snapshot() player.Snapshot { e.mu.Lock(); defer e.mu.Unlock(); return e.snap }
func setup(t *testing.T) (*Service, *fakeEngine) {
	t.Helper()
	st, state, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	engine := &fakeEngine{}
	s := New(st, state, bili.New(""), engine, "")
	t.Cleanup(s.Close)
	return s, engine
}
func eventually(t *testing.T, test func() bool) {
	t.Helper()
	until := time.Now().Add(3 * time.Second)
	for !test() {
		if time.Now().After(until) {
			t.Fatal("timed out")
		}
		time.Sleep(10 * time.Millisecond)
	}
}
func local(id string) model.Track {
	return model.Track{ID: id, Title: id, LocalPath: id + ".wav", DurationMS: 100000}
}
func TestPreviousAlwaysSkipsAndStopKeepsQueue(t *testing.T) {
	s, e := setup(t)
	tracks := []model.Track{local("a"), local("b"), local("c")}
	_ = s.Play(tracks[1], tracks)
	eventually(t, func() bool { s.mu.RLock(); defer s.mu.RUnlock(); return s.loaded })
	_ = s.Command("seek", 35)
	if err := s.Command("previousTrack", 0); err != nil {
		t.Fatal(err)
	}
	eventually(t, func() bool { s.mu.RLock(); defer s.mu.RUnlock(); return s.loaded && s.state.Player.Track.ID == "a" })
	if err := s.Command("stop", 0); err != nil {
		t.Fatal(err)
	}
	if s.Player().Playing || s.Player().Position != 0 || len(s.Queue()) != 3 {
		t.Fatal(s.Player())
	}
	if e.Snapshot().Playing {
		t.Fatal("still playing")
	}
}
func TestRestoredPositionAndPauseDuringLoad(t *testing.T) {
	s, e := setup(t)
	track := local("resume")
	s.mu.Lock()
	s.state.Player.Track = &track
	s.state.Player.Position = 43
	s.state.Queue = []model.Track{track}
	s.mu.Unlock()
	_ = s.Command("play", 0)
	eventually(t, func() bool { s.mu.RLock(); defer s.mu.RUnlock(); return s.loaded })
	e.mu.Lock()
	start := e.start
	e.mu.Unlock()
	if start != 43 {
		t.Fatal("lost resume position", start)
	}
	gate := make(chan struct{})
	requested := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var data any
		switch r.URL.Path {
		case "/x/web-interface/nav":
			data = map[string]any{"wbi_img": map[string]any{"img_url": "https://i/7cd084941338484aae1ad9425b84077c.png", "sub_url": "https://i/4932caff0ff746eab6f01bf08b70ac45.png"}}
		default:
			close(requested)
			select {
			case <-gate:
			case <-r.Context().Done():
				return
			}
			data = map[string]any{"dash": map[string]any{"audio": []any{map[string]any{"baseUrl": "https://example.invalid/audio", "bandwidth": 192000}}}}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": data})
	}))
	defer server.Close()
	s.client.APIBase = server.URL
	_ = s.Play(model.Track{ID: "remote", BVID: "BV1234567890", CID: 1}, nil)
	<-requested
	_ = s.Command("pause", 0)
	close(gate)
	eventually(t, func() bool { s.mu.RLock(); defer s.mu.RUnlock(); return s.loaded })
	if e.Snapshot().Playing {
		t.Fatal("load overrode pause")
	}
}
func TestLateLoadCannotReplaceNewTrack(t *testing.T) {
	s, e := setup(t)
	requested := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(requested); <-r.Context().Done() }))
	defer server.Close()
	s.client.APIBase = server.URL
	_ = s.Play(model.Track{ID: "old", BVID: "BV1234567890", CID: 1}, nil)
	<-requested
	_ = s.Play(local("new"), nil)
	eventually(t, func() bool { s.mu.RLock(); defer s.mu.RUnlock(); return s.loaded })
	if s.Player().Track.ID != "new" {
		t.Fatal(s.Player())
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(e.loads) != 1 || e.loads[0] != "new.wav" {
		t.Fatal(e.loads)
	}
}
func TestHTTPRejectsForeignOriginAndNeverExposesSession(t *testing.T) {
	s, _ := setup(t)
	s.client.SetCookie("SESSDATA=top-secret")
	h := s.Handler()
	r := httptest.NewRequest("POST", "http://localhost/api/command", strings.NewReader(`{"action":"stop"}`))
	r.Header.Set("Origin", "https://foreign.invalid")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal(w.Code)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/api/bootstrap", nil))
	if strings.Contains(w.Body.String(), "top-secret") {
		t.Fatal("credential exposed")
	}
}
func TestDownloadResumeAndSanitizedPath(t *testing.T) {
	s, _ := setup(t)
	payload := strings.Repeat("audio", 5000)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Range") != "bytes=100-" {
			t.Error(r.Header.Get("Range"))
		}
		w.Header().Set("Content-Range", fmt.Sprintf("bytes 100-%d/%d", len(payload)-1, len(payload)))
		w.WriteHeader(206)
		_, _ = w.Write([]byte(payload[100:]))
	}))
	defer server.Close()
	dir := t.TempDir()
	s.state.Settings.DownloadDir = dir
	track := model.Track{ID: "t", Title: "../bad:name", SourceURL: server.URL, SourceExpiresAt: time.Now().Add(time.Hour).UnixMilli()}
	task := model.Download{ID: "safe", Track: track, Status: "downloading"}
	s.state.Downloads = []model.Download{task}
	path := filepath.Join(dir, safeName(track.Title)+"-safe.m4a")
	_ = os.WriteFile(path+".part", []byte(payload[:100]), 0600)
	if err := s.fetchDownload(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	if string(got) != payload || s.Downloads()[0].Status != "completed" {
		t.Fatal("download corrupted")
	}
}

func TestResolvedArtistIsSavedToOldHistory(t *testing.T) {
	s, _ := setup(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{"bvid": "BV1234567890", "aid": 1, "owner": map[string]any{"mid": 123, "name": "UP", "face": "https://i0.hdslb.com/face.jpg"}}})
	}))
	defer server.Close()
	s.client.APIBase = server.URL
	track := model.Track{ID: "old", BVID: "BV1234567890", Artist: "UP"}
	s.state.History = []model.Track{track}
	s.state.Queue = []model.Track{track}
	s.state.Player.Track = &track
	artist, err := s.ResolveArtist(context.Background(), track)
	if err != nil || artist.MID != 123 {
		t.Fatal(artist, err)
	}
	_, saved, err := store.Open(s.store.Dir)
	if err != nil || saved.History[0].ArtistMID != 123 || saved.Queue[0].ArtistMID != 123 || saved.Player.Track.ArtistMID != 123 {
		t.Fatal("resolved creator was not persisted", err)
	}
}
