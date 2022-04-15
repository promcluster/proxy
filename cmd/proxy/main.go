package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/promcluster/proxy/api"
	"github.com/promcluster/proxy/config"
	"github.com/promcluster/proxy/pkg/backend"
	pkgc "github.com/promcluster/proxy/pkg/consumer"
	"github.com/promcluster/proxy/pkg/filter"
	"github.com/promcluster/proxy/pkg/log"
	pkgq "github.com/promcluster/proxy/pkg/queue"
	"github.com/promcluster/proxy/service/worker"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/version"
	"github.com/spf13/viper"
	"go.uber.org/ratelimit"
)

var configFile string

func init() {
	prometheus.MustRegister(version.NewCollector("proxy"))
	flag.StringVar(&configFile, "config", "", "config file")
	flag.Parse()
	if err := praseConfig(configFile); err != nil {
		panic(err)
	}
}

func main() {
	// init zap logger
	logger, err := log.InitLogger(config.C.Log)
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	reg := prometheus.DefaultRegisterer
	promBackend := backend.NewPromServer(
		ctx,
		reg,
		viper.GetString("SD.name"),
		viper.GetInt("worker.num"),
		time.Duration(viper.GetInt("SD.refreshInterval"))*time.Second,
		logger)

	var queue pkgq.Queue
	if viper.GetString("queue.type") == "memory" {
		queue = pkgq.NewChanQueue(reg, logger)
	} else {
		queue = pkgq.NewDiskQueue(reg, config.C.Queue, logger)
	}

	lf, err := filter.NewMetricsFilter(reg,
		viper.GetUint64("api.maxSeriesCountLimit"),
		viper.GetDuration("api.seriesCountFlushInterval"))
	if err != nil {
		panic(err)
	}
	consumer := pkgc.NewRemoteConsumer(ctx, reg, promBackend, []filter.Filter{lf}, logger)
	err = worker.StartWorkers(ctx, reg, viper.GetInt("worker.num"), queue, consumer, logger)
	if err != nil {
		panic(err)
	}

	limiter := ratelimit.New(viper.GetInt("api.rateLimit"))
	service, err := api.New(reg, config.C.API, queue, limiter, logger)
	if err != nil {
		panic(err)
	}
	err = service.Start(ctx)
	if err != nil {
		panic(err)
	}

	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, syscall.SIGTERM, syscall.SIGINT)
	<-terminate
	service.Close(ctx)
	cancel()
	_ = logger.Sync()
}

func praseConfig(configFile string) error {
	viper.SetEnvPrefix("promucluster")
	viper.AutomaticEnv()
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)

	if configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		viper.SetConfigName("config")
		viper.AddConfigPath(".")
	}
	if err := viper.ReadInConfig(); err != nil {
		return err
	}
	return viper.Unmarshal(&config.C)
}
