package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi"

	"gofermart/internal/config"
	"gofermart/internal/handler"

	"gofermart/internal/service"
	"gofermart/internal/storage"
)

type App struct {
	httpServer *http.Server
	storage    *storage.DB
}

func NewApp() *App {
	config.SetConfig()

	if status, _ := handler.ConnectionDBCheck(); status != http.StatusOK {

		return nil
	}

	return &App{
		storage: storage.NewDB(),
	}
}

func registerHTTPEndpoints(router *chi.Mux, storage storage.DB) {
	h := handler.NewHandler(storage)

	router.Route("/api", func(r chi.Router) {
		r.Use(handler.CodingMiddleware)

		r.Route("/user", func(r chi.Router) {
			r.Use(handler.AuthMiddleware)
			r.Post("/register", h.RegisterAction)
			r.Post("/login", h.LoginAction)
			r.Post("/orders", h.PostOrdresAction)
			r.Get("/orders", h.GetOrdresAction)
			r.Route("/balance", func(r chi.Router) {
				r.Get("/", h.BalanceAction)
				r.Post("/withdraw", h.WithdrawAction)
			})
			r.Get("/withdrawals", h.WithdrawalsAction)
		})
	})
}

func (a *App) Run(ctx context.Context) error {
	route := chi.NewRouter()
	address := config.GetConfigServerAddress()
	registerHTTPEndpoints(route, *a.storage)

	a.httpServer = &http.Server{
		Addr:    address,
		Handler: route,
	}

	ticker := time.NewTicker(1 * time.Second)
	tickerChan := make(chan bool)

	go service.AccrualService(a.storage, ticker, tickerChan)

	go func() {
		if err := a.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to listen and serve: %+v", err)
		}

	}()

	<-ctx.Done()

	ctx, shutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdown()

	quit := make(chan struct{}, 1)
	go func() {
		// time.Sleep(3 * time.Second)
		ticker.Stop()
		tickerChan <- true
		quit <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("server shutdown: %w", ctx.Err())
	case <-quit:
		log.Println("finished")
	}

	return nil
}
