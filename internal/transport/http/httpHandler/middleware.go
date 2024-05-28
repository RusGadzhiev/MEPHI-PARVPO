package httpHandler

import (
	"net/http"
	"time"

	"github.com/RusGadzhiev/MEPHI-PARVPO/pkg/logger"
)

func (h *HttpHandler) LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		logger.Info("New request: ", "method - ", r.Method, " remote_addr - ", r.RemoteAddr, " url - ", r.URL.Path)
		logger.Infof("Time to response: %v", time.Since(start))
	})
}

func (h *HttpHandler) PanicRecoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				logger.Error("Url: ", r.URL.Path, " Recovered: ", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
