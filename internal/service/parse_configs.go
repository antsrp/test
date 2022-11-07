package service

import (
	"fmt"
	"os"

	"github.com/antsrp/balance_service/internal/postgres"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

func ParseDBConfig(logger *zap.Logger) *postgres.PSQLConfig {
	f, err := os.Open(fmt.Sprintf("%s\\%s", getPathToConfigsFolder(), "db_config.yaml"))
	if err != nil {
		logger.Sugar().Fatal("Can't read config of db: ", err)
	}
	defer f.Close()

	var cfg postgres.PSQLConfig

	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if err != nil {
		logger.Sugar().Fatal("Can't parse config of db: ", err)
	}

	return &cfg
}
