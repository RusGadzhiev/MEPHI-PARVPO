package kafka

import (
	"sync"

	"github.com/IBM/sarama"
	"github.com/RusGadzhiev/MEPHI-PARVPO/pkg/logger"
	"github.com/RusGadzhiev/MEPHI-PARVPO/internal/config"
)

const (
	RequestsTopic = "requests"
	ResponseTopic = "responses"
)

type ApiBroker struct {
	ResponseChannels map[string]chan *sarama.ConsumerMessage
	Producer         sarama.SyncProducer
	Consumer         sarama.Consumer
	PartConsumer     sarama.PartitionConsumer
}


func StartApiBroker(cfg config.Kafka) (*ApiBroker, func()) {
	responseChannels := make(map[string]chan *sarama.ConsumerMessage, 100)
	var mu sync.Mutex

	producer, err := sarama.NewSyncProducer([]string{cfg.Addr}, nil)
	if err != nil {
		logger.Fatalf("Failed to create producer: %v", err)
	}

	consumer, err := sarama.NewConsumer([]string{cfg.Addr}, nil)
	if err != nil {
		logger.Fatalf("Failed to create consumer: %v", err)
	}

	partConsumer, err := consumer.ConsumePartition(ResponseTopic, 0, sarama.OffsetNewest)
	if err != nil {
		logger.Fatalf("Failed to consume partition: %v", err)
	}

	// Горутина для обработки входящих сообщений от Kafka
	go func() {
		for msg := range partConsumer.Messages(){
			responseID := string(msg.Key)
			mu.Lock()
			ch, exists := responseChannels[responseID]
			if exists {
				ch <- msg
				// delete(responseChannels, responseID)
			}
			mu.Unlock()
		}
		logger.Infof("Channel closed, exiting goroutine")
	}()

	return &ApiBroker{
		Producer: producer,
		Consumer: consumer,
		PartConsumer: partConsumer,
		ResponseChannels: responseChannels,
	}, func() {
		producer.Close()
		consumer.Close()
		partConsumer.Close()
	}
}
