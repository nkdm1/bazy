package api

import (
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/nkdm1/bazy/internal/database"
)

func (a *Api) Mount() http.Handler {
	r := chi.NewRouter()
	r.Use(
		middleware.RequestID,
		middleware.RealIP,
		middleware.Logger,
		middleware.Recoverer,
		middleware.Timeout(10*time.Second),
		a.limitBodySize(1024*1024),
	)

	r.Route("/", func(r chi.Router) {
		r.Post("/login", a.login)
		r.Get("/status", a.status)
		r.Route("/register", func(r chi.Router) {
			r.Post("/", a.register)
			r.Post("/confirm", a.registerConfirm)
		})
	})
	return r
}

func (a *Api) Run(h http.Handler) {
	log.Println("server up on :8080")
	if err := http.ListenAndServe(":8080", h); err != nil {
		panic(err)
	}
}

func Init() *Api {
	return &Api{
		Database: database.Init(),
	}
}

type Api struct {
	Database *database.Database
}
