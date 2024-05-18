package httpHandler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/RusGadzhiev/MEPHI-PARVPO/internal/service"
	"github.com/RusGadzhiev/MEPHI-PARVPO/pkg/logger"
	"github.com/gorilla/mux"
)

type Service interface {
	GetConcerts(ctx context.Context) ([]service.Concert, error)
	AddRecord(ctx context.Context, record service.Record) error
}

type HttpHandler struct {
	service Service
}

func NewHttpHandler(service Service) *HttpHandler {
	return &HttpHandler{
		service: service,
	}
}

func (h *HttpHandler) Router() *mux.Router {
	r := mux.NewRouter()
	r.StrictSlash(true)
	r.HandleFunc("/get-concerts", h.GetConcerts).Methods("GET")
	r.HandleFunc("/add-record", h.AddRecord).Methods("POST")

	r.Use(func(hdl http.Handler) http.Handler {
		return h.PanicRecoverMiddleware(hdl)
	})
	r.Use(func(hdl http.Handler) http.Handler {
		return h.LoggingMiddleware(hdl)
	})
	
	return r
}

func (h *HttpHandler) GetConcerts(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	concerts, err := h.service.GetConcerts(ctx)
	if err != nil {
		logger.Errorf("GetConcerts err: %w", err)
		h.serverError(w)
		return
	}

	w.WriteHeader(http.StatusOK)
	err = renderJSON(w, concerts)
	if err != nil {
		logger.Errorf("RenderJson err: %w", err)
	}
}

func (h *HttpHandler) AddRecord(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	record := &service.Record{
		Concert: r.URL.Query().Get("concert"),
		Username: r.URL.Query().Get("user"),
	}

	if record.Concert == "" || record.Username == "" {
		logger.Info("No request data")
		h.clientError(w)
		return
	}

	err := h.service.AddRecord(ctx, *record)
	if err == service.ErrSoldOut {
		logger.Info(err, record.Concert)
		w.WriteHeader(http.StatusOK)
		err = renderJSON(w, err.Error())
		if err != nil {
			logger.Errorf("RenderJson err: %w", err)
		}
		return
	} else if err == service.ErrNoConcert {
		logger.Infof("No concert: %s", record.Concert)
		h.clientError(w)
		return
	} else if err == service.ErrUserRegistred {
		logger.Infof("User: %s, Concert: %s,  %s", record.Username, record.Concert, err.Error())
		h.clientError(w)
		return
	} else if err != nil {
		logger.Errorf("AddRecord err: %w", err)
		h.serverError(w)
		return
	}

	w.WriteHeader(http.StatusCreated)
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