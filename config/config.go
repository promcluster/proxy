package config

import (
	"time"

	"github.com/promcluster/proxy/pkg/log"
	"github.com/promcluster/proxy/pkg/queue"
)

// C is global configuration object
var C Configuration

// Configuration is global configuration struct
type Configuration struct {
	API    APIConfiguration    `yaml:"api"`
	SD     ServiceDiscovery    `yaml:"SD"`
	Worker WorkerConfiguration `yaml:"worker"`
	Queue  queue.Config        `yaml:"queue"`
	Auth   AuthConfiguration   `yaml:"auth"`
	Log    log.Config          `yaml:"log"`
}

type APIConfiguration struct { //nolint: maligned
	Listen                   string        `yaml:"listen"`
	MaxBodySizeLimit         int           `yaml:"maxBodySizeLimit"`
	Pprof                    bool          `yaml:"pprof"`
	RateLimit                int           `yaml:"rateLimit"`
	MaxSeriesCountLimit      uint64        `yaml:"maxSeriesCountLimit"`
	SeriesCountFlushInterval time.Duration `yaml:"seriesCountFlushInterval"`
	PushGatewayEnable        bool          `yaml:"pushGatewayEnable"`

	QueryEnable bool   `yaml:"queryEnable"`
	QueryAddr   string `yaml:"queryAddr"`
}

type ServiceDiscovery struct {
	Name            string `yaml:"name"`
	RefreshInterval int    `yaml:"refreshInterval"`
}

type WorkerConfiguration struct {
	Num int `yaml:"num"`
}

type AuthConfiguration struct {
	Enable bool   `yaml:"enable"`
	User   string `yaml:"user"`
	Token  string `yaml:"token"`
}
