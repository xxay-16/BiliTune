package player

type Snapshot struct {
	Playing   bool
	Position  float64
	Duration  float64
	Buffered  float64
	Buffering bool
	Ended     bool
	Error     string
}

type Engine interface {
	Load(string, map[string]string, float64) error
	Play() error
	Pause() error
	Stop() error
	Seek(float64) error
	SetVolume(float64) error
	SetRate(float64) error
	Snapshot() Snapshot
	Close()
}
