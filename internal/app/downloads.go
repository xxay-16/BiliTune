package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"github.com/xxay-16/BiliTune/internal/bili"
	"github.com/xxay-16/BiliTune/internal/model"
)

func (s *Service) Downloads() []model.Download { return s.snapshot().Downloads }
func (s *Service) AddDownload(track model.Track) error {
	if track.ID == "" {
		return fmt.Errorf("没有可下载的歌曲")
	}
	sum := sha256.Sum256([]byte(track.ID))
	id := hex.EncodeToString(sum[:8])
	s.mu.Lock()
	for _, task := range s.state.Downloads {
		if task.ID == id || task.Track.ID == track.ID {
			s.mu.Unlock()
			return s.DownloadAction(task.ID, "resume")
		}
	}
	s.state.Downloads = append([]model.Download{{ID: id, Track: track, Status: "queued"}}, s.state.Downloads...)
	s.mu.Unlock()
	s.persist()
	s.scheduleDownloads()
	return nil
}
func (s *Service) DownloadAction(id, action string) error {
	s.downloadMu.Lock()
	cancel := s.downloadCancels[id]
	s.downloadMu.Unlock()
	s.mu.Lock()
	found := false
	for i := range s.state.Downloads {
		if s.state.Downloads[i].ID != id {
			continue
		}
		found = true
		task := &s.state.Downloads[i]
		switch action {
		case "pause":
			if task.Status != "completed" {
				task.Status = "paused"
			}
		case "resume":
			if task.Status != "completed" {
				task.Status = "queued"
				task.Error = ""
			}
		case "remove":
			s.state.Downloads = append(s.state.Downloads[:i], s.state.Downloads[i+1:]...)
		default:
			s.mu.Unlock()
			return fmt.Errorf("无效的下载操作")
		}
		break
	}
	s.mu.Unlock()
	if !found {
		return fmt.Errorf("下载任务不存在")
	}
	if cancel != nil && (action == "pause" || action == "remove") {
		cancel()
	}
	s.persist()
	if action == "resume" {
		s.scheduleDownloads()
	}
	return nil
}
func (s *Service) scheduleDownloads() {
	if s.ctx.Err() != nil {
		return
	}
	s.downloadMu.Lock()
	defer s.downloadMu.Unlock()
	if s.ctx.Err() != nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	limit := s.state.Settings.DownloadConcurrency
	if limit < 1 {
		limit = 2
	}
	for i := range s.state.Downloads {
		if len(s.downloadWorkers) >= limit {
			return
		}
		task := s.state.Downloads[i]
		if task.Status != "queued" || s.downloadWorkers[task.ID] {
			continue
		}
		ctx, cancel := context.WithCancel(s.ctx)
		s.downloadWorkers[task.ID] = true
		s.downloadCancels[task.ID] = cancel
		s.state.Downloads[i].Status = "downloading"
		s.workers.Add(1)
		go s.download(ctx, task)
	}
}
func (s *Service) updateDownload(id string, fn func(*model.Download)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.state.Downloads {
		if s.state.Downloads[i].ID == id {
			fn(&s.state.Downloads[i])
			return
		}
	}
}
func safeName(raw string) string {
	runes := []rune(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || strings.ContainsRune(`<>:"/\|?*`, r) {
			return '-'
		}
		return r
	}, raw))
	if len(runes) > 100 {
		runes = runes[:100]
	}
	result := strings.Trim(string(runes), " .")
	if result == "" {
		return "BiliTune"
	}
	return result
}
func (s *Service) download(ctx context.Context, task model.Download) {
	defer s.workers.Done()
	defer func() {
		s.downloadMu.Lock()
		delete(s.downloadWorkers, task.ID)
		delete(s.downloadCancels, task.ID)
		s.downloadMu.Unlock()
		s.persist()
	}()
	err := s.fetchDownload(ctx, task)
	if err != nil {
		s.updateDownload(task.ID, func(item *model.Download) {
			if item.Status == "queued" && ctx.Err() != nil {
				return
			}
			if item.Status == "paused" || ctx.Err() != nil {
				item.Status = "paused"
				item.Error = ""
			} else {
				item.Status = "failed"
				item.Error = err.Error()
			}
		})
	}
}
func (s *Service) fetchDownload(ctx context.Context, task model.Download) error {
	settings := s.snapshot().Settings
	_, source, err := s.client.Resolve(ctx, task.Track, settings.AudioQuality)
	if err != nil {
		return err
	}
	directory := settings.DownloadDir
	if directory == "" {
		home, _ := os.UserHomeDir()
		directory = filepath.Join(home, "Downloads", "BiliTune")
	}
	if err = os.MkdirAll(directory, 0700); err != nil {
		return err
	}
	extension := ".m4a"
	if source.Quality == "FLAC" {
		extension = ".flac"
	}
	path := filepath.Join(directory, safeName(task.Track.Title)+"-"+safeName(task.ID)+extension)
	if task.Path != "" && filepath.IsAbs(task.Path) {
		path = task.Path
	}
	partial := path + ".part"
	file, err := os.OpenFile(partial, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	offset := info.Size()
	req, err := http.NewRequestWithContext(ctx, "GET", source.URL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", bili.UserAgent)
	req.Header.Set("Referer", bili.Referer)
	if offset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}
	client := &http.Client{Transport: http.DefaultTransport}
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != 200 && response.StatusCode != 206 {
		return fmt.Errorf("下载服务器 HTTP %d", response.StatusCode)
	}
	if response.StatusCode == 200 && offset > 0 {
		if err = file.Truncate(0); err != nil {
			return err
		}
		offset = 0
	}
	if response.StatusCode == 206 {
		var start, end, total int64
		if _, e := fmt.Sscanf(response.Header.Get("Content-Range"), "bytes %d-%d/%d", &start, &end, &total); e != nil || start != offset {
			return fmt.Errorf("下载服务器返回了无效的续传位置")
		}
	}
	if _, err = file.Seek(offset, io.SeekStart); err != nil {
		return err
	}
	total := response.ContentLength
	if total > 0 {
		total += offset
	}
	s.updateDownload(task.ID, func(item *model.Download) { item.Path = path; item.Received = offset; item.Total = total })
	buffer := make([]byte, 64*1024)
	received := offset
	for {
		n, readErr := response.Body.Read(buffer)
		if n > 0 {
			written, writeErr := file.Write(buffer[:n])
			if writeErr != nil {
				return writeErr
			}
			received += int64(written)
			s.updateDownload(task.ID, func(item *model.Download) { item.Received = received })
			if written != n {
				return io.ErrShortWrite
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
	if total > 0 && received != total {
		return fmt.Errorf("下载未完成：%s / %s 字节", strconv.FormatInt(received, 10), strconv.FormatInt(total, 10))
	}
	if err = file.Sync(); err != nil {
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	if err = os.Rename(partial, path); err != nil {
		return err
	}
	s.updateDownload(task.ID, func(item *model.Download) {
		item.Status = "completed"
		item.Path = path
		item.Received = received
		item.Total = received
		item.Track.LocalPath = path
		item.Error = ""
	})
	return nil
}

func (s *Service) RevealDownload(id string) error {
	for _, task := range s.Downloads() {
		if task.ID == id && task.Path != "" {
			s.mu.RLock()
			reveal := s.desktop.Reveal
			s.mu.RUnlock()
			if reveal != nil {
				return reveal(task.Path)
			}
		}
	}
	return fmt.Errorf("下载文件暂不可用")
}
