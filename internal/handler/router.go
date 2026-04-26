package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/feedbackpulse/backend/internal/tenant"
	"github.com/feedbackpulse/backend/internal/whisper"
	"github.com/feedbackpulse/backend/pkg/ratelimit"
)

type Deps struct {
	Tenants       *tenant.Store
	Whisper       *whisper.Client
	AdminKey      string
	EncryptSecret string
}

func NewRouter(d Deps) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "X-Site-ID"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	limiter := ratelimit.New(5.0/60.0, 10)

	r.Get("/health", handleHealth)
	r.With(limiter.Middleware).Post("/register", handleRegister(d))
	r.With(limiter.Middleware).Post("/feedback", handleFeedback(d))

	r.Route("/admin", func(r chi.Router) {
		r.Use(adminAuth(d.AdminKey))
		r.Get("/tenants", handleListTenants(d))
	})

	return r
}

func adminAuth(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-Admin-Key") != key {
				jsonError(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
