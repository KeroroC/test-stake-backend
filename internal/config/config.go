package config

import (
	"log"

	"github.com/spf13/viper"
)

type Config struct {
	Server    Server    `mapstructure:"server"`
	Database  Database  `mapstructure:"database"`
	Redis     Redis     `mapstructure:"redis"`
	ETHConfig ETHConfig `mapstructure:"eth"`
}

type Server struct {
	Port string `mapstructure:"port"`
	Host string `mapstructure:"host"`
	Mode string `mapstructure:"mode"`
}

type Database struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
}

type Redis struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type ETHConfig struct {
	RPCUrl       string `mapstructure:"rpc_url"`
	WSUrl        string `mapstructure:"ws_url"`
	StakeAddress string `mapstructure:"stake_address"`
}

func Load() *Config {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		log.Fatalf("Error parsing config file, %s", err)
	}

	return &config
}
