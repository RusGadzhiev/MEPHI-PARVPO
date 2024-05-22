package main

import (
	"context"
	"os/signal"
	"syscall"
	"time"

	"github.com/RusGadzhiev/MEPHI-PARVPO/internal/broker/kafka"
	"github.com/RusGadzhiev/MEPHI-PARVPO/internal/config"
	"github.com/RusGadzhiev/MEPHI-PARVPO/internal/service"
	"github.com/RusGadzhiev/MEPHI-PARVPO/internal/storage/postgres"
	"github.com/RusGadzhiev/MEPHI-PARVPO/internal/transport/http/httpHandler"
	"github.com/RusGadzhiev/MEPHI-PARVPO/internal/transport/http/httpServer"
	"github.com/RusGadzhiev/MEPHI-PARVPO/pkg/logger"
)

func main() {
	cfg := config.MustLoad()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	storage, err := postgres.NewPostgresStorage(ctx, cfg.PgDb)
	if err != nil {
		logger.Fatal(err)
	}
	logger.Info("Storage postgres started successfully")

	service := service.NewService(storage)

	time.Sleep(7 * time.Second)
	apiBroker, closeApi := kafka.StartApiBroker(cfg.Kafka)
	defer closeApi()

	closeService := kafka.StartServiceKafkaBroker(cfg.Kafka, service)
	defer closeService()

	httpHandler := httpHandler.NewHttpHandler(*apiBroker)
	server := httpServer.NewHttpServer(ctx, httpHandler, cfg.Server)

	if err := server.Run(ctx); err != nil {
		logger.Fatal(err)
	}
}
