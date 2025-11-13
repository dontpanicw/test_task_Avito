package main

import (
	"go.uber.org/zap"
	"test_task_avito/backend/config"
	"test_task_avito/backend/internal/app"
)

func main() {
	//create logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic("failed to create logger: " + err.Error())
	}
	defer logger.Sync()

	cfg, err := config.NewConfig(logger)
	if err != nil {
		logger.Fatal("error creating config", zap.Error(err))
	}

	if err := app.Start(cfg, logger); err != nil {
		logger.Fatal("failed to start application", zap.Error(err))
	}
}
