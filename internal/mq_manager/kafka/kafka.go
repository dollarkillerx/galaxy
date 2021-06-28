package kafka

import (
	"encoding/json"
	"fmt"
	"log"
	"runtime"
	"sync"

	"github.com/Shopify/sarama"
	"github.com/dollarkillerx/async_utils"
	"github.com/dollarkillerx/galaxy/pkg"
	"github.com/pingcap/errors"
)

type Kafka struct {
	cfg         pkg.Task
	producer    sarama.SyncProducer
	taskChannel chan pkg.MQEvent
	closeTask   chan struct{}

	poolSize int
}

func (k *Kafka) InitMQ(cfg pkg.Task) error {
	kafkaConf := sarama.NewConfig()
	if cfg.KafkaConf.EnableSASL {
		kafkaConf.Net.SASL.Enable = true
		kafkaConf.Net.SASL.Mechanism = sarama.SASLTypePlaintext
		kafkaConf.Net.SASL.User = cfg.KafkaConf.User
		kafkaConf.Net.SASL.Password = cfg.KafkaConf.Password
	}
	kafkaConf.Producer.Retry.Max = 5
	kafkaConf.Producer.RequiredAcks = sarama.WaitForAll
	kafkaConf.Producer.Return.Successes = true
	kafkaConf.Producer.Partitioner = sarama.NewRandomPartitioner

	producer, err := sarama.NewSyncProducer(cfg.KafkaConf.Brokers, kafkaConf)
	if err != nil {
		return errors.WithStack(err)
	}

	k.cfg = cfg
	k.producer = producer
	k.taskChannel = make(chan pkg.MQEvent, 1000)
	k.closeTask = make(chan struct{})
	k.poolSize = runtime.NumCPU() * 4

	go k.core()
	return nil
}

func (k *Kafka) core() {
	defer func() {
		err := k.producer.Close()
		if err != nil {
			log.Println(err)
		}
	}()

loop:
	for {
		select {
		case <-k.closeTask:
			break loop
		case event, ex := <-k.taskChannel:
			if !ex {
				break loop
			}
			marshal, err := json.Marshal(event)
			if err != nil {
				log.Println(err)
				continue
			}

			_, _, err = k.producer.SendMessage(&sarama.ProducerMessage{
				Topic: fmt.Sprintf("%s.%s.%s", k.cfg.TaskID, event.Database, event.Table),
				Key:   sarama.ByteEncoder(fmt.Sprintf("%s.%s", event.Database, event.Table)),
				Value: sarama.ByteEncoder(marshal),
			})
			if err != nil {
				log.Println(err)
			}
		}
	}
}

func (k *Kafka) SendMSG(event []pkg.MQEvent) error {
	//k.taskChannel <- event
	msg := k.packageMSG(event)
	if len(msg) == 0 {
		return nil
	}
	return k.producer.SendMessages(msg)
}

func (k *Kafka) packageMSG(event []pkg.MQEvent) []*sarama.ProducerMessage {
	var result []*sarama.ProducerMessage
	var mu sync.Mutex

	over := make(chan struct{})
	poolFunc := async_utils.NewPoolFunc(k.poolSize, func() {
		close(over)
	})

	for i := range event {
		idx := i
		poolFunc.Send(func() {
			marshal, err := json.Marshal(event[idx])
			if err != nil {
				log.Println(err)
				return
			}

			mu.Lock()
			result = append(result, &sarama.ProducerMessage{
				Topic: fmt.Sprintf("%s.%s.%s", k.cfg.TaskID, event[idx].Database, event[idx].Table),
				Key:   sarama.ByteEncoder(fmt.Sprintf("%s.%s", event[idx].Database, event[idx].Table)),
				Value: sarama.ByteEncoder(marshal),
			})
			mu.Unlock()
		})
	}
	poolFunc.Over()
	<-over
	return result
}

func (k *Kafka) Close() error {
	close(k.closeTask)
	return nil
}
