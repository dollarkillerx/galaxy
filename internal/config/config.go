package config

type conf struct {
	ListenAddr string `yaml:"ListenAddr"`
}

var Conf *conf

func InitConfig() error {
	Conf = &conf{ListenAddr: "0.0.0.0:8089"}
	return nil
}
