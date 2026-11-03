package bili

import (
	"context"
	"fmt"
	"github.com/xxay-16/BiliTune/internal/model"
	"net/http"
	"strconv"
	"testing"
)

func TestMediaPreservesCreatorIdentity(t *testing.T) {
	for _, test := range []struct {
		name     string
		item     map[string]any
		favorite bool
		want     int64
	}{{"search", map[string]any{"aid": 10, "mid": 123, "author": "UP", "upic": "//i0.hdslb.com/face.jpg"}, false, 123}, {"favorites", map[string]any{"id": 10, "upper": map[string]any{"mid": 456, "name": "UP"}}, true, 456}, {"ranking", map[string]any{"aid": 10, "owner": map[string]any{"mid": 789, "name": "UP"}}, false, 789}} {
		t.Run(test.name, func(t *testing.T) {
			track := media(test.item, test.favorite)
			if track.ArtistMID != test.want || track.Artist != "UP" {
				t.Fatalf("lost uploader: %+v", track)
			}
		})
	}
}
func TestResolveLegacyArtistAndKnownMID(t *testing.T) {
	calls := 0
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Query().Get("bvid") != "BV1234567890" {
			t.Error("missing video id")
		}
		envelope(w, map[string]any{"bvid": "BV1234567890", "aid": 1, "owner": map[string]any{"mid": 403449453, "name": "Spylent", "face": "//i0.hdslb.com/face.jpg"}})
	})
	a, err := c.ResolveArtist(context.Background(), model.Track{ID: "old", BVID: "BV1234567890"})
	if err != nil || a.MID != 403449453 || a.Name != "Spylent" {
		t.Fatal(a, err)
	}
	_, err = c.ResolveArtist(context.Background(), model.Track{ArtistMID: 42, Artist: "Known"})
	if err != nil || calls != 1 {
		t.Fatal("known creator should not need a lookup", calls, err)
	}
}
func TestArtistAlbumsAllPages(t *testing.T) {
	calls := 0
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		page, _ := strconv.Atoi(r.URL.Query().Get("page_num"))
		if r.URL.Query().Get("mid") != "123" {
			t.Error("missing owner")
		}
		list := []any{}
		for i := (page - 1) * 20; i < page*20 && i < 25; i++ {
			list = append(list, map[string]any{"meta": map[string]any{"season_id": i + 1, "name": fmt.Sprintf("album%d", i+1), "total": 8}})
		}
		envelope(w, map[string]any{"items_lists": map[string]any{"page": map[string]any{"total": 25}, "seasons_list": list, "series_list": []any{}}})
	})
	albums, err := c.ArtistAlbums(context.Background(), 123)
	if err != nil || len(albums) != 25 || calls != 2 || albums[24].Title != "album25" {
		t.Fatal(len(albums), calls, err)
	}
}
func TestArtistAlbumsKeepSeasonAndSeriesWithSameID(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		envelope(w, map[string]any{"items_lists": map[string]any{"page": map[string]any{"total": 2}, "seasons_list": []any{map[string]any{"meta": map[string]any{"season_id": 7, "name": "合集"}}}, "series_list": []any{map[string]any{"meta": map[string]any{"series_id": 7, "name": "系列"}, "archives": []any{map[string]any{"pic": "//i0.hdslb.com/cover.jpg"}}}}}})
	})
	albums, err := c.ArtistAlbums(context.Background(), 123)
	if err != nil || len(albums) != 2 || albums[0].Type != "season" || albums[1].Type != "series" || albums[1].CoverURL == "" {
		t.Fatal(albums, err)
	}
}
func TestAlbumAllTracks(t *testing.T) {
	for _, kind := range []string{"season", "series"} {
		t.Run(kind, func(t *testing.T) {
			pages := 0
			c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/x/series/series" {
					envelope(w, map[string]any{"meta": map[string]any{"series_id": 7, "name": "专辑", "total": 61}})
					return
				}
				pages++
				key := "page_num"
				if kind == "series" {
					key = "pn"
					if r.URL.Query().Get("series_id") != "7" || r.URL.Query().Get("sort") != "asc" {
						t.Error("wrong series request")
					}
				}
				page, _ := strconv.Atoi(r.URL.Query().Get(key))
				list := []any{}
				for i := (page - 1) * 30; i < page*30 && i < 61; i++ {
					list = append(list, map[string]any{"aid": i + 1, "bvid": fmt.Sprintf("BV%010d", i+1), "title": fmt.Sprintf("track%d", i+1), "duration": 100})
				}
				envelope(w, map[string]any{"meta": map[string]any{"season_id": 7, "name": "专辑", "total": 61}, "page": map[string]any{"total": 61}, "archives": list})
			})
			detail, err := c.Album(context.Background(), 123, 7, kind)
			if err != nil || pages != 3 || len(detail.Tracks) != 61 || detail.Tracks[60].Title != "track61" || detail.Tracks[0].ArtistMID != 123 || detail.Album.ID != 7 {
				t.Fatal(pages, len(detail.Tracks), err)
			}
		})
	}
}
func TestAlbumRejectsRepeatingPagination(t *testing.T) {
	calls := 0
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		envelope(w, map[string]any{"meta": map[string]any{"total": 100}, "archives": []any{map[string]any{"aid": 1, "title": "same"}}})
	})
	_, err := c.Album(context.Background(), 123, 7, "season")
	if err == nil || calls != 2 {
		t.Fatal("repeated page must stop", calls, err)
	}
}
