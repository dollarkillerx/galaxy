package config

type conf struct {
	ListenAddr string `yaml:"ListenAddr"`
}

var Conf *conf

func InitConfig() error {
	Conf = &conf{}
	return nil
}
