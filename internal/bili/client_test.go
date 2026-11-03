package bili

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/xxay-16/BiliTune/internal/model"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

func modelTrack(aid int64) model.Track { return model.Track{ID: "test", AID: aid} }

func testClient(t *testing.T, fn http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(fn)
	t.Cleanup(server.Close)
	c := New("SESSDATA=test; bili_jct=csrf-test")
	c.APIBase = server.URL
	c.PassportBase = server.URL
	return c
}
func envelope(w http.ResponseWriter, data any) {
	_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": data})
}

func TestFolderAllPages(t *testing.T) {
	pages := 0
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		pages++
		p, _ := strconv.Atoi(r.URL.Query().Get("pn"))
		if r.URL.Query().Get("ps") != "20" {
			t.Error("missing page size")
		}
		items := []any{}
		for i := (p - 1) * 20; i < p*20 && i < 142; i++ {
			items = append(items, map[string]any{"id": i + 1, "bvid": fmt.Sprintf("BV%010d", i+1), "title": fmt.Sprintf("song%d", i), "duration": 100})
		}
		envelope(w, map[string]any{"medias": items, "has_more": p*20 < 142})
	})
	tracks, err := c.FolderTracks(context.Background(), 99, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(tracks) != 142 || pages != 8 || tracks[141].Title != "song141" {
		t.Fatalf("got %d tracks / %d pages", len(tracks), pages)
	}
}
func TestFolderShortPageHonorsMore(t *testing.T) {
	pages := 0
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		pages++
		envelope(w, map[string]any{"medias": []any{map[string]any{"id": pages, "title": "song"}}, "has_more": pages < 3})
	})
	tracks, err := c.FolderTracks(context.Background(), 1, "")
	if err != nil || len(tracks) != 3 {
		t.Fatalf("%d %v", len(tracks), err)
	}
}
func TestFavoriteMoveIsAtomic(t *testing.T) {
	calls := 0
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "POST" || r.URL.Path != "/x/v3/fav/resource/deal" {
			t.Error("wrong endpoint")
		}
		_ = r.ParseForm()
		for k, want := range map[string]string{"rid": "123", "type": "2", "add_media_ids": "22", "del_media_ids": "11", "csrf": "csrf-test"} {
			if r.Form.Get(k) != want {
				t.Errorf("%s: %q", k, r.Form.Get(k))
			}
		}
		envelope(w, nil)
	})
	if err := c.Favorite(context.Background(), modelTrack(123), 11, 22); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatal(calls)
	}
}
func TestSign(t *testing.T) {
	params := url.Values{"foo": {"114"}, "bar": {"514"}, "zab": {"1919810"}}
	got := Sign(params, "ea1db124af3c7062474693fa704f4ff8", 1702204169)
	if got.Get("w_rid") != "8f6f2b5b3d485fe1886cec6a0be8c5d4" {
		t.Fatalf("signature %v", got)
	}
	dirty := Sign(url.Values{"foo": {"a!'()*b c"}}, "key", 1)
	if dirty.Get("foo") != "ab c" {
		t.Fatal(dirty)
	}
	if params.Get("wts") != "" {
		t.Fatal("sign mutated caller")
	}
}
func TestPostNeedsCSRF(t *testing.T) {
	c := New("SESSDATA=test")
	_, err := c.Request(context.Background(), "POST", "/unused", nil, false)
	if err == nil || !strings.Contains(err.Error(), "bili_jct") {
		t.Fatal(err)
	}
}
func TestLogoutClearsCookieJar(t *testing.T) {
	c := New("SESSDATA=old")
	u, _ := url.Parse(c.APIBase)
	c.jar.SetCookies(u, []*http.Cookie{{Name: "SESSDATA", Value: "old"}})
	c.SetCookie("")
	if len(c.jar.Cookies(u)) != 0 {
		t.Fatal("stale authenticated cookie")
	}
}
func TestLRC(t *testing.T) {
	lines := ParseLRC("[ar:artist]\n[01:02.50][00:03.2]副歌\n[00:01.00]开头")
	if len(lines) != 3 || lines[0].Time != 1 || lines[2].Time != 62.5 || lines[1].Text != "副歌" {
		t.Fatal(lines)
	}
}
