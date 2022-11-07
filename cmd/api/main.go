package main

import (
	"log"
	"os"

	"github.com/antsrp/balance_service/internal/postgres"
	"github.com/antsrp/balance_service/internal/service"
	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal("Can't create zap logger: ", err)
	}

	cfg := service.ParseDBConfig(logger)

	db, err := postgres.SQLConnect(cfg, logger)
	if err != nil {
		logger.Sugar().Fatal("Can't create db: ", err)
	}

	userStorage, err := postgres.CreateUserStorage(db)
	if err != nil {
		logger.Sugar().Fatal("Can't create a user storage", err)
	}
	defer handleCloser(logger, "user storage", userStorage)

	transactionStorage, err := postgres.CreateTransactionStorage(db, cfg.Limitations.PageLimit)
	if err != nil {
		logger.Sugar().Fatal("Can't create a user storage", err)
	}
	defer handleCloser(logger, "reservation storage", transactionStorage)

	serv := service.CreateNewService(userStorage, transactionStorage)

	h, err := createNewHandler(logger, serv)
	if err != nil {
		logger.Sugar().Fatal("Can't create a new handler", err)
	}
	r := h.Routes()

	startServer(logger, r, make(chan os.Signal))
}
