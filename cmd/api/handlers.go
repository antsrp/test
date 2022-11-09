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

	httpSwagger "github.com/swaggo/http-swagger"
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
		r.Get("/swagger/*", httpSwagger.Handler(
			httpSwagger.URL("/swagger/doc.json"),
		))
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
	case service.OrderNotFound, service.UserNotFound, service.InvalidData, service.InvalidDate, service.OperationOfDifferentUser, service.AlreadyClosedTransaction:
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

// @Summary Get user balance
// @Description Get user balance by id
// @Tags Routes
// @Produce json
// @Param user_id query string true "id of user"
// @Success 200 {object} service.Response
// @Failure 400,500 {object} service.Response
// @Router /get-balance [get]
func (h Handler) getBalance(w http.ResponseWriter, r *http.Request) {
	user_id := r.URL.Query().Get("user_id")

	resp := h.service.GetUserBalanceLogic(user_id)

	h.writeResponse(w, resp, http.StatusOK)
}

// @Summary Add user balance
// @Description Add balance of user by the amount of "balance" parameter
// @Tags Routes
// @Accept json
// @Produce json
// @Param input body models.AddBalanceRequest true "information of operation to add balance"
// @Success 202 {object} service.Response
// @Failure 400,500 {object} service.Response
// @Router /add-balance [post]
func (h Handler) addBalance(w http.ResponseWriter, r *http.Request) {
	body := h.readBody(r)
	defer r.Body.Close()

	resp := h.service.AddBalanceLogic(body)

	h.writeResponse(w, resp, http.StatusAccepted)
}

// @Summary Get revenue of operation
// @Description Get revenue of operation that started before
// @Tags Routes
// @Accept json
// @Produce json
// @Param input body models.RevenueRequest true "information of operation to get revenue of"
// @Success 202 {object} service.Response
// @Failure 400,500 {object} service.Response
// @Router /get-revenue [put]
func (h Handler) getRevenue(w http.ResponseWriter, r *http.Request) {
	body := h.readBody(r)
	defer r.Body.Close()

	resp := h.service.RevenueLogic(body)

	h.writeResponse(w, resp, http.StatusOK)
}

// @Summary Reserve cash for operation
// @Description Reserve cash for the subsequent operation
// @Tags Routes
// @Accept json
// @Produce json
// @Param input body models.ReserveRequest true "information of operation reserve"
// @Success 202 {object} service.Response
// @Failure 400,422,500 {object} service.Response
// @Router /reserve [post]
func (h Handler) reserveCash(w http.ResponseWriter, r *http.Request) {
	body := h.readBody(r)
	defer r.Body.Close()

	resp := h.service.CashReservationLogic(body)

	h.writeResponse(w, resp, http.StatusAccepted)
}

// @Summary Get summary
// @Description Get summary of revenue grouped by services
// @Tags Routes
// @Produce json
// @Param year query int true "year to collect the report"
// @Param month query int true "month to collect the report"
// @Success 200 {object} service.Response
// @Failure 400,500 {object} service.Response
// @Router /summary [get]
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

// @Summary Get operations of user
// @Description Show operations of interest to him
// @Tags Routes
// @Produce json
// @Param user_id query int true "id of user"
// @Param page query int false "page of operation's report; if not specified, operations are returned all together"
// @Param sort query string false "date, sum"
// @Param direction query string false "ASC, DESC"
// @Success 200 {object} service.Response
// @Failure 400,500 {object} service.Response
// @Router /operations [get]
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
