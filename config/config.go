package config

import (
	"os"

	"github.com/ONSdigital/go-ns/log"
	"github.com/ian-kent/gofigure"
)

// Config ...
type Config struct {
	gofigure     interface{} `order:"env"`
	BindAddr     string      `env:"BIND_ADDR"`
	AWSAccessKey string      `env:"AWS_ACCESS_KEY_ID"`
	AWSSecretKey string      `env:"AWS_SECRET_ACCESS_KEY"`
}

// Get ...
func Get() *Config {
	cfg := Config{
		BindAddr: ":3000",
	}

	if err := gofigure.Gofigure(&cfg); err != nil {
		log.Error(err, nil)
		os.Exit(1)
	}

	return &cfg
}
