package bili

import (
	"context"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/xxay-16/BiliTune/internal/model"
)

var tags = regexp.MustCompile(`<[^>]*>`)
var bvidPattern = regexp.MustCompile(`BV[0-9A-Za-z]{10}`)
var aidPattern = regexp.MustCompile(`(?i)\bav(\d+)`)
var audioPattern = regexp.MustCompile(`(?i)\bau(\d+)`)

func clean(value any) string { return html.UnescapeString(tags.ReplaceAllString(String(value), "")) }
func duration(value any) int64 {
	if s, ok := value.(string); ok && strings.Contains(s, ":") {
		var total int64
		for _, part := range strings.Split(s, ":") {
			total = total*60 + Int(part)
		}
		return total * 1000
	}
	return Int(value) * 1000
}
func media(item map[string]any, favorite bool) model.Track {
	bvid := String(item["bvid"])
	if bvid == "" {
		bvid = String(item["bv_id"])
	}
	aid := Int(item["aid"])
	if aid == 0 {
		aid = Int(item["id"])
	}
	owner := Map(item["owner"])
	if favorite {
		owner = Map(item["upper"])
	}
	artist := String(owner["name"])
	if artist == "" {
		artist = String(item["author"])
	}
	cover := String(item["pic"])
	if cover == "" {
		cover = String(item["cover"])
	}
	id := bvid
	if id == "" {
		id = fmt.Sprintf("av%d", aid)
	}
	if Int(item["type"]) == 12 || String(item["business"]) == "audio" {
		return model.Track{ID: fmt.Sprintf("au%d", aid), AudioID: aid, Title: clean(item["title"]), Artist: artist, DurationMS: duration(item["duration"]), Type: "audio", CoverURL: URL(cover)}
	}
	return model.Track{ID: id, Title: clean(item["title"]), Artist: artist, DurationMS: duration(item["duration"]), Type: "video", CoverURL: URL(cover), BVID: bvid, AID: aid, CID: Int(item["cid"]), PlayCount: Int(Map(item["stat"])["view"])}
}
func (c *Client) Account(ctx context.Context) (*model.Account, error) {
	data, err := c.Request(ctx, "GET", "/x/web-interface/nav", nil, false)
	if err != nil {
		return nil, err
	}
	if data["isLogin"] != true {
		return nil, nil
	}
	return &model.Account{MID: Int(data["mid"]), Name: String(data["uname"]), Avatar: URL(String(data["face"])), VIP: Int(Map(data["vip"])["status"]) > 0}, nil
}
func (c *Client) QR(ctx context.Context) (map[string]any, error) {
	return c.Request(ctx, "GET", "passport:/x/passport-login/web/qrcode/generate", nil, false)
}
func (c *Client) PollQR(ctx context.Context, key string) (map[string]any, error) {
	return c.Request(ctx, "GET", "passport:/x/passport-login/web/qrcode/poll", url.Values{"qrcode_key": {key}, "source": {"main-fe-header"}}, false)
}

func (c *Client) Folders(ctx context.Context, mid int64) ([]model.Folder, error) {
	data, err := c.Request(ctx, "GET", "/x/v3/fav/folder/created/list-all", url.Values{"up_mid": {strconv.FormatInt(mid, 10)}, "type": {"2"}}, false)
	if err != nil {
		return nil, err
	}
	folders := []model.Folder{}
	for _, value := range List(data["list"]) {
		item := Map(value)
		folders = append(folders, model.Folder{ID: Int(item["id"]), Title: clean(item["title"]), Count: int(Int(item["media_count"])), CoverURL: URL(String(item["cover"])), Intro: String(item["intro"]), Private: Int(item["attr"])&1 != 0})
	}
	return folders, nil
}
func (c *Client) FolderTracks(ctx context.Context, id int64, keyword string) ([]model.Track, error) {
	tracks := []model.Track{}
	seen := map[string]bool{}
	for page := 1; ; page++ {
		params := url.Values{"media_id": {strconv.FormatInt(id, 10)}, "pn": {strconv.Itoa(page)}, "ps": {"20"}, "order": {"mtime"}, "type": {"0"}, "tid": {"0"}, "platform": {"web"}}
		if keyword != "" {
			params.Set("keyword", keyword)
		}
		data, err := c.Request(ctx, "GET", "/x/v3/fav/resource/list", params, false)
		if err != nil {
			return nil, err
		}
		items := List(data["medias"])
		if len(items) == 0 {
			break
		}
		for _, value := range items {
			track := media(Map(value), true)
			if !seen[track.ID] {
				seen[track.ID] = true
				tracks = append(tracks, track)
			}
		}
		more, exists := data["has_more"]
		if exists {
			if more == false || Int(more) == 0 && more != true {
				break
			}
		} else if len(items) < 20 {
			break
		}
	}
	return tracks, nil
}
func (c *Client) CreateFolder(ctx context.Context, title, intro string, private bool) error {
	privacy := "0"
	if private {
		privacy = "1"
	}
	_, err := c.Request(ctx, http.MethodPost, "/x/v3/fav/folder/add", url.Values{"title": {strings.TrimSpace(title)}, "intro": {intro}, "privacy": {privacy}}, false)
	return err
}
func (c *Client) Favorite(ctx context.Context, track model.Track, from, to int64) error {
	if from != 0 && from == to {
		return nil
	}
	if track.AudioID > 0 {
		return fmt.Errorf("此收藏夹操作暂只支持视频音轨，请在 Bilibili 管理音频区收藏")
	}
	if track.AID == 0 {
		var err error
		track, err = c.Hydrate(ctx, track)
		if err != nil {
			return err
		}
	}
	if track.AID == 0 {
		return fmt.Errorf("此歌曲缺少可收藏的视频编号")
	}
	values := url.Values{"rid": {strconv.FormatInt(track.AID, 10)}, "type": {"2"}, "add_media_ids": {""}, "del_media_ids": {""}}
	if from > 0 {
		values.Set("del_media_ids", strconv.FormatInt(from, 10))
	}
	if to > 0 {
		values.Set("add_media_ids", strconv.FormatInt(to, 10))
	}
	_, err := c.Request(ctx, http.MethodPost, "/x/v3/fav/resource/deal", values, false)
	return err
}

func (c *Client) Search(ctx context.Context, query string, page int, musicOnly bool) ([]model.Track, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []model.Track{}, nil
	}
	if match := audioPattern.FindStringSubmatch(query); len(match) > 1 {
		data, err := c.Request(ctx, "GET", "/audio/music-service-c/web/song/info", url.Values{"sid": {match[1]}}, false)
		if err != nil {
			return nil, err
		}
		return []model.Track{{ID: "au" + match[1], AudioID: Int(match[1]), Type: "audio", Title: clean(data["title"]), Artist: String(data["author"]), CoverURL: URL(String(data["cover"])), DurationMS: duration(data["duration"])}}, nil
	}
	if bvid := bvidPattern.FindString(query); bvid != "" {
		track, err := c.Hydrate(ctx, model.Track{BVID: bvid, ID: bvid})
		if err != nil {
			return nil, err
		}
		return []model.Track{track}, nil
	}
	if match := aidPattern.FindStringSubmatch(query); len(match) > 1 {
		track, err := c.Hydrate(ctx, model.Track{AID: Int(match[1]), ID: "av" + match[1]})
		if err != nil {
			return nil, err
		}
		return []model.Track{track}, nil
	}
	params := url.Values{"search_type": {"video"}, "keyword": {query}, "page": {strconv.Itoa(page)}, "page_size": {"20"}, "order": {"totalrank"}}
	if musicOnly {
		params.Set("tids", "3")
	}
	data, err := c.Request(ctx, "GET", "/x/web-interface/wbi/search/type", params, true)
	if err != nil {
		return nil, err
	}
	tracks := []model.Track{}
	for _, item := range List(data["result"]) {
		tracks = append(tracks, media(Map(item), false))
	}
	return tracks, nil
}

type Shelf struct {
	Title  string        `json:"title"`
	Tracks []model.Track `json:"tracks"`
}

func (c *Client) Discover(ctx context.Context) ([]Shelf, error) {
	shelves := []Shelf{}
	for _, zone := range []struct{ ID, Title string }{{"3", "音乐排行榜"}, {"28", "原创音乐"}, {"31", "翻唱精选"}, {"29", "音乐现场"}} {
		data, err := c.Request(ctx, "GET", "/x/web-interface/ranking/v2", url.Values{"rid": {zone.ID}, "type": {"all"}}, true)
		if err != nil {
			if len(shelves) > 0 {
				break
			}
			return nil, err
		}
		shelf := Shelf{Title: zone.Title, Tracks: []model.Track{}}
		for _, item := range List(data["list"]) {
			shelf.Tracks = append(shelf.Tracks, media(Map(item), false))
			if len(shelf.Tracks) >= 12 {
				break
			}
		}
		if len(shelf.Tracks) > 0 {
			shelves = append(shelves, shelf)
		}
	}
	return shelves, nil
}

func (c *Client) Hydrate(ctx context.Context, track model.Track) (model.Track, error) {
	params := url.Values{}
	if track.BVID != "" {
		params.Set("bvid", track.BVID)
	} else if track.AID > 0 {
		params.Set("aid", strconv.FormatInt(track.AID, 10))
	} else {
		return track, fmt.Errorf("缺少视频编号")
	}
	data, err := c.Request(ctx, "GET", "/x/web-interface/view", params, false)
	if err != nil {
		return track, err
	}
	hydrated := media(data, false)
	if track.Title != "" {
		hydrated.Title = track.Title
	}
	if track.Artist != "" {
		hydrated.Artist = track.Artist
	}
	if track.ID != "" {
		hydrated.ID = track.ID
	}
	if hydrated.CID == 0 {
		items := List(data["pages"])
		if len(items) > 0 {
			hydrated.CID = Int(Map(items[0])["cid"])
		}
	}
	return hydrated, nil
}

func (c *Client) Resolve(ctx context.Context, track model.Track, quality string) (model.Track, model.Source, error) {
	if quality == "auto" && track.SourceURL != "" && track.SourceExpiresAt > time.Now().Add(time.Minute).UnixMilli() {
		return track, model.Source{URL: track.SourceURL, Quality: "缓存音源", ExpiresAt: track.SourceExpiresAt}, nil
	}
	if track.AudioID > 0 {
		data, err := c.Request(ctx, "GET", "/audio/music-service-c/url", url.Values{"songid": {strconv.FormatInt(track.AudioID, 10)}, "quality": {"2"}, "privilege": {"2"}, "platform": {"web"}, "mid": {c.CookieValue("DedeUserID")}}, false)
		if err != nil {
			return track, model.Source{}, err
		}
		urls := List(data["cdns"])
		if len(urls) == 0 {
			return track, model.Source{}, fmt.Errorf("此音频暂时没有可用音源")
		}
		return track, model.Source{URL: String(urls[0]), Quality: "音频区", ExpiresAt: time.Now().Add(2 * time.Hour).UnixMilli()}, nil
	}
	if track.CID == 0 || track.BVID == "" {
		var err error
		track, err = c.Hydrate(ctx, track)
		if err != nil {
			return track, model.Source{}, err
		}
	}
	data, err := c.Request(ctx, "GET", "/x/player/wbi/playurl", url.Values{"bvid": {track.BVID}, "cid": {strconv.FormatInt(track.CID, 10)}, "fnval": {"4048"}, "fnver": {"0"}, "fourk": {"1"}}, true)
	if err != nil {
		return track, model.Source{}, err
	}
	dash := Map(data["dash"])
	audio := List(dash["audio"])
	if quality == "lossless" || quality == "auto" {
		flac := Map(Map(dash["flac"])["audio"])
		if String(flac["base_url"]) != "" || String(flac["baseUrl"]) != "" {
			audio = append(audio, flac)
		}
	}
	sort.SliceStable(audio, func(i, j int) bool {
		if quality == "auto" || quality == "lossless" {
			a, b := Int(Map(audio[i])["id"]) == 30251, Int(Map(audio[j])["id"]) == 30251
			if a != b {
				return a
			}
		}
		return Int(Map(audio[i])["bandwidth"]) > Int(Map(audio[j])["bandwidth"])
	})
	if quality == "low" {
		sort.SliceStable(audio, func(i, j int) bool { return Int(Map(audio[i])["bandwidth"]) < Int(Map(audio[j])["bandwidth"]) })
	}
	if quality == "medium" && len(audio) > 1 {
		audio = append(audio[1:], audio[0])
	}
	for _, item := range audio {
		m := Map(item)
		raw := String(m["baseUrl"])
		if raw == "" {
			raw = String(m["base_url"])
		}
		if raw == "" {
			continue
		}
		source := model.Source{URL: raw, Quality: fmt.Sprintf("%d kbps", Int(m["bandwidth"])/1000), ExpiresAt: time.Now().Add(2 * time.Hour).UnixMilli()}
		if Int(m["id"]) == 30251 {
			source.Quality = "FLAC"
		}
		for _, key := range []string{"backupUrl", "backup_url"} {
			for _, backup := range List(m[key]) {
				source.Backups = append(source.Backups, String(backup))
			}
		}
		track.SourceURL = raw
		track.SourceExpiresAt = source.ExpiresAt
		return track, source, nil
	}
	return track, model.Source{}, fmt.Errorf("此视频没有可用的独立音轨，可能已失效或需要登录")
}
