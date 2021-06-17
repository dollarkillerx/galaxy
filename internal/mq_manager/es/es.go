package es

import "github.com/dollarkillerx/galaxy/pkg"

type ES struct {
}

func (E *ES) InitMQ(cfg pkg.Task) error {
	panic("implement me")
}

func (E *ES) SendMSG(event pkg.MQEvent) error {
	panic("implement me")
}

func (E *ES) Close() error {
	panic("implement me")
}
