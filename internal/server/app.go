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
	channel    *service.Channel
}

func NewApp() *App {
	config.SetConfig()
	InputCh := make(chan int)
	listener := service.NewListener(InputCh)

	if status, _ := handler.ConnectionDBCheck(); status != http.StatusOK {

		return nil
	}

	return &App{
		storage: storage.NewDB(),
		channel: listener,
	}
}

func registerHTTPEndpoints(router *chi.Mux, storage storage.DB, service service.Channel) {
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

		r.Route("/orders", func(r chi.Router) {
			r.Route("/{number}", func(r chi.Router) {
				r.Get("/", h.AccrualAction)
			})
		})
	})
}

func (a *App) Run(ctx context.Context) error {
	route := chi.NewRouter()
	address := config.GetConfigServerAddress()
	registerHTTPEndpoints(route, *a.storage, *a.channel)

	a.httpServer = &http.Server{
		Addr:    address,
		Handler: route,
	}

	// go service.AccrualService(a.channel.InputChannel)

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
		time.Sleep(3 * time.Second)
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
