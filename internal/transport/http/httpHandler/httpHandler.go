package httpHandler

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/RusGadzhiev/MEPHI-PARVPO/internal/broker/kafka"
	"github.com/RusGadzhiev/MEPHI-PARVPO/internal/service"
	"github.com/RusGadzhiev/MEPHI-PARVPO/pkg/logger"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type HttpHandler struct {
	broker kafka.ApiBroker
}

func (h *HttpHandler) AddRecord(w http.ResponseWriter, r *http.Request) {
	var mu sync.Mutex

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

	bytes, err := json.Marshal(message)
	if err != nil {
		logger.Errorf("failed to marshal JSON err: %v", err)
		h.serverError(w)
		return
	}

	msg := &sarama.ProducerMessage{
		Topic: kafka.RequestsTopic,
		Key:   sarama.StringEncoder(requestID),
		Value: sarama.ByteEncoder(bytes),
	}

	_, _, err = h.broker.Producer.SendMessage(msg)
	if err != nil {
		logger.Errorf("Failed to send message to Kafka: %v", err)
		h.serverError(w)
		return
	}

	responseCh := make(chan *sarama.ConsumerMessage)
	mu.Lock()
	h.broker.ResponseChannels[requestID] = responseCh
	mu.Unlock()

	select {
	case responseMsg := <-responseCh:
		var receivedMessage string
		err := json.Unmarshal(responseMsg.Value, &receivedMessage)
		if err != nil {
			logger.Errorf("Error unmarshaling JSON: %v", err)
			h.serverError(w)
			return
		}
		logger.Infof("Response: %v", receivedMessage)
		w.WriteHeader(http.StatusOK)
		err = renderJSON(w, receivedMessage)
		if err != nil {
			logger.Errorf("RenderJson err: %w", err)
		}
	case <-time.After(4 * time.Second):
		logger.Infof("timeout waiting for response")
		h.serverError(w)
	}
	mu.Lock()
	close(h.broker.ResponseChannels[requestID])
	delete(h.broker.ResponseChannels, requestID)
	mu.Unlock()
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

func NewHttpHandler(b kafka.ApiBroker) *HttpHandler {
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
