package mq_manager

import "github.com/dollarkillerx/galaxy/pkg"

type MQ interface {
	InitMQ(cfg pkg.Task) error
	SendMSG(event []pkg.MQEvent) error
	Close() error
}
