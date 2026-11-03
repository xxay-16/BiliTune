package model

const Version = "0.2.1"

type Track struct {
	ID              string `json:"id"`
	Title           string `json:"title"`
	Artist          string `json:"artist"`
	DurationMS      int64  `json:"durationMs"`
	Type            string `json:"type"`
	CoverURL        string `json:"coverUrl,omitempty"`
	BVID            string `json:"bvid,omitempty"`
	AID             int64  `json:"aid,omitempty"`
	CID             int64  `json:"cid,omitempty"`
	AudioID         int64  `json:"audioId,omitempty"`
	SourceURL       string `json:"sourceUrl,omitempty"`
	SourceExpiresAt int64  `json:"sourceExpiresAt,omitempty"`
	PlayCount       int64  `json:"playCount,omitempty"`
	LocalPath       string `json:"localPath,omitempty"`
}

type Folder struct {
	ID       int64  `json:"id"`
	Title    string `json:"title"`
	Count    int    `json:"count"`
	CoverURL string `json:"coverUrl,omitempty"`
	Intro    string `json:"intro,omitempty"`
	Private  bool   `json:"private"`
}

type Account struct {
	MID    int64  `json:"mid"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
	VIP    bool   `json:"vip"`
}

type Hotkey struct {
	Action    string   `json:"action"`
	Key       string   `json:"key"`
	Modifiers []string `json:"modifiers"`
}

type Settings struct {
	Theme               string   `json:"theme"`
	CloseBehavior       string   `json:"closeBehavior"`
	AudioQuality        string   `json:"audioQuality"`
	LyricsSource        string   `json:"lyricsSource"`
	PlaybackRate        float64  `json:"playbackRate"`
	HistoryLimit        int      `json:"historyLimit"`
	DownloadDir         string   `json:"downloadDir"`
	DownloadConcurrency int      `json:"downloadConcurrency"`
	Hotkeys             []Hotkey `json:"hotkeys"`
	AutoStart           bool     `json:"autoStart"`
}

func DefaultSettings() Settings {
	return Settings{Theme: "light", CloseBehavior: "tray", AudioQuality: "auto", LyricsSource: "auto", PlaybackRate: 1, HistoryLimit: 100, DownloadConcurrency: 2, Hotkeys: []Hotkey{}}
}

type Player struct {
	Track        *Track  `json:"track"`
	Playing      bool    `json:"playing"`
	Position     float64 `json:"position"`
	Duration     float64 `json:"duration"`
	Buffered     float64 `json:"buffered"`
	Buffering    bool    `json:"buffering"`
	Volume       float64 `json:"volume"`
	Repeat       string  `json:"repeat"`
	Shuffle      bool    `json:"shuffle"`
	Quality      string  `json:"quality"`
	Error        string  `json:"error,omitempty"`
	QueueVersion int64   `json:"queueVersion"`
	QueueCount   int     `json:"queueCount"`
}

type Download struct {
	ID       string `json:"id"`
	Track    Track  `json:"track"`
	Status   string `json:"status"`
	Received int64  `json:"received"`
	Total    int64  `json:"total"`
	Path     string `json:"path,omitempty"`
	Error    string `json:"error,omitempty"`
}

type State struct {
	AppearanceVersion int        `json:"appearanceVersion"`
	Settings          Settings   `json:"settings"`
	History           []Track    `json:"history"`
	SearchHistory     []string   `json:"searchHistory"`
	HiddenFolders     []int64    `json:"hiddenFolders"`
	Downloads         []Download `json:"downloads"`
	Player            Player     `json:"player"`
	Queue             []Track    `json:"queue"`
}

type Bootstrap struct {
	Version       string            `json:"version"`
	Settings      Settings          `json:"settings"`
	Account       *Account          `json:"account"`
	Player        Player            `json:"player"`
	History       []Track           `json:"history"`
	SearchHistory []string          `json:"searchHistory"`
	Migrated      bool              `json:"migrated"`
	EngineReady   bool              `json:"engineReady"`
	EngineError   string            `json:"engineError,omitempty"`
	HotkeyErrors  map[string]string `json:"hotkeyErrors"`
}

type Source struct {
	URL       string
	Backups   []string
	Quality   string
	ExpiresAt int64
}

type Lyric struct {
	Time float64 `json:"time"`
	Text string  `json:"text"`
}
