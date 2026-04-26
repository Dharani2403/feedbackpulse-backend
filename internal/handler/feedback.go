package handler

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/feedbackpulse/backend/internal/crypto"
	"github.com/feedbackpulse/backend/internal/sheets"
)

var validEmojis = map[string]bool{
	"happy":   true,
	"neutral": true,
	"sad":     true,
}

const maxAudioSize = 5 * 1024 * 1024 // 5MB

func handleFeedback(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(6 << 20); err != nil {
			jsonError(w, "invalid form data", http.StatusBadRequest)
			return
		}

		// --- Required: site_id ---
		siteID := strings.TrimSpace(r.FormValue("site_id"))
		if siteID == "" {
			jsonError(w, "site_id is required", http.StatusBadRequest)
			return
		}

		// Validate tenant exists
		t, err := d.Tenants.GetByID(siteID)
		if err != nil {
			log.Printf("tenant lookup error: %v", err)
			jsonError(w, "internal error", http.StatusInternalServerError)
			return
		}
		if t == nil {
			jsonError(w, "unknown site_id", http.StatusUnauthorized)
			return
		}

		// Optional origin check
		if t.AllowedHost != "" {
			origin := r.Header.Get("Origin")
			if origin == "" {
				origin = r.Header.Get("Referer")
			}
			if !strings.Contains(origin, t.AllowedHost) {
				jsonError(w, "origin not allowed", http.StatusForbidden)
				return
			}
		}

		// --- Required: emoji ---
		emoji := strings.TrimSpace(r.FormValue("emoji"))
		if !validEmojis[emoji] {
			jsonError(w, fmt.Sprintf("emoji must be one of: happy, neutral, sad"), http.StatusBadRequest)
			return
		}

		// --- Optional: email ---
		email := strings.TrimSpace(r.FormValue("email"))

		// --- Optional: audio ---
		var transcript string
		audioFile, audioHeader, audioErr := r.FormFile("audio")
		if audioErr == nil {
			defer audioFile.Close()
			if audioHeader.Size > maxAudioSize {
				jsonError(w, "audio file too large (max 5MB)", http.StatusRequestEntityTooLarge)
				return
			}
			audioBytes, err := io.ReadAll(audioFile)
			if err != nil {
				jsonError(w, "failed to read audio", http.StatusInternalServerError)
				return
			}
			transcript, err = d.Whisper.Transcribe(audioBytes, audioHeader.Filename)
			if err != nil {
				log.Printf("whisper transcription failed (site=%s): %v", siteID, err)
				transcript = ""
			}
		}

		// --- Decrypt tenant credentials ---
		credsJSON, err := crypto.Decrypt(t.EncryptedCredsJSON, d.EncryptSecret)
		if err != nil {
			log.Printf("decrypt creds error (site=%s): %v", siteID, err)
			jsonError(w, "internal error", http.StatusInternalServerError)
			return
		}

		// --- Build per-tenant Sheets service ---
		svc, err := sheets.NewServiceForCredentials(credsJSON)
		if err != nil {
			log.Printf("sheets service error (site=%s): %v", siteID, err)
			jsonError(w, "internal error", http.StatusInternalServerError)
			return
		}

		// --- Write to Google Sheets ---
		row := sheets.FeedbackRow{
			Timestamp:  time.Now(),
			SiteID:     siteID,
			Email:      email,
			Emoji:      emoji,
			Transcript: transcript,
		}

		if err := sheets.EnsureHeader(svc, t.SheetID); err != nil {
			log.Printf("ensure header error (sheet=%s): %v", t.SheetID, err)
		}

		if err := sheets.Append(svc, t.SheetID, row); err != nil {
			log.Printf("sheets append error (sheet=%s): %v", t.SheetID, err)
			jsonError(w, "failed to store feedback", http.StatusInternalServerError)
			return
		}

		log.Printf("feedback stored | site=%s emoji=%s email=%s transcript_len=%d",
			siteID, emoji, email, len(transcript))

		jsonOK(w, map[string]string{
			"status":     "ok",
			"transcript": transcript,
		})
	}
}
