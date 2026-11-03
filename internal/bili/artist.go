package bili

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/xxay-16/BiliTune/internal/model"
)

func (c *Client) ResolveArtist(ctx context.Context, track model.Track) (model.Artist, error) {
	if track.ArtistMID > 0 {
		return model.Artist{MID: track.ArtistMID, Name: track.Artist, Avatar: track.ArtistAvatar}, nil
	}
	if track.AudioID > 0 {
		data, err := c.Request(ctx, "GET", "/audio/music-service-c/web/song/info", url.Values{"sid": {strconv.FormatInt(track.AudioID, 10)}}, false)
		if err != nil {
			return model.Artist{}, err
		}
		if mid := Int(data["uid"]); mid > 0 {
			return model.Artist{MID: mid, Name: clean(data["author"])}, nil
		}
	} else if track.BVID != "" || track.AID > 0 {
		hydrated, err := c.Hydrate(ctx, track)
		if err != nil {
			return model.Artist{}, err
		}
		if hydrated.ArtistMID > 0 {
			return model.Artist{MID: hydrated.ArtistMID, Name: hydrated.Artist, Avatar: hydrated.ArtistAvatar}, nil
		}
	}
	return model.Artist{}, fmt.Errorf("这首音频没有可用的 UP 主账号信息")
}

func (c *Client) Artist(ctx context.Context, mid int64) (model.Artist, error) {
	if mid <= 0 {
		return model.Artist{}, fmt.Errorf("UP 主账号无效")
	}
	// The public card endpoint also supplies archive/follower counts without WBI.
	data, err := c.Request(ctx, "GET", "/x/web-interface/card", url.Values{"mid": {strconv.FormatInt(mid, 10)}, "photo": {"true"}}, false)
	if err == nil {
		card := Map(data["card"])
		if String(card["name"]) != "" {
			return model.Artist{MID: mid, Name: clean(card["name"]), Avatar: URL(String(card["face"])), Signature: clean(card["sign"]), Followers: Int(data["follower"]), VideoCount: int(Int(data["archive_count"])), Banner: URL(String(Map(data["space"])["l_img"]))}, nil
		}
	}
	if ctx.Err() != nil {
		return model.Artist{}, ctx.Err()
	}
	data, err = c.Request(ctx, "GET", "/x/space/wbi/acc/info", url.Values{"mid": {strconv.FormatInt(mid, 10)}}, true)
	if err != nil {
		return model.Artist{}, err
	}
	if String(data["name"]) == "" {
		return model.Artist{}, fmt.Errorf("无法读取这个 UP 主的公开主页")
	}
	return model.Artist{MID: mid, Name: clean(data["name"]), Avatar: URL(String(data["face"])), Signature: clean(data["sign"]), Banner: URL(String(data["top_photo"]))}, nil
}

func albumFrom(item map[string]any, kind string, mid int64) model.Album {
	meta := Map(item["meta"])
	if len(meta) == 0 {
		meta = item
	}
	idKey := "season_id"
	if kind == "series" {
		idKey = "series_id"
	}
	id := Int(meta[idKey])
	if id == 0 {
		id = Int(meta["id"])
	}
	title := clean(meta["name"])
	if title == "" {
		title = clean(meta["title"])
	}
	cover := URL(String(meta["cover"]))
	if cover == "" {
		archives := List(item["archives"])
		if len(archives) > 0 {
			cover = URL(String(Map(archives[0])["pic"]))
		}
	}
	return model.Album{ID: id, Type: kind, MID: mid, Title: title, CoverURL: cover, Intro: clean(meta["description"]), Count: int(Int(meta["total"]))}
}

func (c *Client) ArtistAlbums(ctx context.Context, mid int64) ([]model.Album, error) {
	if mid <= 0 {
		return nil, fmt.Errorf("UP 主账号无效")
	}
	albums := []model.Album{}
	seen := map[string]bool{}
	for page := 1; ; page++ {
		data, err := c.Request(ctx, "GET", "/x/polymer/web-space/seasons_series_list", url.Values{"mid": {strconv.FormatInt(mid, 10)}, "page_num": {strconv.Itoa(page)}, "page_size": {"20"}}, false)
		if err != nil {
			return nil, err
		}
		items := Map(data["items_lists"])
		if len(items) == 0 {
			items = data
		}
		pageItems := 0
		added := 0
		for _, group := range []struct{ key, kind string }{{"seasons_list", "season"}, {"series_list", "series"}} {
			list := List(items[group.key])
			pageItems += len(list)
			for _, value := range list {
				album := albumFrom(Map(value), group.kind, mid)
				if album.ID <= 0 {
					continue
				}
				key := fmt.Sprintf("%s:%d", album.Type, album.ID)
				if !seen[key] {
					seen[key] = true
					albums = append(albums, album)
					added++
				}
			}
		}
		if pageItems == 0 {
			break
		}
		total := int(Int(Map(items["page"])["total"]))
		if total == 0 {
			total = int(Int(Map(data["page"])["total"]))
		}
		if total > 0 && len(albums) >= total {
			break
		}
		if total == 0 && pageItems < 20 {
			break
		}
		if added == 0 {
			return nil, fmt.Errorf("Bilibili 返回了重复的专辑分页，请稍后重试")
		}
	}
	return albums, nil
}

func (c *Client) ArtistVideos(ctx context.Context, mid int64, page int) (model.ArtistVideos, error) {
	if mid <= 0 {
		return model.ArtistVideos{}, fmt.Errorf("UP 主账号无效")
	}
	if page < 1 {
		page = 1
	}
	data, err := c.Request(ctx, "GET", "/x/space/wbi/arc/search", url.Values{"mid": {strconv.FormatInt(mid, 10)}, "pn": {strconv.Itoa(page)}, "ps": {"30"}, "order": {"pubdate"}}, true)
	if err != nil {
		return model.ArtistVideos{}, err
	}
	tracks := []model.Track{}
	for _, value := range List(Map(data["list"])["vlist"]) {
		track := media(Map(value), false)
		if track.DurationMS == 0 {
			track.DurationMS = duration(Map(value)["length"])
		}
		if track.ArtistMID == 0 {
			track.ArtistMID = mid
		}
		tracks = append(tracks, track)
	}
	total := int(Int(Map(data["page"])["count"]))
	return model.ArtistVideos{Tracks: tracks, Page: page, Total: total, HasMore: len(tracks) > 0 && (total > 0 && page*30 < total || total == 0 && len(tracks) == 30)}, nil
}

func (c *Client) Album(ctx context.Context, mid, id int64, kind string) (model.AlbumDetail, error) {
	if mid <= 0 || id <= 0 || (kind != "season" && kind != "series") {
		return model.AlbumDetail{}, fmt.Errorf("专辑信息无效")
	}
	album := model.Album{ID: id, MID: mid, Type: kind}
	if kind == "series" {
		data, err := c.Request(ctx, "GET", "/x/series/series", url.Values{"series_id": {strconv.FormatInt(id, 10)}}, false)
		if err != nil {
			return model.AlbumDetail{}, err
		}
		album = albumFrom(data, kind, mid)
		if album.ID == 0 {
			album.ID = id
		}
	}
	tracks := []model.Track{}
	seen := map[string]bool{}
	for page := 1; ; page++ {
		endpoint := "/x/polymer/web-space/seasons_archives_list"
		values := url.Values{"mid": {strconv.FormatInt(mid, 10)}, "season_id": {strconv.FormatInt(id, 10)}, "page_num": {strconv.Itoa(page)}, "page_size": {"30"}, "sort_reverse": {"false"}}
		if kind == "series" {
			endpoint = "/x/series/archives"
			values = url.Values{"mid": {strconv.FormatInt(mid, 10)}, "series_id": {strconv.FormatInt(id, 10)}, "pn": {strconv.Itoa(page)}, "ps": {"30"}, "only_normal": {"true"}, "sort": {"asc"}}
		}
		data, err := c.Request(ctx, "GET", endpoint, values, false)
		if err != nil {
			return model.AlbumDetail{}, err
		}
		if kind == "season" && page == 1 {
			album = albumFrom(data, kind, mid)
			if album.ID == 0 {
				album.ID = id
			}
		}
		items := List(data["archives"])
		if len(items) == 0 {
			break
		}
		added := 0
		for _, value := range items {
			track := media(Map(value), false)
			if track.AID <= 0 && track.BVID == "" {
				continue
			}
			if track.ArtistMID == 0 {
				track.ArtistMID = mid
			}
			if !seen[track.ID] {
				seen[track.ID] = true
				tracks = append(tracks, track)
				added++
			}
		}
		total := int(Int(Map(data["page"])["total"]))
		if total == 0 {
			total = album.Count
		}
		if total > 0 && len(tracks) >= total {
			break
		}
		if total == 0 && len(items) < 30 {
			break
		}
		if added == 0 {
			return model.AlbumDetail{}, fmt.Errorf("Bilibili 返回了重复的专辑歌曲分页，请稍后重试")
		}
	}
	if album.CoverURL == "" && len(tracks) > 0 {
		album.CoverURL = tracks[0].CoverURL
	}
	if album.Count == 0 {
		album.Count = len(tracks)
	}
	return model.AlbumDetail{Album: album, Tracks: tracks}, nil
}
