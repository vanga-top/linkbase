package config

import "time"

const (
	HighPriority   = 1
	NormalPriority = HighPriority + 10
	LowPriority    = NormalPriority + 10
)

type Source interface {
	GetConfigurations() (map[string]string, error)
	GetConfigurationByKey(string) (string, error)
	GetPriority() int
	GetSourceName() string
	SetEventHandler(eh EventHandler)
	UpdateOptions(opt Options)
	Close()
}

// EtcdInfo has attribute for config center source initialization
type EtcdInfo struct {
	UseEmbed   bool
	UseSSL     bool
	Endpoints  []string
	KeyPrefix  string
	CertFile   string
	KeyFile    string
	CaCertFile string
	MinVersion string

	//Pull Configuration interval, unit is second
	RefreshInterval time.Duration
}

// FileInfo has attribute for file source
type FileInfo struct {
	Files           []string
	RefreshInterval time.Duration
}

// Options hold options
type Options struct {
	FileInfo        *FileInfo
	EtcdInfo        *EtcdInfo
	EnvKeyFormatter func(string) string
}

// Option is a func
type Option func(options *Options)

// WithRequiredFiles tell archaius to manage files, if not exist will return error
func WithFilesSource(fi *FileInfo) Option {
	return func(options *Options) {
		options.FileInfo = fi
	}
}

// WithEtcdSource accept the information for initiating a remote source
func WithEtcdSource(ri *EtcdInfo) Option {
	return func(options *Options) {
		options.EtcdInfo = ri
	}
}

// WithEnvSource enable env source
// archaius will read ENV as key value
func WithEnvSource(keyFormatter func(string) string) Option {
	return func(options *Options) {
		options.EnvKeyFormatter = keyFormatter
	}
}

// EventHandler handles config change event
type EventHandler interface {
	OnEvent(event *Event)
	GetIdentifier() string
}

type simpleHandler struct {
	identity string
	onEvent  func(*Event)
}
