package bili

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/xxay-16/BiliTune/internal/model"
)

var lrcTime = regexp.MustCompile(`\[(\d+):(\d+(?:\.\d+)?)\]`)

func ParseLRC(raw string) []model.Lyric {
	lines := []model.Lyric{}
	scanner := bufio.NewScanner(strings.NewReader(raw))
	for scanner.Scan() {
		line := scanner.Text()
		matches := lrcTime.FindAllStringSubmatch(line, -1)
		text := strings.TrimSpace(lrcTime.ReplaceAllString(line, ""))
		if text == "" {
			continue
		}
		for _, m := range matches {
			minute, _ := strconv.Atoi(m[1])
			second, _ := strconv.ParseFloat(m[2], 64)
			lines = append(lines, model.Lyric{Time: float64(minute*60) + second, Text: text})
		}
	}
	sort.SliceStable(lines, func(i, j int) bool { return lines[i].Time < lines[j].Time })
	return lines
}
func (c *Client) text(ctx context.Context, raw string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", URL(raw), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Referer", Referer)
	response, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return nil, fmt.Errorf("歌词服务 HTTP %d", response.StatusCode)
	}
	return io.ReadAll(io.LimitReader(response.Body, 4<<20))
}
func (c *Client) biliLyrics(ctx context.Context, track model.Track) []model.Lyric {
	if track.AudioID > 0 {
		data, err := c.Request(ctx, "GET", "/audio/music-service-c/web/song/info", url.Values{"sid": {strconv.FormatInt(track.AudioID, 10)}}, false)
		if err == nil && String(data["lyric"]) != "" {
			if raw, err := c.text(ctx, String(data["lyric"])); err == nil {
				if lines := ParseLRC(string(raw)); len(lines) > 0 {
					return lines
				}
			}
		}
	}
	if track.CID == 0 {
		var err error
		track, err = c.Hydrate(ctx, track)
		if err != nil {
			return nil
		}
	}
	data, err := c.Request(ctx, "GET", "/x/player/wbi/v2", url.Values{"bvid": {track.BVID}, "cid": {strconv.FormatInt(track.CID, 10)}}, true)
	if err != nil {
		return nil
	}
	for _, item := range List(Map(data["subtitle"])["subtitles"]) {
		subtitle := Map(item)
		raw, err := c.text(ctx, String(subtitle["subtitle_url"]))
		if err != nil {
			continue
		}
		var body struct {
			Body []struct {
				From    float64 `json:"from"`
				Content string  `json:"content"`
			} `json:"body"`
		}
		if json.Unmarshal(raw, &body) != nil {
			continue
		}
		lines := []model.Lyric{}
		for _, line := range body.Body {
			if text := strings.TrimSpace(line.Content); text != "" {
				lines = append(lines, model.Lyric{Time: line.From, Text: text})
			}
		}
		if len(lines) > 0 {
			return lines
		}
	}
	return nil
}
func (c *Client) lrcLyrics(ctx context.Context, track model.Track) []model.Lyric {
	params := url.Values{"track_name": {track.Title}}
	if track.Artist != "" {
		params.Set("artist_name", track.Artist)
	}
	raw, err := c.text(ctx, "https://lrclib.net/api/search?"+params.Encode())
	if err != nil {
		return nil
	}
	var items []struct {
		Synced   string  `json:"syncedLyrics"`
		Plain    string  `json:"plainLyrics"`
		Duration float64 `json:"duration"`
	}
	if json.Unmarshal(raw, &items) != nil {
		return nil
	}
	for _, item := range items {
		if lines := ParseLRC(item.Synced); len(lines) > 0 {
			return lines
		}
	}
	if len(items) > 0 {
		lines := []model.Lyric{}
		for _, text := range strings.Split(items[0].Plain, "\n") {
			if text = strings.TrimSpace(text); text != "" {
				lines = append(lines, model.Lyric{Text: text})
			}
		}
		return lines
	}
	return nil
}
func (c *Client) Lyrics(ctx context.Context, track model.Track, preference string) []model.Lyric {
	var lines []model.Lyric
	if preference == "lrclibOnly" || preference == "lrclibFirst" {
		lines = c.lrcLyrics(ctx, track)
		if len(lines) > 0 || preference == "lrclibOnly" {
			return nonNilLyrics(lines)
		}
	}
	lines = c.biliLyrics(ctx, track)
	if len(lines) > 0 || preference == "biliOnly" {
		return nonNilLyrics(lines)
	}
	return nonNilLyrics(c.lrcLyrics(ctx, track))
}
func nonNilLyrics(lines []model.Lyric) []model.Lyric {
	if lines == nil {
		return []model.Lyric{}
	}
	return lines
}
