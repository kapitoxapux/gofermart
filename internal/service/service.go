package service

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
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

type saver struct {
	file   *os.File
	writer *bufio.Writer
}

func (p *saver) Close() error {

	return p.file.Close()
}

func NewSaver(filename string) (*saver, error) {
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_APPEND, 0777)
	if err != nil {

		return nil, err
	}

	return &saver{
		file:   file,
		writer: bufio.NewWriter(file),
	}, nil
}

func (p *saver) WriteShort(text string) error {
	json, err := json.Marshal(text)
	if err != nil {

		return err
	}
	if _, err := p.writer.Write(json); err != nil {

		return err
	}
	if err := p.writer.WriteByte('\n'); err != nil {

		return err
	}

	return p.writer.Flush()
}

var pathStorage = config.GetConfigPath()

func Logger(text string) {
	saver, _ := NewSaver(pathStorage)
	defer saver.Close()

	_ = saver.WriteShort(text)
}

func AccrualService(storage *storage.DB, ticker *time.Ticker, tickerChan chan bool) {
	// saver, _ := NewSaver(pathStorage)
	// defer saver.Close()

	// _ = saver.WriteShort("in")

	for {
		select {
		case <-tickerChan:

			return
		case <-ticker.C:
			// Logger("in")

			orders := storage.Repo.GetOrdersByStatus()

			for _, order := range orders {
				accrualURL := fmt.Sprintf("http://%s/api/orders/%d", config.GetConfigAccrualAddress(), order.OrderNumber)
				response, err := http.Get(accrualURL)
				if err != nil {
					// Logger(fmt.Sprintf("Client could not create request: %s", err.Error()))
					log.Printf("Client could not create request: %s", err.Error())
				}

				defer response.Body.Close()

				b, err := io.ReadAll(response.Body)
				if err != nil {
					// Logger(err.Error())
					log.Printf("%s", err.Error())
				}
				accrual := Accrual{}
				if err := json.Unmarshal(b, &accrual); err != nil {
					// Logger(err.Error())
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
