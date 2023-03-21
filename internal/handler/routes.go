package handler

import (
	"bufio"
	"compress/gzip"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"gofermart/internal/config"
	"gofermart/internal/models"
	"gofermart/internal/service"
	"gofermart/internal/storage"
)

type Handler struct {
	storage storage.DB
}

func NewHandler(storage storage.DB) *Handler {

	return &Handler{
		storage: storage,
	}
}

type LoginForm struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type Order struct {
	Number   string  `json:"number"`
	Status   string  `json:"status"`
	Accrual  float64 `json:"accrual,omitempty"`
	UploadAt string  `json:"uploaded_at"`
}

type Withdraw struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}

type Processed struct {
	Order    string  `json:"order"`
	Sum      float64 `json:"sum"`
	UploadAt string  `json:"processed_at"`
}

type Balance struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

type gzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

type saver struct {
	file   *os.File
	writer *bufio.Writer
}

func NewSaver(filename string) (*saver, error) {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0777)
	if err != nil {

		return nil, err
	}

	return &saver{
		file:   file,
		writer: bufio.NewWriter(file),
	}, nil
}

func (p *saver) WriteShort(text string) error {
	if _, err := p.writer.Write([]byte(text)); err != nil {

		return err
	}
	if err := p.writer.WriteByte('\n'); err != nil {

		return err
	}

	return p.writer.Flush()
}

func (p *saver) Close() error {

	return p.file.Close()
}

var pathStorage = config.GetConfigPath()

func (w gzipWriter) Write(b []byte) (int, error) {

	return w.Writer.Write(b)
}

func ConnectionDBCheck() (int, string) {
	db, err := sql.Open("pgx", config.GetConfigDBAddress())
	if err != nil {

		return 500, err.Error()
	}
	defer db.Close()
	err = db.Ping()
	if err != nil {

		return 500, err.Error()
	}

	return 200, ""
}

func CodingMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {

			if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
				gzw, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)

					return
				}
				w.Header().Set("Content-Encoding", "gzip")
				w = gzipWriter{
					ResponseWriter: w,
					Writer:         gzw,
				}
				defer gzw.Close()
			}

			if r.Header.Get("Content-Encoding") == "gzip" {
				gzr, err := gzip.NewReader(r.Body)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)

					return
				}
				r.Body = gzr
				defer gzr.Close()
			}

			h.ServeHTTP(w, r)
		},
	)
}

func AuthMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost && r.ContentLength < 1 {
				http.Error(w, "Empty body!", http.StatusBadRequest) // 400 response

				// logger will be here

				return
			}
			if r.Method == http.MethodGet {
				cookie, _ := r.Cookie("user")
				if cookie == nil {
					http.Error(w, "Unauthorized!", http.StatusUnauthorized) // 401 response

					// logger will be here

					return
				}

			}
			h.ServeHTTP(w, r)
		},
	)
}

func (h *Handler) RegisterAction(res http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()

	saver, _ := NewSaver(pathStorage)
	defer saver.Close()

	b, err := io.ReadAll(req.Body)
	if err != nil {

		_ = saver.WriteShort(fmt.Sprintf("%s - ReadAll body failed: %s", time.Now().String(), err.Error()))

		http.Error(res, err.Error(), http.StatusInternalServerError) // 500 response

		return
	}

	form := new(LoginForm)
	if err := json.Unmarshal(b, &form); err != nil {
		_ = saver.WriteShort(fmt.Sprintf("%s - Unmarshal failed: %s", time.Now().String(), err.Error()))

		http.Error(res, err.Error(), http.StatusBadRequest) // 400 response

		return
	}

	if model := h.storage.Repo.UserRegistered(form.Login); model.ID != 0 {

		_ = saver.WriteShort(fmt.Sprintf("%s - Login %s already exist!", time.Now().String(), form.Login))

		http.Error(res, "Login already exist!", http.StatusConflict) // 409 response

		return
	}

	cookieValue := service.SetCookieValue(form.Login, form.Password)
	user := models.User{}
	user.Login = form.Login
	user.Password = cookieValue
	user.CreatedAt = time.Now()
	if err := h.storage.Repo.RegisterUser(&user); err != nil {
		errMessage := fmt.Sprintf("Model saving repository failed %s", err.Error())
		http.Error(res, errMessage, http.StatusConflict) // 409 response

		return
	}

	http.SetCookie(res, service.SetUserCookie(req, cookieValue))
	res.Header().Set("Content-Type", "application/json; charset=utf-8")
	res.WriteHeader(http.StatusOK)
}

func (h *Handler) LoginAction(res http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()

	saver, _ := NewSaver(pathStorage)
	defer saver.Close()

	b, err := io.ReadAll(req.Body)
	if err != nil {

		_ = saver.WriteShort(fmt.Sprintf("%s - ReadAll body failed: %s", time.Now().String(), err.Error()))

		http.Error(res, err.Error(), http.StatusInternalServerError) // 500 response

		return
	}

	form := new(LoginForm)
	if err := json.Unmarshal(b, &form); err != nil {

		_ = saver.WriteShort(fmt.Sprintf("%s - Unmarshal failed: %s", time.Now().String(), err.Error()))

		http.Error(res, err.Error(), http.StatusBadRequest) // 400 response

		return
	}

	passValue := ""
	cookie, _ := req.Cookie("user")
	if cookie == nil {
		passValue = service.SetCookieValue(form.Login, form.Password)
	} else {
		passValue = cookie.Value
	}

	if model := h.storage.Repo.LoginUser(form.Login, passValue); model.ID != 0 {
		http.SetCookie(res, service.SetUserCookie(req, passValue))
		res.Header().Set("Content-Type", "application/json; charset=utf-8")
		res.WriteHeader(http.StatusOK)
	} else {

		_ = saver.WriteShort(fmt.Sprintf("%s - Wrong login/password!", time.Now().String()))

		http.Error(res, "Wrong login/password!", http.StatusUnauthorized) // 401 response

		return
	}
}

func (h *Handler) PostOrdresAction(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(res, "Only Post requests are allowed for this route!", http.StatusBadRequest)

		return
	}
	defer req.Body.Close()

	saver, _ := NewSaver(pathStorage)
	defer saver.Close()

	b, err := io.ReadAll(req.Body)
	if err != nil {

		_ = saver.WriteShort(fmt.Sprintf("%s - ReadAll body failed: %s", time.Now().String(), err.Error()))

		http.Error(res, err.Error(), http.StatusInternalServerError) // 500 response

		return
	}
	cookie, _ := req.Cookie("user")
	if cookie == nil {
		http.Error(res, "Unauthorized!", http.StatusUnauthorized) // 401 response

		// logger will be here

		return
	}
	luhn, _ := strconv.Atoi(string(b))
	if !service.LuhnValid(luhn) {

		_ = saver.WriteShort(fmt.Sprintf("%s - Wrong order number!", time.Now().String()))

		http.Error(res, "Wrong order number!", http.StatusUnprocessableEntity) // 422 response

		return
	}
	user := h.storage.Repo.GetUser(cookie.Value)
	if order := h.storage.Repo.GetOrder(luhn); order.ID != 0 {
		if order.UserID == user.ID {

			_ = saver.WriteShort(fmt.Sprintf("%s - Order already uploaded!", time.Now().String()))

			http.Error(res, "Order already uploaded!", http.StatusOK) // 200 response

			return
		} else {

			_ = saver.WriteShort(fmt.Sprintf("%s - Order already uploaded by another user!", time.Now().String()))

			http.Error(res, "Order already uploaded by another user!", http.StatusConflict) // 409 response

			return
		}
	} else {
		order := models.Order{}
		order.UserID = user.ID
		order.OrderNumber = luhn
		order.Status = "NEW"
		order.Accrual = 0
		order.CreatedAt = time.Now()
		order.UpdatedAt = time.Now()

		h.storage.Repo.SetOrder(&order)
		res.WriteHeader(http.StatusAccepted) // 202 response
	}
}

func (h *Handler) GetOrdresAction(res http.ResponseWriter, req *http.Request) {
	cookie, _ := req.Cookie("user")
	user := h.storage.Repo.GetUser(cookie.Value)
	if user == nil {
		http.Error(res, "User not founded!", http.StatusInternalServerError) // 500 response

		return
	}

	orders := []Order{}
	list := h.storage.Repo.GetOrders(user.ID)
	for _, obj := range list {
		order := new(Order)
		order.Number = strconv.Itoa(obj.OrderNumber)
		order.Status = obj.Status
		order.Accrual = obj.Accrual

		order.UploadAt = obj.CreatedAt.Format(time.RFC3339)
		orders = append(orders, *order)
	}
	if len(orders) == 0 {
		http.Error(res, "No data!", http.StatusNoContent) // 204 response

		return
	}
	p, _ := json.Marshal(orders)
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK) // 200 response
	res.Write([]byte(p))
}

func (h *Handler) BalanceAction(res http.ResponseWriter, req *http.Request) {
	cookie, _ := req.Cookie("user")
	user := h.storage.Repo.GetUser(cookie.Value)
	if user == nil {
		http.Error(res, "User not founded!", http.StatusInternalServerError) // 500 response

		return
	}

	accrouls := 0.0
	withdrawn := 0.0

	orders := h.storage.Repo.GetOrders(user.ID)
	for _, obj := range orders {
		accrouls = accrouls + obj.Accrual
	}

	withdraws := h.storage.Repo.GetWithdraws(user.ID)
	for _, obj := range withdraws {
		withdrawn = withdrawn + obj.Withdraw
	}

	balance := new(Balance)
	balance.Current = accrouls - withdrawn
	balance.Withdrawn = withdrawn

	p, _ := json.Marshal(balance)
	res.Header().Set("Content-Type", "application/json; charset=utf-8")
	res.WriteHeader(http.StatusOK) // 200 response
	res.Write([]byte(p))
}

func (h *Handler) WithdrawAction(res http.ResponseWriter, req *http.Request) {
	cookie, _ := req.Cookie("user")

	defer req.Body.Close()
	b, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)

		return
	}

	withdraw := Withdraw{}
	if err := json.Unmarshal(b, &withdraw); err != nil {
		http.Error(res, err.Error(), http.StatusNotImplemented)

		return
	}
	luhn, _ := strconv.Atoi(withdraw.Order)
	if !service.LuhnValid(luhn) {
		http.Error(res, "Wrong order number!", http.StatusUnprocessableEntity) // 422 response

		return
	}

	if order := h.storage.Repo.GetOrder(luhn); order.ID != 0 {
		if order.Accrual < withdraw.Sum {
			http.Error(res, "Not enouth balance!", http.StatusPaymentRequired) // 402 response

			return
		}
	}

	user := h.storage.Repo.GetUser(cookie.Value)
	if user == nil {
		http.Error(res, "User not founded!", http.StatusInternalServerError) // 500 response

		return
	}
	balance := models.Balance{}
	balance.UserID = user.ID
	balance.OrderID = luhn
	balance.Withdraw = withdraw.Sum
	balance.CreatedAt = time.Now()
	balance.UpdatedAt = time.Now()
	if err := h.storage.Repo.SetWithdraw(&balance); err != nil {
		http.Error(res, "User not founded!", http.StatusInternalServerError) // 500 response

		return
	}
	res.Header().Set("Content-Type", "application/json; charset=utf-8")
	res.WriteHeader(http.StatusOK) // 200 response
}

func (h *Handler) WithdrawalsAction(res http.ResponseWriter, req *http.Request) {
	cookie, _ := req.Cookie("user")

	user := h.storage.Repo.GetUser(cookie.Value)
	if user == nil {
		http.Error(res, "User not founded!", http.StatusInternalServerError) // 500 response

		return
	}
	processes := []Processed{}
	list := h.storage.Repo.GetWithdraws(user.ID)
	for _, obj := range list {
		processed := new(Processed)
		processed.Order = strconv.Itoa(obj.OrderID)
		processed.Sum = obj.Withdraw
		processed.UploadAt = obj.UpdatedAt.Format(time.RFC3339)
		processes = append(processes, *processed)
	}
	if len(processes) == 0 {
		http.Error(res, "No data!", http.StatusNoContent) // 204 response

		return
	}
	p, _ := json.Marshal(processes)
	res.Header().Set("Content-Type", "application/json; charset=utf-8")
	res.WriteHeader(http.StatusOK) // 200 response
	res.Write([]byte(p))
}
