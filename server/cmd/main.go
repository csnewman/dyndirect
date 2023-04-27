package main

import (
	"go.uber.org/zap"
)

func main() {
	rawLogger, _ := zap.NewDevelopment()
	defer rawLogger.Sync()

	logger := rawLogger.Sugar()
	logger.Infow("DynDirect Server")
}
