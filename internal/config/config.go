package config

import (
	"os"
	"strings"
)

type conf struct {
	ListenAddr string `yaml:"ListenAddr"`
}

var Conf *conf

func InitConfig() error {
	defaultAddr := "0.0.0.0:8689"
	envAddr := os.Getenv("ListenAddr")
	if envAddr != "" {
		defaultAddr = strings.TrimSpace(envAddr)
	}

	Conf = &conf{ListenAddr: defaultAddr}
	return nil
}
