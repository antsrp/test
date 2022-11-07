package service

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/antsrp/balance_service/internal/postgres"
	"github.com/antsrp/balance_service/internal/reports"
	"go.uber.org/zap"
)

var service *Service

type OperationsParams struct {
	userID    int
	page      int
	sortby    string
	direction string
}

func TestMain(m *testing.M) {

	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal("Can't create zap logger: ", err)
	}

	cfg := ParseDBConfig(logger)

	db, err := postgres.SQLConnect(cfg, logger)
	if err != nil {
		logger.Sugar().Fatal("Can't create db: ", err)
	}

	us, err := postgres.CreateUserStorage(db)
	if err != nil {
		logger.Sugar().Fatal("Can't create a user storage: ", err)
	}
	rs, err := postgres.CreateTransactionStorage(db, cfg.Limitations.PageLimit)
	if err != nil {
		logger.Sugar().Fatal("Can't create a transaction storage: ", err)
	}
	service = CreateNewService(us, rs)

	code := m.Run()

	os.Exit(code)
}

type Operation int

const (
	ADD Operation = iota + 1
	RESERVE
	REVENUE
	CHECK
)

type TestObject struct {
	operation Operation
	data      []byte
	id        string
}

func (o Operation) String() string {
	switch o {
	case ADD:
		return `ADD`
	case RESERVE:
		return `RESERVE`
	case REVENUE:
		return `REVENUE`
	case CHECK:
		return `CHECK`
	default:
		return `EMPTY`
	}
}

func TestService(t *testing.T) {

	input := []TestObject{
		{operation: ADD, data: []byte(`
		{
			"user_id": 1,
			"balance": 500,
			"time": "2022-09-28T01:19:20Z",
			"comment": "for breakfast"
		}`)},
		{operation: ADD, data: []byte(`
		{
			"user_id": 2,
			"balance": 700,
			"time": "2022-10-12T07:49:13Z",
			"comment": "happy birthday!"
		}`)},
		{operation: RESERVE, data: []byte(`
		{
			"user_id": 2,
			"order_id": 1,
			"service_id": 3,
			"cost": 100,
			"comment": "present"
		}`)},
		{operation: REVENUE, data: []byte(`
		{
			"user_id": 2,
			"order_id": 1,
			"service_id": 3,
			"closed_at": "2022-10-24T12:40:32Z",
			"cost": 100
		}`)},
		{operation: CHECK, id: "2"},
		{operation: ADD, data: []byte(`
		{
			"user_id": 3,
			"balance": 400,
			"time": "2022-10-26T20:14:39Z",
			"comment": "..."
		}`)},
		{operation: ADD, data: []byte(`
		{
			"user_id": 3,
			"balance": 1000,
			"time": "2022-10-26T21:14:39Z",
			"comment": "....."
		}`)},
		{operation: CHECK, id: "3"},
		{operation: RESERVE, data: []byte(`
		{
			"user_id": 3,
			"order_id": 2,
			"service_id": 1,
			"cost": 300,
			"comment": "i want this service too"
		}`)},
		{operation: REVENUE, data: []byte(`
		{
			"user_id": 3,
			"order_id": 2,
			"service_id": 1,
			"closed_at": "2022-10-27T03:40:12Z",
			"cost": 300
		}`)},
		{operation: RESERVE, data: []byte(`
		{
			"user_id": 3,
			"order_id": 3,
			"service_id": 3,
			"cost": 100,
			"comment": "i have money for sure"
		}`)},
		{operation: REVENUE, data: []byte(`
		{
			"user_id": 3,
			"order_id": 3,
			"service_id": 3,
			"closed_at": "2022-10-28T22:19:19Z",
			"cost": 100
		}`)},
		{operation: RESERVE, data: []byte(`
		{
			"user_id": 1,
			"order_id": 4,
			"service_id": 1,
			"cost": 400,
			"comment": "i wanna favor 1"
		}`)},
		{operation: CHECK, id: "1"},
		{operation: REVENUE, data: []byte(`
		{
			"user_id": 1,
			"order_id": 5,
			"service_id": 2,
			"closed_at": "2022-11-04T06:15:51Z",
			"cost": 300
		}`)},
		{operation: REVENUE, data: []byte(`
		{
			"user_id": 1,
			"order_id": 4,
			"service_id": 1,
			"closed_at": "2022-11-04T07:09:40Z",
			"cost": 200
		}`)},
		{operation: REVENUE, data: []byte(`
		{
			"user_id": 2,
			"order_id": 4,
			"service_id": 1,
			"closed_at": "2022-11-04T07:09:40Z",
			"cost": 400
		}`)},
		{operation: REVENUE, data: []byte(`
		{
			"user_id": 1,
			"order_id": 4,
			"service_id": 1,
			"closed_at": "2022-11-04T07:12:45Z",
			"cost": 400
		}`)},
		{operation: REVENUE, data: []byte(`
		{
			"user_id": 1,
			"order_id": 4,
			"service_id": 1,
			"closed_at": "2022-11-04T07:13:32Z",
			"cost": 400
		}`)},
		{operation: CHECK, id: "1"},
		{operation: RESERVE, data: []byte(`
		{
			"user_id": 1,
			"order_id": 5,
			"service_id": 2,
			"cost": 300,
			"comment": "now it's time for favor 2"
		}`)},
		{operation: CHECK, id: "1"},
	}

	expection := []Response{
		{Error: nil, Message: OperationSuccessful},
		{Error: nil, Message: OperationSuccessful},
		{Error: nil, Message: OperationSuccessful},
		{Error: nil, Message: OperationSuccessful},
		//{Error: nil, Message: OperationSuccessful, Data: `{"balance": 600}`},
		{Error: nil, Message: OperationSuccessful, Data: Balance{Value: 600}},
		{Error: nil, Message: OperationSuccessful},
		{Error: nil, Message: OperationSuccessful},
		//{Error: nil, Message: OperationSuccessful, Data: `{"balance": 1400}`},
		{Error: nil, Message: OperationSuccessful, Data: Balance{Value: 1400}},
		{Error: nil, Message: OperationSuccessful},
		{Error: nil, Message: OperationSuccessful},
		{Error: nil, Message: OperationSuccessful},
		{Error: nil, Message: OperationSuccessful},
		{Error: nil, Message: OperationSuccessful},
		//{Error: nil, Message: OperationSuccessful, Data: `{"balance": 500}`},
		{Error: nil, Message: OperationSuccessful, Data: Balance{Value: 500}},
		{Error: ErrOrderNotFound, Message: OrderNotFound},
		{Error: ErrDifferentCosts, Message: ErrDifferentCosts.Error()},
		{Error: ErrOrderNotFound, Message: OperationOfDifferentUser},
		{Error: nil, Message: OperationSuccessful},
		{Error: ErrAlreadyClosedTransaction, Message: AlreadyClosedTransaction},
		//{Error: nil, Message: OperationSuccessful, Data: `{"balance": 100}`},
		{Error: nil, Message: OperationSuccessful, Data: Balance{Value: 100}},
		{Error: ErrInsufficientFunds, Message: ErrInsufficientFunds.Error()},
		//{Error: nil, Message: OperationSuccessful, Data: `{"balance": 100}`},
		{Error: nil, Message: OperationSuccessful, Data: Balance{Value: 100}},
	}

	for i, val := range input {
		var result *Response
		switch val.operation {
		case ADD:
			result = service.AddBalanceLogic(val.data)
		case CHECK:
			result = service.GetUserBalanceLogic(val.id)
		case RESERVE:
			result = service.CashReservationLogic(val.data)
		case REVENUE:
			result = service.RevenueLogic(val.data)
		}
		if result.Error != expection[i].Error {
			t.Errorf("Row %v, Operation %v, actual error: %v, expected: %v", i+1, val.operation, result.Error, expection[i].Error)
		}
		if result.Message != expection[i].Message {
			t.Errorf("Row %v, Operation %v, actual message: %v, expected: %v", i+1, val.operation, result.Message, expection[i].Message)
		}
		if expection[i].Data != "" && result.Data != expection[i].Data {
			t.Errorf("Row %v, Operation %v, actual data: %v, expected: %v", i+1, val.operation, result.Data, expection[i].Data)
		}
	}

}

func TestOperationsDefault(t *testing.T) {

	params := &OperationsParams{userID: 3, page: 0, sortby: "", direction: ""}

	times := []string{
		"2022-10-26T20:14:39Z",
		"2022-10-26T21:14:39Z",
		"2022-10-27T03:40:12Z",
		"2022-10-28T22:19:19Z",
	}

	expected := []reports.Operation{
		{Type: "in", Sum: 400, Comment: "..."},
		{Type: "in", Sum: 1000, Comment: "....."},
		{Type: "out", Favor: "Favor 1", Sum: 300, Comment: "i want this service too"},
		{Type: "out", Favor: "Favor 3", Sum: 100, Comment: "i have money for sure"},
	}

	for i := range expected {
		t, _ := time.Parse("2006-01-02T15:04:05Z", times[i])
		expected[i].Time = &t
	}

	data, err := json.MarshalIndent(expected, "", "\t")
	if err != nil {
		log.Fatal(err)
	}

	expection := Response{Error: nil, Message: OperationSuccessful, Data: data}

	result := service.GetOperations(params.userID, params.page, params.sortby, params.direction)
	b, err := json.MarshalIndent(result.Data, "", "\t")
	if err != nil {
		log.Fatal(err)
	}
	a := string(b)
	e := string(data)

	if result.Error != expection.Error {
		t.Errorf("Test operations, actual error: %v, expected: %v", result.Error, expection.Error)
	}
	if result.Message != expection.Message {
		t.Errorf("Test operations, actual message: %v, expected: %v", result.Message, expection.Message)
	}
	if a != e {
		t.Errorf("Test operations, actual data: %v, expected: %v", a, e)
	}
}

func TestOperationsDateAsc(t *testing.T) {

	params := &OperationsParams{userID: 3, page: 1, sortby: "date", direction: "ASC"}

	times := []string{
		"2022-10-26T20:14:39Z",
		"2022-10-26T21:14:39Z",
		"2022-10-27T03:40:12Z",
		"2022-10-28T22:19:19Z",
	}

	expected := []reports.Operation{
		{Type: "in", Sum: 400, Comment: "..."},
		{Type: "in", Sum: 1000, Comment: "....."},
		{Type: "out", Favor: "Favor 1", Sum: 300, Comment: "i want this service too"},
		{Type: "out", Favor: "Favor 3", Sum: 100, Comment: "i have money for sure"},
	}

	for i := range expected {
		t, _ := time.Parse("2006-01-02T15:04:05Z", times[i])
		expected[i].Time = &t
	}

	data, err := json.MarshalIndent(expected, "", "\t")
	if err != nil {
		log.Fatal(err)
	}

	expection := Response{Error: nil, Message: OperationSuccessful, Data: data}

	result := service.GetOperations(params.userID, params.page, params.sortby, params.direction)
	b, err := json.MarshalIndent(result.Data, "", "\t")
	if err != nil {
		log.Fatal(err)
	}
	a := string(b)
	e := string(data)

	if result.Error != expection.Error {
		t.Errorf("Test operations, actual error: %v, expected: %v", result.Error, expection.Error)
	}
	if result.Message != expection.Message {
		t.Errorf("Test operations, actual message: %v, expected: %v", result.Message, expection.Message)
	}
	if a != e {
		t.Errorf("Test operations, actual data: %v, expected: %v", a, e)
	}
}

func TestOperationsDateDesc(t *testing.T) {

	params := &OperationsParams{userID: 3, page: 1, sortby: "date", direction: "DESC"}

	times := []string{
		"2022-10-28T22:19:19Z",
		"2022-10-27T03:40:12Z",
		"2022-10-26T21:14:39Z",
		"2022-10-26T20:14:39Z",
	}

	expected := []reports.Operation{
		{Type: "out", Favor: "Favor 3", Sum: 100, Comment: "i have money for sure"},
		{Type: "out", Favor: "Favor 1", Sum: 300, Comment: "i want this service too"},
		{Type: "in", Sum: 1000, Comment: "....."},
		{Type: "in", Sum: 400, Comment: "..."},
	}

	for i := range expected {
		t, _ := time.Parse("2006-01-02T15:04:05Z", times[i])
		expected[i].Time = &t
	}

	data, err := json.MarshalIndent(expected, "", "\t")
	if err != nil {
		log.Fatal(err)
	}

	expection := Response{Error: nil, Message: OperationSuccessful, Data: data}

	result := service.GetOperations(params.userID, params.page, params.sortby, params.direction)
	b, err := json.MarshalIndent(result.Data, "", "\t")
	if err != nil {
		log.Fatal(err)
	}
	a := string(b)
	e := string(data)

	if result.Error != expection.Error {
		t.Errorf("Test operations, actual error: %v, expected: %v", result.Error, expection.Error)
	}
	if result.Message != expection.Message {
		t.Errorf("Test operations, actual message: %v, expected: %v", result.Message, expection.Message)
	}
	if a != e {
		t.Errorf("Test operations, actual data: %v, expected: %v", a, e)
	}
}

func TestOperationsSumAsc(t *testing.T) {

	params := &OperationsParams{userID: 3, page: 1, sortby: "sum", direction: "ASC"}

	times := []string{
		"2022-10-28T22:19:19Z",
		"2022-10-27T03:40:12Z",
		"2022-10-26T20:14:39Z",
		"2022-10-26T21:14:39Z",
	}

	expected := []reports.Operation{
		{Type: "out", Favor: "Favor 3", Sum: 100, Comment: "i have money for sure"},
		{Type: "out", Favor: "Favor 1", Sum: 300, Comment: "i want this service too"},
		{Type: "in", Sum: 400, Comment: "..."},
		{Type: "in", Sum: 1000, Comment: "....."},
	}

	for i := range expected {
		t, _ := time.Parse("2006-01-02T15:04:05Z", times[i])
		expected[i].Time = &t
	}

	data, err := json.MarshalIndent(expected, "", "\t")
	if err != nil {
		log.Fatal(err)
	}

	expection := Response{Error: nil, Message: OperationSuccessful, Data: data}

	result := service.GetOperations(params.userID, params.page, params.sortby, params.direction)
	b, err := json.MarshalIndent(result.Data, "", "\t")
	if err != nil {
		log.Fatal(err)
	}
	a := string(b)
	e := string(data)

	if result.Error != expection.Error {
		t.Errorf("Test operations, actual error: %v, expected: %v", result.Error, expection.Error)
	}
	if result.Message != expection.Message {
		t.Errorf("Test operations, actual message: %v, expected: %v", result.Message, expection.Message)
	}
	if a != e {
		t.Errorf("Test operations, actual data: %v, expected: %v", a, e)
	}
}

func TestOperationsSumDesc(t *testing.T) {

	params := &OperationsParams{userID: 3, page: 1, sortby: "sum", direction: "DESC"}

	times := []string{
		"2022-10-26T21:14:39Z",
		"2022-10-26T20:14:39Z",
		"2022-10-27T03:40:12Z",
		"2022-10-28T22:19:19Z",
	}

	expected := []reports.Operation{
		{Type: "in", Sum: 1000, Comment: "....."},
		{Type: "in", Sum: 400, Comment: "..."},
		{Type: "out", Favor: "Favor 1", Sum: 300, Comment: "i want this service too"},
		{Type: "out", Favor: "Favor 3", Sum: 100, Comment: "i have money for sure"},
	}

	for i := range expected {
		t, _ := time.Parse("2006-01-02T15:04:05Z", times[i])
		expected[i].Time = &t
	}

	data, err := json.MarshalIndent(expected, "", "\t")
	if err != nil {
		log.Fatal(err)
	}

	expection := Response{Error: nil, Message: OperationSuccessful, Data: data}

	result := service.GetOperations(params.userID, params.page, params.sortby, params.direction)
	b, err := json.MarshalIndent(result.Data, "", "\t")
	if err != nil {
		log.Fatal(err)
	}
	a := string(b)
	e := string(data)

	if result.Error != expection.Error {
		t.Errorf("Test operations, actual error: %v, expected: %v", result.Error, expection.Error)
	}
	if result.Message != expection.Message {
		t.Errorf("Test operations, actual message: %v, expected: %v", result.Message, expection.Message)
	}
	if a != e {
		t.Errorf("Test operations, actual data: %v, expected: %v", a, e)
	}
}

func addRowsForPagination() {
	input := []TestObject{
		{operation: ADD, data: []byte(`
		{
			"user_id": 3,
			"balance": 200,
			"time": "2022-09-29T06:19:20Z",
			"comment": "omg"
		}`)},
		{operation: ADD, data: []byte(`
		{
			"user_id": 3,
			"balance": 1700,
			"time": "2022-10-29T13:55:32Z",
			"comment": "you are funny"
		}`)},
	}

	for _, val := range input {
		service.AddBalanceLogic(val.data)
	}
}

func addOperationsForPagination() {
	input := []TestObject{
		{operation: RESERVE, data: []byte(`
		{
			"user_id": 3,
			"order_id": 6,
			"service_id": 2,
			"cost": 1800,
			"comment": "expensive pleasure."
		}`)},
		{operation: REVENUE, data: []byte(`
		{
			"user_id": 3,
			"order_id": 6,
			"service_id": 2,
			"closed_at": "2022-10-30T16:42:44Z",
			"cost": 1800
		}`)},
	}

	for i, val := range input {
		if i%2 == 0 {
			service.CashReservationLogic(val.data)
		} else {
			service.RevenueLogic(val.data)
		}
	}
}

func TestPagination1(t *testing.T) {

	addRowsForPagination()
	addOperationsForPagination()

	params := &OperationsParams{userID: 3, page: 1, sortby: "sum", direction: "DESC"}

	times := []string{
		"2022-10-30T16:42:44Z",
		"2022-10-29T13:55:32Z",
		"2022-10-26T21:14:39Z",
		"2022-10-26T20:14:39Z",
		"2022-10-27T03:40:12Z",
	}

	expected := []reports.Operation{
		{Type: "out", Favor: "Favor 2", Sum: 1800, Comment: "expensive pleasure."},
		{Type: "in", Sum: 1700, Comment: "you are funny"},
		{Type: "in", Sum: 1000, Comment: "....."},
		{Type: "in", Sum: 400, Comment: "..."},
		{Type: "out", Favor: "Favor 1", Sum: 300, Comment: "i want this service too"},
	}

	for i := range expected {
		t, _ := time.Parse("2006-01-02T15:04:05Z", times[i])
		expected[i].Time = &t
	}

	data, err := json.MarshalIndent(expected, "", "\t")
	if err != nil {
		log.Fatal(err)
	}

	expection := Response{Error: nil, Message: OperationSuccessful, Data: data}

	result := service.GetOperations(params.userID, params.page, params.sortby, params.direction)
	b, err := json.MarshalIndent(result.Data, "", "\t")
	if err != nil {
		log.Fatal(err)
	}
	a := string(b)
	e := string(data)

	if result.Error != expection.Error {
		t.Errorf("Test operations, actual error: %v, expected: %v", result.Error, expection.Error)
	}
	if result.Message != expection.Message {
		t.Errorf("Test operations, actual message: %v, expected: %v", result.Message, expection.Message)
	}
	if a != e {
		t.Errorf("Test operations, actual data: %v, expected: %v", a, e)
	}
}

func TestPagination2(t *testing.T) {
	params := &OperationsParams{userID: 3, page: 2, sortby: "sum", direction: "DESC"}

	times := []string{
		"2022-09-29T06:19:20Z",
		"2022-10-28T22:19:19Z",
	}

	expected := []reports.Operation{
		{Type: "in", Sum: 200, Comment: "omg"},
		{Type: "out", Favor: "Favor 3", Sum: 100, Comment: "i have money for sure"},
	}

	for i := range expected {
		t, _ := time.Parse("2006-01-02T15:04:05Z", times[i])
		expected[i].Time = &t
	}

	data, err := json.MarshalIndent(expected, "", "\t")
	if err != nil {
		log.Fatal(err)
	}

	expection := Response{Error: nil, Message: OperationSuccessful, Data: data}

	result := service.GetOperations(params.userID, params.page, params.sortby, params.direction)
	b, err := json.MarshalIndent(result.Data, "", "\t")
	if err != nil {
		log.Fatal(err)
	}
	a := string(b)
	e := string(data)

	if result.Error != expection.Error {
		t.Errorf("Test operations, actual error: %v, expected: %v", result.Error, expection.Error)
	}
	if result.Message != expection.Message {
		t.Errorf("Test operations, actual message: %v, expected: %v", result.Message, expection.Message)
	}
	if a != e {
		t.Errorf("Test operations, actual data: %v, expected: %v", a, e)
	}
}

func TestSummary(t *testing.T) {

	year, month := 2022, 10

	expection := Response{Error: nil, Message: OperationSuccessful}

	e := `Favor 1;300
Favor 2;1800
Favor 3;200
`

	result := service.GetSummaryLogic(year, month)

	csv := fmt.Sprintf("%s\\%s", getPathToReportsFolder(), result.Data)

	b, err := os.ReadFile(csv)
	if err != nil {
		log.Fatal(err)
	}

	a := string(b)

	if result.Error != expection.Error {
		t.Errorf("Test operations, actual error: %v, expected: %v", result.Error, expection.Error)
	}
	if result.Message != expection.Message {
		t.Errorf("Test operations, actual message: %v, expected: %v", result.Message, expection.Message)
	}
	if a != e {
		t.Errorf("Test operations, actual data: %v, expected: %v", a, e)
	}
}
