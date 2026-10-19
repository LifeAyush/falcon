package parsing

import (
	"log"
	"sync"

	"github.com/BurntSushi/toml"
)

// Config is the root TOML configuration.
type Config struct {
	Core     Core     `toml:"core"`
	LimitReq LimitReq `toml:"limitReq"`
	Proxy    Proxy    `toml:"proxy"`
}

type Core struct {
	Listen            int  `toml:"listen"`
	ApiPort           int  `toml:"apiport"`
	LimitMaxConn      int  `toml:"limitMaxConn"`
	ReadTimeout       int  `toml:"readTimeout"`
	WriteTimeout      int  `toml:"writeTimeout"`
	IdleTimeout       int  `toml:"idleTimeout"`
	EnableServerStats bool `toml:"enableServerStats"`
	HealthIntervalMs  int  `toml:"healthIntervalMs"`
	HealthTimeoutMs   int  `toml:"healthTimeoutMs"`
}

type LimitReq struct {
	Enable    bool `toml:"enable"`
	Interval  int  `toml:"interval"`
	Frequency int  `toml:"frequency"`
	Mode      int  `toml:"mode"`
}

type Proxy struct {
	Routes []Route `toml:"routes"`
}

type Route struct {
	Endpoint string    `toml:"endpoint"`
	Match    int       `toml:"match"`
	Balancer int       `toml:"balancer"`
	Backends []Backend `toml:"backends"`
}

type Backend struct {
	Url        string `toml:"url"`
	VarName    string `toml:"varName"`
	Weight     uint32 `toml:"weight"`
	HealthPath string `toml:"healthPath"`
}

type File struct {
	FilePath string
}

var (
	config     Config
	once       sync.Once
	ConfigFile *File
)

func init() {
	ConfigFile = new(File)
}

func GetConfig() *Config {
	once.Do(func() {
		_, err := toml.DecodeFile(ConfigFile.FilePath, &config)
		if err != nil {
			log.Fatal(err)
		}
	})
	return &config
}

// DecodeConfigFile loads config from path (for tests). Resets once so it can be called again.
func ResetForTest() {
	once = sync.Once{}
	config = Config{}
}
