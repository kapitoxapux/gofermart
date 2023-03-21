package config

import (
	"flag"
	"os"
)

type Config struct {
	ServerAddress  string `env:"RUN_ADDRESS"`
	AccrualAddress string `env:"ACCRUAL_SYSTEM_ADDRESS"`
	DBAddress      string `env:"DATABASE_URI"`
}

var ServerConfig Config
var Secretkey = []byte("Diplom_Go-13_Info")

func SetConfig() Config {
	addr := flag.String("a", "localhost:8080", "RUN_ADDRESS")
	base := flag.String("d", "", "ACCRUAL_SYSTEM_ADDRESS")
	db := flag.String("r", "host=localhost port=5432 user=postgres password= dbname=postgres sslmode=disable", "DATABASE_URI")
	flag.Parse()

	if serverAddress := os.Getenv("RUN_ADDRESS"); serverAddress == "" {
		ServerConfig.ServerAddress = *addr
	} else {
		ServerConfig.ServerAddress = serverAddress
	}

	if accrualAddress := os.Getenv("ACCRUAL_SYSTEM_ADDRESS"); accrualAddress == "" {
		ServerConfig.AccrualAddress = *base
	} else {
		ServerConfig.AccrualAddress = accrualAddress
	}

	if dbAddress := os.Getenv("DATABASE_URI"); dbAddress == "" {
		ServerConfig.DBAddress = *db
	} else {
		ServerConfig.DBAddress = dbAddress
	}

	return ServerConfig
}

func GetConfigServerAddress() string {

	return ServerConfig.ServerAddress
}

func GetConfigAccrualAddress() string {

	return ServerConfig.AccrualAddress
}

func GetConfigDBAddress() string {

	return ServerConfig.DBAddress
}

func GetConfigPath() string {

	return "logger.log"
}
