package kafka

import (
	"context"
	"encoding/json"
	"time"

	"github.com/IBM/sarama"
	"github.com/RusGadzhiev/MEPHI-PARVPO/internal/config"
	"github.com/RusGadzhiev/MEPHI-PARVPO/internal/service"
	"github.com/RusGadzhiev/MEPHI-PARVPO/pkg/logger"
)

type Message struct {
	ID    string         `json:"id"`
	Value service.Record `json:"value"`
}

type Service interface {
	AddRecord(ctx context.Context, record service.Record) error
}

func StartServiceKafkaBroker(cfg config.Kafka, service Service) func() {
	producer, err := sarama.NewSyncProducer([]string{cfg.Addr}, nil)
	if err != nil {
		logger.Fatalf("Failed to create producer: %v", err)
	}

	consumer, err := sarama.NewConsumer([]string{cfg.Addr}, nil)
	if err != nil {
		logger.Fatalf("Failed to create consumer: %v", err)
	}

	partConsumer, err := consumer.ConsumePartition(RequestsTopic, 0, sarama.OffsetNewest)
	if err != nil {
		logger.Fatalf("Failed to consume partition: %v", err)
	}

	// кажется можно в отдельной горутине
	go func() {
		for {
			select {
			// (обработка входящего сообщения и отправка ответа в Kafka)
			case msg, ok := <-partConsumer.Messages():
				if !ok {
					logger.Infof("Channel closed, exiting goroutine")
					return
				}

				var receivedMessage Message
				err := json.Unmarshal(msg.Value, &receivedMessage)

				if err != nil {
					logger.Infof("Error unmarshaling JSON: %v", err)
					continue
				}

				logger.Infof("Received message: %+v\n", receivedMessage)

				ctx, cancel := context.WithTimeout(context.Background(), 10 * time.Second)
				err = service.AddRecord(ctx, receivedMessage.Value)
				cancel()
				var respMsg string
				if err != nil {
					respMsg = err.Error()
				} else {
					respMsg = "Place reserved"
				}

				bytes, err := json.Marshal(respMsg)
				if err != nil {
					logger.Infof("Failed to marshal respomse message to Kafka: %v", err)
					continue
				}

				// Формируем ответное сообщение
				resp := &sarama.ProducerMessage{
					Topic: ResponseTopic,
					Key:   sarama.StringEncoder(receivedMessage.ID),
					Value: sarama.ByteEncoder(bytes),
				}

				_, _, err = producer.SendMessage(resp)
				if err != nil {
					logger.Infof("Failed to send message to Kafka: %v", err)
				}
			}
		}
	}()

	return func() {
		producer.Close()
		consumer.Close()
		partConsumer.Close()
	}
}
