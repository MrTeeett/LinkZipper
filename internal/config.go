package internal

import (
	"github.com/spf13/viper"
	"log"
)

type ServerConfig struct {
	Port int `mapstructure:"port"`
}

type LimitsConfig struct {
	MaxTasks        int      `mapstructure:"maxTasks"`
	MaxFilesPerTask int      `mapstructure:"maxFilesPerTask"`
	AllowedExts     []string `mapstructure:"allowedExtensions"`
}

type Config struct {
	Server ServerConfig `mapstructure:"server"`
	Limits LimitsConfig `mapstructure:"limits"`
}

func Load() *Config {
	viper.SetConfigFile("config.yaml")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config: %v", err)
	}
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatalf("Unable to decode into struct: %v", err)
	}
	return &cfg
}
