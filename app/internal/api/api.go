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
		r.Get("/matches/upcoming", a.getUpcomingMatches)
		r.Get("/matches/completed", a.getCompletedMatches)
		r.Get("/matches/{match_id}", a.getMatchDetails)
		r.Route("/forgotPassword", func(r chi.Router) {
			r.Post("/", a.forgotPassword)
			r.Post("/confirm", a.updatePassword)
		})
		r.Route("/user", func(r chi.Router) {
			r.Use(a.authorize)
			r.Post("/logout", a.logout)
			r.Delete("/", a.deleteAccount)
			r.Get("/changePassword", a.requestNewPassword)
			r.Post("/changePassword/confirm", a.updatePassword)
			r.Post("/rate", a.rateRefereePerformance)
		})
		r.Route("/referee", func(r chi.Router) {
			r.Use(a.authorize)
			r.Use(a.refereeOnly)
			r.Post("/availability", a.addAvailability)
			r.Delete("/availability", a.removeAvailability)
			r.Get("/profile", a.getRefereeProfile)
			r.Post("/license", a.submitLicenseRequest)
			r.Post("/setPhone", a.requestNewPhone)
			r.Post("/setPhone/confirm", a.updatePhone)
			r.Post("/assignment/respond", a.respondToAssignment)
			r.Get("/assignments/pending", a.getPendingAssignments)
			r.Post("/assignment/cancel", a.cancelAssignment)
			r.Get("/schedule", a.getRefereeSchedule)
		})
		r.Route("/admin", func(r chi.Router) {
			r.Use(a.authorize)
			r.Use(a.adminOnly)
			r.Post("/wages", a.updateWages)
			r.Post("/referee", a.setRefereeProfile)
			r.Post("/teams", a.createTeam)
			r.Post("/venues", a.createVenue)
			r.Post("/matches", a.createMatch)
			r.Post("/match/cancel", a.cancelMatch)
			r.Post("/match/reschedule", a.rescheduleMatch)
			r.Post("/match/assign", a.assignReferee)
			r.Post("/match/assignment/revoke", a.revokeAssignment)
			r.Get("/referees", a.getRefereeDirectory)
			r.Get("/referees/available", a.searchAvailableReferees)
		})
		r.Route("/register", func(r chi.Router) {
			r.Post("/", a.register)
			r.Post("/confirm", a.updatePassword)
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
