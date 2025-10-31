package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	App struct {
		Env  string
		Port string
	}
	// ...other configurations...
}

func LoadConfig() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	cfg := &Config{}
	// ...map environment variables to config struct...
	return cfg, nil
}
