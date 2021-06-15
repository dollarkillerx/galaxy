package kafka

import "github.com/dollarkillerx/galaxy/pkg"

type Kafka struct {
}

func (k *Kafka) InitMQ(cfg pkg.Task) error {
	panic("implement me")
}

func (k *Kafka) SendMSG(event pkg.MQEvent) error {
	panic("implement me")
}

func (k *Kafka) Close() {
	panic("implement me")
}
