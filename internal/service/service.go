package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/hex"

	// "encoding/json"
	// "fmt"
	// "gofermart/internal/config"
	// "io"
	// "log"
	// "strconv"
	"net/http"
	"sync"
	"time"
)

type User struct {
	Login    string `json:"longURL"`
	Password string `json:"shortURL"`
	Sign     []byte `json:"sign"`
}

type Channel struct {
	InputChannel chan int
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

func NewListener(inputCh chan int) *Channel {

	return &Channel{
		InputChannel: inputCh,
	}
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

func FanOut(inputCh chan int, n int) []chan int {
	chs := make([]chan int, 0, n)
	for i := 0; i < n; i++ {
		ch := make(chan int)
		chs = append(chs, ch)
	}

	go func() {
		defer func(chs []chan int) {
			for _, ch := range chs {
				close(ch)
			}
		}(chs)

		for i := 0; ; i++ {
			if i == len(chs) {
				i = 0
			}

			list, ok := <-inputCh
			if !ok {
				return
			}
			ch := chs[i]
			ch <- list
		}
	}()

	return chs
}

func FanIn(inputChs ...chan int) chan int {
	outCh := make(chan int)
	go func() {
		wg := &sync.WaitGroup{}
		for _, inputCh := range inputChs {
			wg.Add(1)
			go func(inputCh chan int) {
				defer wg.Done()
				for order := range inputCh {
					outCh <- order
				}
			}(inputCh)
		}
		wg.Wait()
		close(outCh)
	}()

	return outCh
}

func NewWorker(input, out chan int) {
	go func() {
		for shorter := range input {

			// some checks maybe?

			out <- shorter
		}
		close(out)
	}()
}

func AccrualService(inputCh chan int) {
	// workersCount := 2
	// workerChs := make([]chan int, 0, workersCount)
	// fanOutChs := FanOut(inputCh, workersCount)
	// for _, fanOutCh := range fanOutChs {
	// 	workerCh := make(chan int)
	// 	NewWorker(fanOutCh, workerCh)
	// 	workerChs = append(workerChs, workerCh)
	// }
	// for id := range FanIn(workerChs...) {
	// 	accrualURL := fmt.Sprintf("http://%s/api/orders/%d", config.GetConfigServerAddress(), id)
	// 	response, err := http.Get(accrualURL)
	// 	if err != nil {
	// 		log.Println("Client could not create request: ", err)

	// 		// logger will be here

	// 	}
	// 	defer response.Body.Close()
	// 	b, err := io.ReadAll(response.Body)
	// 	if err != nil {

	// 		// logger will be here

	// 		return
	// 	}

	// 	accrual := Accrual{}
	// 	if err := json.Unmarshal(b, &accrual); err != nil {
	// 		// http.Error(res, err.Error(), http.StatusInternalServerError)

	// 		// logger will be here

	// 		return

	// 	}

	// 	luhn, _ := strconv.Atoi(accrual.Order)
	// 	if order := h.storage.Repo.GetOrder(luhn); order.ID != 0 {
	// 		if order.Status != accrual.Status {
	// 			h.storage.Repo.SetAccrual(luhn, accrual.Status, accrual.Accrual)
	// 		}

	// 	} else {

	// 		// logger will be here

	// 		return
	// 	}

	// 	log.Println(response)
	// }

}
