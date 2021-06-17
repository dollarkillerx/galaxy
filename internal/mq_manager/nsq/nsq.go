package nsq

import "github.com/dollarkillerx/galaxy/pkg"

type NSQ struct {
}

func (N *NSQ) InitMQ(cfg pkg.Task) error {
	panic("implement me")
}

func (N *NSQ) SendMSG(event pkg.MQEvent) error {
	panic("implement me")
}

func (N *NSQ) Close() error {
	panic("implement me")
}
