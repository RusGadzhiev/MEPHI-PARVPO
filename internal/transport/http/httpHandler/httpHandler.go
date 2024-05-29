package httpHandler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/RusGadzhiev/MEPHI-PARVPO/internal/broker/kafka"
	"github.com/RusGadzhiev/MEPHI-PARVPO/internal/service"
	"github.com/RusGadzhiev/MEPHI-PARVPO/pkg/logger"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type Broker interface {
	// отправляет запрос к сервису и возвращает ответное сообщение
	Send(ctx context.Context, msg kafka.Message) (string, error)
}

type HttpHandler struct {
	broker Broker
}

func (h *HttpHandler) AddRecord(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10 * time.Second)
	defer cancel()

	record := service.Record{
		Concert:  r.URL.Query().Get("concert"),
		Username: r.URL.Query().Get("user"),
	}

	if record.Concert == "" || record.Username == "" {
		logger.Warn("Request parameters are missing")
		h.clientError(w)
		return
	}

	requestID := uuid.New().String()

	message := kafka.Message{
		ID:    requestID,
		Value: record,
	}

	receivedMessage, err := h.broker.Send(ctx, message)
	if err != nil {
		logger.Errorf("Broker error: %w", err)
		h.serverError(w)
		return
	}

	logger.Infof("Response: %v", receivedMessage)
	w.WriteHeader(http.StatusOK)
	err = renderJSON(w, receivedMessage)
	if err != nil {
		logger.Errorf("RenderJson err: %w", err)
	}
}

// renderJSON преобразует 'v' в формат JSON и записывает результат, в виде ответа, в w.
func renderJSON(w http.ResponseWriter, v interface{}) error {
	json, err := json.Marshal(v)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(json)
	return err
}

func (h *HttpHandler) serverError(w http.ResponseWriter) {
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func (h *HttpHandler) clientError(w http.ResponseWriter) {
	http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
}

func NewHttpHandler(b Broker) *HttpHandler {
	return &HttpHandler{
		broker: b,
	}
}

func (h *HttpHandler) Router() *mux.Router {
	r := mux.NewRouter()
	r.StrictSlash(true)
	r.HandleFunc("/add-record", h.AddRecord).Methods("POST")

	r.Use(func(hdl http.Handler) http.Handler {
		return h.PanicRecoverMiddleware(hdl)
	})
	r.Use(func(hdl http.Handler) http.Handler {
		return h.LoggingMiddleware(hdl)
	})

	return r
}
