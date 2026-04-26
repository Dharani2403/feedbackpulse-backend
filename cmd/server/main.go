package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/feedbackpulse/backend/internal/config"
	"github.com/feedbackpulse/backend/internal/handler"
	"github.com/feedbackpulse/backend/internal/tenant"
	"github.com/feedbackpulse/backend/internal/whisper"
)

func main() {
	cfg := config.Load()

	tenantStore, err := tenant.NewStore(cfg.DBPath)
	if err != nil {
		log.Fatalf("failed to open tenant store: %v", err)
	}
	defer tenantStore.Close()

	whisperClient := whisper.NewClient(cfg.WhisperURL, cfg.WhisperSecret)

	router := handler.NewRouter(handler.Deps{
		Tenants:       tenantStore,
		Whisper:       whisperClient,
		AdminKey:      cfg.AdminKey,
		EncryptSecret: cfg.EncryptSecret,
	})

	port := cfg.Port
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("FeedbackPulse backend listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-stop
	log.Println("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
	log.Println("Done.")
}
