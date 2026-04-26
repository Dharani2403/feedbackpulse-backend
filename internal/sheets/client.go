package sheets

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	googlesheets "google.golang.org/api/sheets/v4"
)

// FeedbackRow is one row written to the Google Sheet.
type FeedbackRow struct {
	Timestamp  time.Time
	SiteID     string
	Email      string
	Emoji      string
	Transcript string
}

// NewServiceForCredentials creates a Sheets service from a raw credentials JSON string.
// Called per-request using the tenant's own credentials.
func NewServiceForCredentials(credentialsJSON string) (*googlesheets.Service, error) {
	ctx := context.Background()
	creds, err := google.CredentialsFromJSON(
		ctx,
		[]byte(credentialsJSON),
		googlesheets.SpreadsheetsScope,
	)
	if err != nil {
		return nil, fmt.Errorf("parse google credentials: %w", err)
	}
	svc, err := googlesheets.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("create sheets service: %w", err)
	}
	return svc, nil
}

// ValidateAccess checks if the credentials can read the given sheet.
// Used during registration to verify before saving.
func ValidateAccess(credentialsJSON, sheetID string) error {
	svc, err := NewServiceForCredentials(credentialsJSON)
	if err != nil {
		return err
	}
	_, err = svc.Spreadsheets.Values.Get(sheetID, "A1").Do()
	if err != nil {
		return fmt.Errorf("cannot access sheet: %w", err)
	}
	return nil
}

// EnsureHeader writes the header row if the sheet is empty.
func EnsureHeader(svc *googlesheets.Service, sheetID string) error {
	resp, err := svc.Spreadsheets.Values.Get(sheetID, "A1:E1").Do()
	if err != nil {
		return fmt.Errorf("get header row: %w", err)
	}
	if len(resp.Values) > 0 && len(resp.Values[0]) > 0 {
		return nil
	}
	header := []interface{}{"Timestamp", "Site ID", "Email", "Emoji", "Voice Transcript"}
	_, err = svc.Spreadsheets.Values.Append(sheetID, "A1",
		&googlesheets.ValueRange{Values: [][]interface{}{header}},
	).ValueInputOption("RAW").Do()
	return err
}

// Append writes a feedback row to the sheet.
func Append(svc *googlesheets.Service, sheetID string, row FeedbackRow) error {
	values := []interface{}{
		row.Timestamp.UTC().Format(time.RFC3339),
		row.SiteID,
		row.Email,
		row.Emoji,
		row.Transcript,
	}
	_, err := svc.Spreadsheets.Values.Append(sheetID, "A1",
		&googlesheets.ValueRange{Values: [][]interface{}{values}},
	).ValueInputOption("RAW").InsertDataOption("INSERT_ROWS").Do()
	if err != nil {
		return fmt.Errorf("append row to sheet %s: %w", sheetID, err)
	}
	return nil
}
