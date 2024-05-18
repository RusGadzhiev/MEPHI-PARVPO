package httpServer

import (
	"context"
	"net/http"
	"time"

	"github.com/RusGadzhiev/MEPHI-PARVPO/internal/config"
	"github.com/RusGadzhiev/MEPHI-PARVPO/internal/transport/http/httpHandler"
	"github.com/RusGadzhiev/MEPHI-PARVPO/pkg/logger"
)

type HttpServer struct {
	server http.Server
}

func NewHttpServer(ctx context.Context, h *httpHandler.HttpHandler, cfg config.Server) *HttpServer {
	return &HttpServer{
		server: http.Server{
			Addr:         ":" + cfg.Port,
			ReadTimeout:  cfg.Timeout,
			WriteTimeout: cfg.Timeout,
			IdleTimeout:  cfg.IdleTimeout,
			Handler:      h.Router(),
		},
	}
}

func (s *HttpServer) Run(ctx context.Context) error {
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Listen error: ", err)
		}
	}()
	logger.Info("Start listen http server at " + s.server.Addr)

	<-ctx.Done()
	logger.Info("Gracefully stopping...")
	
	shtCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := s.server.Shutdown(shtCtx)
	return err

}
