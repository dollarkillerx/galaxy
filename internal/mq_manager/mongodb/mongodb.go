package mongodb

import "github.com/dollarkillerx/galaxy/pkg"

type MongoDB struct {
}

func (m *MongoDB) InitMQ(cfg pkg.Task) error {
	panic("implement me")
}

func (m *MongoDB) SendMSG(event pkg.MQEvent) error {
	panic("implement me")
}

func (m *MongoDB) Close() error {
	panic("implement me")
}
