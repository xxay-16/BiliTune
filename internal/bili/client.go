package bili

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/142.0.0.0 Safari/537.36"
const Referer = "https://www.bilibili.com/"

type Client struct {
	HTTP         *http.Client
	APIBase      string
	PassportBase string
	mu           sync.RWMutex
	cookie       string
	wbi          string
	wbiAt        time.Time
	rateMu       sync.Mutex
	next         time.Time
	jar          *sessionJar
}

type sessionJar struct {
	mu    sync.Mutex
	value *cookiejar.Jar
}

func (j *sessionJar) Cookies(u *url.URL) []*http.Cookie {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.value.Cookies(u)
}
func (j *sessionJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.value.SetCookies(u, cookies)
}
func (j *sessionJar) clear() { j.mu.Lock(); defer j.mu.Unlock(); j.value, _ = cookiejar.New(nil) }

type APIError struct {
	Code    int
	Message string
}

func (e *APIError) Error() string { return fmt.Sprintf("Bilibili：%s（%d）", e.Message, e.Code) }

func New(cookie string) *Client {
	value, _ := cookiejar.New(nil)
	jar := &sessionJar{value: value}
	return &Client{HTTP: &http.Client{Timeout: 25 * time.Second, Jar: jar}, APIBase: "https://api.bilibili.com", PassportBase: "https://passport.bilibili.com", cookie: cookie, jar: jar}
}
func (c *Client) SetCookie(cookie string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cookie = cookie
	c.wbi = ""
	if c.jar != nil {
		c.jar.clear()
	}
}
func (c *Client) Cookie() string { c.mu.RLock(); defer c.mu.RUnlock(); return c.cookie }
func (c *Client) CookieValue(name string) string {
	r := http.Request{Header: http.Header{"Cookie": []string{c.Cookie()}}}
	for _, cookie := range r.Cookies() {
		if cookie.Name == name {
			return cookie.Value
		}
	}
	return ""
}
func (c *Client) gate(ctx context.Context) error {
	c.rateMu.Lock()
	wait := time.Until(c.next)
	if wait < 0 {
		wait = 0
	}
	c.next = time.Now().Add(wait + 150*time.Millisecond)
	c.rateMu.Unlock()
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (c *Client) Request(ctx context.Context, method, path string, values url.Values, signed bool) (map[string]any, error) {
	if values == nil {
		values = url.Values{}
	}
	if signed {
		key, err := c.wbiKey(ctx)
		if err != nil {
			return nil, err
		}
		values = Sign(values, key, time.Now().Unix())
	}
	base := c.APIBase
	if strings.HasPrefix(path, "passport:") {
		base = c.PassportBase
		path = strings.TrimPrefix(path, "passport:")
	}
	if err := c.gate(ctx); err != nil {
		return nil, err
	}
	var body io.Reader
	endpoint := base + path
	if method == http.MethodGet {
		if len(values) > 0 {
			endpoint += "?" + values.Encode()
		}
	} else {
		csrf := c.CookieValue("bili_jct")
		if csrf == "" {
			return nil, fmt.Errorf("请先登录 Bilibili，当前登录状态缺少 bili_jct")
		}
		values.Set("csrf", csrf)
		body = strings.NewReader(values.Encode())
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, err
	}
	c.Headers(req)
	if method != http.MethodGet {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	response, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("连接 Bilibili 失败：%w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		if response.StatusCode == 412 {
			return nil, fmt.Errorf("Bilibili 暂时拒绝了请求（HTTP 412），请登录后重试或稍后再试")
		}
		return nil, fmt.Errorf("Bilibili HTTP %d，请稍后重试", response.StatusCode)
	}
	var envelope struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, 16<<20))
	if err = decoder.Decode(&envelope); err != nil {
		return nil, fmt.Errorf("Bilibili 返回了无法识别的响应")
	}
	if envelope.Code != 0 {
		return nil, &APIError{envelope.Code, envelope.Message}
	}
	data := map[string]any{}
	if len(envelope.Data) > 0 && string(envelope.Data) != "null" {
		dec := json.NewDecoder(strings.NewReader(string(envelope.Data)))
		dec.UseNumber()
		if err = dec.Decode(&data); err != nil {
			return nil, err
		}
	}
	if strings.HasPrefix(path, "/x/passport-login/web/qrcode/poll") && Int(data["code"]) == 0 {
		cookies := map[string]string{}
		for _, item := range response.Cookies() {
			cookies[item.Name] = item.Value
		}
		if raw := String(data["url"]); raw != "" {
			if u, e := url.Parse(raw); e == nil {
				for _, k := range []string{"SESSDATA", "bili_jct", "DedeUserID", "ac_time_value"} {
					if value := u.Query().Get(k); value != "" {
						cookies[k] = value
					}
				}
			}
		}
		if cookies["SESSDATA"] != "" {
			parts := []string{}
			for k, v := range cookies {
				parts = append(parts, k+"="+v)
			}
			c.SetCookie(strings.Join(parts, "; "))
		}
	}
	return data, nil
}

func (c *Client) Headers(req *http.Request) {
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Referer", Referer)
	req.Header.Set("Origin", "https://www.bilibili.com")
	if cookie := c.Cookie(); cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
}

var mixin = []int{46, 47, 18, 2, 53, 8, 23, 32, 15, 50, 10, 31, 58, 3, 45, 35, 27, 43, 5, 49, 33, 9, 42, 19, 29, 28, 14, 39, 12, 38, 41, 13, 37, 48, 7, 16, 24, 55, 40, 61, 26, 17, 0, 1, 60, 51, 30, 4, 22, 25, 54, 21, 56, 59, 6, 63, 57, 62, 11, 36, 20, 34, 44, 52}

func MixinKey(source string) string {
	var result strings.Builder
	for _, index := range mixin {
		if index < len(source) {
			result.WriteByte(source[index])
		}
		if result.Len() == 32 {
			break
		}
	}
	return result.String()
}
func Sign(input url.Values, key string, unix int64) url.Values {
	values := url.Values{}
	for k, list := range input {
		for _, value := range list {
			value = strings.Map(func(r rune) rune {
				if strings.ContainsRune("!'()*", r) {
					return -1
				}
				return r
			}, value)
			values.Add(k, value)
		}
	}
	values.Set("wts", strconv.FormatInt(unix, 10))
	query := strings.ReplaceAll(values.Encode(), "+", "%20")
	digest := md5.Sum([]byte(query + key))
	values.Set("w_rid", hex.EncodeToString(digest[:]))
	return values
}
func (c *Client) wbiKey(ctx context.Context) (string, error) {
	c.mu.RLock()
	key, at := c.wbi, c.wbiAt
	c.mu.RUnlock()
	if key != "" && time.Since(at) < 6*time.Hour {
		return key, nil
	}
	data, err := c.Request(ctx, "GET", "/x/web-interface/nav", nil, false)
	if err != nil {
		return "", err
	}
	imgs := Map(data["wbi_img"])
	stem := func(raw string) string {
		u, _ := url.Parse(raw)
		if u == nil {
			return ""
		}
		name := u.Path[strings.LastIndex(u.Path, "/")+1:]
		if i := strings.Index(name, "."); i >= 0 {
			name = name[:i]
		}
		return name
	}
	key = MixinKey(stem(String(imgs["img_url"])) + stem(String(imgs["sub_url"])))
	if len(key) != 32 {
		return "", fmt.Errorf("无法取得 Bilibili 请求签名，请稍后重试")
	}
	c.mu.Lock()
	c.wbi = key
	c.wbiAt = time.Now()
	c.mu.Unlock()
	return key, nil
}

func Map(value any) map[string]any {
	if m, ok := value.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}
func List(value any) []any {
	if list, ok := value.([]any); ok {
		return list
	}
	return []any{}
}
func String(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}
func Int(value any) int64 {
	switch v := value.(type) {
	case json.Number:
		i, _ := v.Int64()
		return i
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	case string:
		i, _ := strconv.ParseInt(v, 10, 64)
		return i
	}
	return 0
}
func Number(value any) float64 {
	switch v := value.(type) {
	case json.Number:
		n, _ := v.Float64()
		return n
	case float64:
		return v
	default:
		n, _ := strconv.ParseFloat(String(value), 64)
		return n
	}
}
func URL(raw string) string {
	if strings.HasPrefix(raw, "//") {
		return "https:" + raw
	}
	return strings.Replace(raw, "http://", "https://", 1)
}
