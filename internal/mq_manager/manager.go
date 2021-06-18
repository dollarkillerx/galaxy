package mq_manager

import (
	"fmt"

	"github.com/dollarkillerx/galaxy/internal/mq_manager/es"
	"github.com/dollarkillerx/galaxy/internal/mq_manager/kafka"
	"github.com/dollarkillerx/galaxy/internal/mq_manager/mongodb"
	"github.com/dollarkillerx/galaxy/internal/mq_manager/nsq"
	"github.com/dollarkillerx/galaxy/pkg"
	"github.com/pkg/errors"
)

type manager struct {
	mq map[string]MQ
}

var Manager = manager{
	mq: map[string]MQ{},
}

// TODO: 处理重复问题
func (m *manager) Register(task pkg.Task) error {
	_, ex := m.mq[task.TaskBaseData.TaskID]
	if ex {
		return errors.New("Task ID already exists")
	}

	switch {
	case task.KafkaConf != nil:
		kfk := kafka.Kafka{}
		err := kfk.InitMQ(task)
		if err != nil {
			return errors.WithStack(err)
		}

		m.mq[task.TaskBaseData.TaskID] = &kfk
	case task.NsqConf != nil:
		nq := nsq.NSQ{}
		err := nq.InitMQ(task)
		if err != nil {
			return errors.WithStack(err)
		}

		m.mq[task.TaskBaseData.TaskID] = &nq
	case task.MongoDBConf != nil:
		mongo := mongodb.MongoDB{}
		err := mongo.InitMQ(task)
		if err != nil {
			return errors.WithStack(err)
		}

		m.mq[task.TaskBaseData.TaskID] = &mongo
	case task.ESConf != nil:
		e := es.ES{}
		err := e.InitMQ(task)
		if err != nil {
			return errors.WithStack(err)
		}

		m.mq[task.TaskBaseData.TaskID] = &e
	default:
		return errors.New("MQ CONFIG ERROR")
	}

	return nil
}

func (m *manager) Get(taskID string) (MQ, error) {
	mq, ex := m.mq[taskID]
	if !ex {
		return nil, errors.New(fmt.Sprintf("not exits: %s", taskID))
	}

	return mq, nil
}

func (m *manager) Close(taskID string) error {
	mq, ex := m.mq[taskID]
	if !ex {
		return errors.New(fmt.Sprintf("not exits: %s", taskID))
	}

	err := mq.Close()
	if err != nil {
		return err
	}

	delete(m.mq, taskID)
	return nil
}
