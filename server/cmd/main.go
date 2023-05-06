package main

import (
	"github.com/csnewman/dyndirect/server"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func main() {
	rawLogger, _ := zap.NewDevelopment()

	//nolint:errcheck
	defer rawLogger.Sync()

	logger := rawLogger.Sugar()
	logger.Infow("DynDirect Server")

	var cfg server.Config

	viper.SetConfigFile("config.yml")

	if err := viper.ReadInConfig(); err != nil {
		logger.Fatalw("Config error", "err", err)
	}

	err := viper.Unmarshal(&cfg)
	if err != nil {
		logger.Fatalw("Config error", "err", err)
	}

	var store server.Store

	if cfg.Store == "mem" {
		mem, err := server.NewMemStore(logger)
		if err != nil {
			logger.Fatal(err)
		}

		go mem.AutoCleanup()

		store = mem
	} else if cfg.Store == "redis" {
		store = server.NewRedisStore(cfg)
	} else {
		logger.Fatalw("Invalid store provided", "store", cfg.Store)
	}

	s := server.New(logger, cfg, store)

	if err := s.Start(); err != nil {
		logger.Fatal(err)
	}
}
