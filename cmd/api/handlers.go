package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/antsrp/balance_service/internal/service"
	"github.com/go-chi/chi"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type URLPath struct {
	URL string `json:"path"`
}

type Handler struct {
	logger  *zap.SugaredLogger
	service *service.Service
}

func createNewHandler(logger *zap.Logger, s *service.Service) (*Handler, error) {

	return &Handler{
		logger:  logger.Sugar(),
		service: s,
	}, nil
}

func (h Handler) Routes() chi.Router {

	fileServer := http.FileServer(http.Dir(service.REPORTS_RELATIVE_PATH))

	r := chi.NewRouter()

	r.Route("/", func(r chi.Router) {
		r.Get("/api/v1/get-balance", h.getBalance)
		r.Post("/api/v1/add-balance", h.addBalance)
		r.Post("/api/v1/reserve", h.reserveCash)
		r.Put("/api/v1/get-revenue", h.getRevenue)
		r.Get("/api/v1/operations", h.getOperations)
		r.Get("/api/v1/summary", h.getSummary)
		r.Handle("/reports/*", http.StripPrefix("/reports/", fileServer))
	})

	return r
}

func (h Handler) readBody(r *http.Request) []byte {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		h.logger.Info("cannot read body!")
	}

	return body
}

func (h Handler) httpCodeByMessage(msg string, defaultCode int) int {
	var code int
	switch msg {
	case service.OperationUnsuccessfulInternalError:
		code = http.StatusInternalServerError
	case service.DifferentCosts, service.InsufficientFunds:
		code = http.StatusUnprocessableEntity
	case service.OrderNotFound, service.UserNotFound, service.InvalidData, service.InvalidDate, service.OperationOfDifferentUser:
		code = http.StatusBadRequest
	default:
		code = defaultCode
	}
	return code
}

func (h Handler) marshalResponse(resp *service.Response) []byte {
	if data, err := json.MarshalIndent(resp, "", "\t"); err != nil {
		h.logger.Warnf("can't marshal response: %s", err.Error())
		return nil
	} else {
		return data
	}
}

func (h Handler) writeResponse(w http.ResponseWriter, resp *service.Response, defaultStatusCode int) {
	w.Header().Add("Content-type", "application/json")
	w.WriteHeader(h.httpCodeByMessage(resp.Message, defaultStatusCode))
	if resp.Error != nil {
		h.logger.Error(resp.Error)
	}

	if data := h.marshalResponse(resp); data != nil {
		w.Write(data)
	}
}

func (h Handler) getBalance(w http.ResponseWriter, r *http.Request) {
	user_id := r.URL.Query().Get("user_id")

	resp := h.service.GetUserBalanceLogic(user_id)

	h.writeResponse(w, resp, http.StatusOK)
}

func (h Handler) addBalance(w http.ResponseWriter, r *http.Request) {
	body := h.readBody(r)
	defer r.Body.Close()

	resp := h.service.AddBalanceLogic(body)

	h.writeResponse(w, resp, http.StatusAccepted)
}

func (h Handler) getRevenue(w http.ResponseWriter, r *http.Request) {
	body := h.readBody(r)
	defer r.Body.Close()

	resp := h.service.RevenueLogic(body)

	h.writeResponse(w, resp, http.StatusOK)
}

func (h Handler) reserveCash(w http.ResponseWriter, r *http.Request) {
	body := h.readBody(r)
	defer r.Body.Close()

	resp := h.service.CashReservationLogic(body)

	h.writeResponse(w, resp, http.StatusAccepted)
}

func (h Handler) getSummary(w http.ResponseWriter, r *http.Request) {

	month, err := strconv.Atoi(r.URL.Query().Get("month"))
	if err != nil {
		h.writeResponse(w, &service.Response{Error: err, Message: service.InvalidData}, http.StatusBadRequest)
		return
	}
	year, err := strconv.Atoi(r.URL.Query().Get("year"))
	if err != nil {
		h.writeResponse(w, &service.Response{Error: err, Message: service.InvalidData}, http.StatusBadRequest)
		return
	}

	resp := h.service.GetSummaryLogic(year, month)
	if resp.Error == nil {
		url := URLPath{}
		url.URL = fmt.Sprintf(`%s/reports/%s`, r.Host, resp.Data)
		resp.Data = url
	} else {
		resp.Data = nil
	}

	h.writeResponse(w, resp, http.StatusOK)
}

func (h Handler) getOperations(w http.ResponseWriter, r *http.Request) {
	var id, page int
	id, err := strconv.Atoi(r.URL.Query().Get("user_id"))
	if err != nil {
		h.writeResponse(w, &service.Response{Error: errors.Wrap(err, "can't parse user id"), Message: service.InvalidData}, http.StatusBadRequest)
		return
	}
	page_param := r.URL.Query().Get("page")
	if page_param != "" {
		page, err = strconv.Atoi(page_param)
		if err != nil {
			h.writeResponse(w, &service.Response{Error: errors.Wrap(err, "can't parse page"), Message: service.InvalidData}, http.StatusBadRequest)
			return
		}
	}
	sort, direction := strings.Trim(r.URL.Query().Get("sort"), `\"`), strings.Trim(r.URL.Query().Get("direction"), `\"`)

	resp := h.service.GetOperations(id, page, sort, direction)
	h.writeResponse(w, resp, http.StatusOK)
}
