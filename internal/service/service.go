package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"gofermart/internal/config"
	"gofermart/internal/storage"
)

type User struct {
	Login    string `json:"longURL"`
	Password string `json:"shortURL"`
	Sign     []byte `json:"sign"`
}

type Accrual struct {
	Order   string  `json:"order"`
	Status  string  `json:"status"`
	Accrual float64 `json:"accrual,omitempty"`
}

func NewUser() User {
	user := User{}

	user.Login = ""
	user.Password = ""

	return user
}

func SetUserCookie(req *http.Request, data string) *http.Cookie {
	expiration := time.Now().Add(60 * time.Second)

	return &http.Cookie{
		Name:  "user",
		Value: data,
		// Path:    req.URL.Path,
		Expires: expiration,
	}
}

func SetCookieValue(login string, password string) string {
	key := sha256.Sum256([]byte(password)) // ключ шифрования
	aesblock, _ := aes.NewCipher(key[:32])
	aesgcm, _ := cipher.NewGCM(aesblock)
	// создаём вектор инициализации
	nonceSize := aesgcm.NonceSize()
	nonce := key[len(key)-nonceSize:]
	dst := aesgcm.Seal(nil, nonce, []byte(login), nil) // симметрично зашифровываем

	return hex.EncodeToString(dst)
}

func LuhnValid(number int) bool {
	return (number%10+checksum(number/10))%10 == 0
}

func checksum(number int) int {
	var luhn int

	for i := 0; number > 0; i++ {
		cur := number % 10

		if i%2 == 0 { // even
			cur = cur * 2
			if cur > 9 {
				cur = cur%10 + cur/10
			}
		}

		luhn += cur
		number = number / 10
	}
	return luhn % 10
}

func AccrualService(storage *storage.DB, ticker *time.Ticker, tickerChan chan bool) {
	for {
		select {
		case <-tickerChan:
			return
		case <-ticker.C:
			orders := storage.Repo.GetOrdersByStatus()
			for _, order := range orders {
				accrualURL := fmt.Sprintf("%s/api/orders/%d", config.GetConfigAccrualAddress(), order.OrderNumber)
				response, err := http.Get(accrualURL)
				if err != nil {
					log.Printf("Client could not create request: %s", err.Error())
				}
				defer response.Body.Close()
				accrual := Accrual{}
				json.NewDecoder(response.Body).Decode(&accrual)
				if err != nil {
					log.Printf("%s", err.Error())
				}
				luhn, _ := strconv.Atoi(accrual.Order)
				if order := storage.Repo.GetOrder(luhn); order.ID != 0 {
					if order.Status != accrual.Status {
						storage.Repo.SetAccrual(luhn, accrual.Status, accrual.Accrual)
					}

				}
			}

		}
	}
}
