package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/feedbackpulse/backend/internal/crypto"
	"github.com/feedbackpulse/backend/internal/sheets"
)

type registerRequest struct {
	Name            string `json:"name"`
	SheetID         string `json:"sheet_id"`
	AllowedHost     string `json:"allowed_host"`
	CredentialsJSON string `json:"credentials_json"` // raw Google service account JSON
}

type registerResponse struct {
	SiteID  string `json:"site_id"`
	Snippet string `json:"snippet"`
	Message string `json:"message"`
}

func handleRegister(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req registerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "invalid JSON body", http.StatusBadRequest)
			return
		}

		req.Name = strings.TrimSpace(req.Name)
		req.SheetID = strings.TrimSpace(req.SheetID)
		req.AllowedHost = strings.TrimSpace(req.AllowedHost)
		req.CredentialsJSON = strings.TrimSpace(req.CredentialsJSON)

		// Validate required fields
		if req.Name == "" {
			jsonError(w, "name is required", http.StatusBadRequest)
			return
		}
		if req.SheetID == "" {
			jsonError(w, "sheet_id is required", http.StatusBadRequest)
			return
		}
		if req.CredentialsJSON == "" {
			jsonError(w, "credentials_json is required", http.StatusBadRequest)
			return
		}

		// Validate credentials can actually access the sheet
		if err := sheets.ValidateAccess(req.CredentialsJSON, req.SheetID); err != nil {
			log.Printf("register validation failed (sheet=%s): %v", req.SheetID, err)
			jsonError(w,
				"Could not access your Google Sheet. Make sure you shared it with the service account email in your credentials JSON.",
				http.StatusBadRequest,
			)
			return
		}

		// Encrypt credentials before storing
		encrypted, err := crypto.Encrypt(req.CredentialsJSON, d.EncryptSecret)
		if err != nil {
			log.Printf("encrypt credentials error: %v", err)
			jsonError(w, "internal error", http.StatusInternalServerError)
			return
		}

		// Save tenant
		t, err := d.Tenants.Create(req.Name, req.SheetID, req.AllowedHost, encrypted)
		if err != nil {
			log.Printf("create tenant error: %v", err)
			jsonError(w, "failed to register", http.StatusInternalServerError)
			return
		}

		log.Printf("new tenant registered: %s (site_id=%s sheet=%s)", t.Name, t.ID, t.SheetID)

		snippet := buildSnippet(t.ID)

		w.WriteHeader(http.StatusCreated)
		jsonOK(w, registerResponse{
			SiteID:  t.ID,
			Snippet: snippet,
			Message: "Registration successful! Copy the snippet below and paste it into your website.",
		})
	}
}

func buildSnippet(siteID string) string {
	return fmt.Sprintf(`<script>
  window.FeedbackPulse = {
    siteId:     "%s",
    backendUrl: "https://feedbackpulse-backend-48ty.onrender.com",
    userEmail:  "",        // optional: set to logged-in user's email
    position:   "bottom-right",
  };
</script>
<script src="https://cdn.jsdelivr.net/gh/YOUR_USERNAME/feedbackpulse-widget@latest/feedbackpulse.js"></script>`, siteID)
}
