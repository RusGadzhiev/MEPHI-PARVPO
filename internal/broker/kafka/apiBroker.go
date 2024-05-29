package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/RusGadzhiev/MEPHI-PARVPO/internal/config"
	"github.com/RusGadzhiev/MEPHI-PARVPO/pkg/logger"
)

const (
	RequestsTopic = "requests"
	ResponseTopic = "responses"
)

type ApiBroker struct {
	mu               *sync.Mutex
	responseChannels map[string]chan *sarama.ConsumerMessage
	producer         sarama.SyncProducer
}

func (b *ApiBroker) Send(ctx context.Context, msg Message) (string, error) {
	bytes, err := json.Marshal(msg)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON err: %v", err)
	}

	message := &sarama.ProducerMessage{
		Topic: RequestsTopic,
		Key:   sarama.StringEncoder(msg.ID),
		Value: sarama.ByteEncoder(bytes),
	}

	_, _, err = b.producer.SendMessage(message)
	if err != nil {
		return "", fmt.Errorf("failed to send message to Kafka: %v", err)
	}

	responseCh := make(chan *sarama.ConsumerMessage)
	b.mu.Lock()
	b.responseChannels[msg.ID] = responseCh
	b.mu.Unlock()
	defer func() {
		b.mu.Lock()
		close(b.responseChannels[msg.ID])
		delete(b.responseChannels, msg.ID)
		b.mu.Unlock()
	}()

	select {
	case responseMsg := <-responseCh:
		var receivedMessage string
		err := json.Unmarshal(responseMsg.Value, &receivedMessage)
		if err != nil {
			return "", fmt.Errorf("error unmarshaling JSON: %v", err)
		}
		return receivedMessage, nil
	case <-time.After(4 * time.Second):
		return "", fmt.Errorf("timeout waiting for response")
	}
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
		for msg := range partConsumer.Messages() {
			responseID := string(msg.Key)
			mu.Lock()
			ch, exists := responseChannels[responseID]
			if exists {
				ch <- msg
			}
			mu.Unlock()
		}
		logger.Infof("Channel closed, exiting goroutine")
	}()

	return &ApiBroker{
			mu:               &mu,
			producer:         producer,
			responseChannels: responseChannels,
		}, func() {
			producer.Close()
			consumer.Close()
			partConsumer.Close()
		}
}
