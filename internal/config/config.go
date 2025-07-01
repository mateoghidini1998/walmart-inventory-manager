package config

import "os"

type Config struct {
	ServerAdress string
	DBHost       string
	DBPort       string
	DBUser       string
	DBPassword   string
	DBName       string
}

func NewConfig() *Config {
	return &Config{
		ServerAdress: os.Getenv("SERVER_ADDRESS"),
		DBHost: os.Getenv("DB_HOST"),
		DBPort: os.Getenv("DB_PORT"),
		DBUser: os.Getenv("DB_USER"),
		DBPassword: os.Getenv("DB_PASSWORD"),
		DBName: os.Getenv("DB_NAME"),
	}
}
