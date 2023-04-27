package main

import (
	"github.com/csnewman/dyndirect/server"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func main() {
	rawLogger, _ := zap.NewDevelopment()
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

	s := server.New(logger, cfg)

	if err := s.Start(); err != nil {
		logger.Fatal(err)
	}

}
