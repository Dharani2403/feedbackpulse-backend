package tenant

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

// Tenant represents a business using FeedbackPulse.
type Tenant struct {
	ID                 string    `json:"id"`           // site_id passed in widget script tag
	Name               string    `json:"name"`         // human label e.g. "Acme Bakery"
	SheetID            string    `json:"sheet_id"`     // Google Sheet ID
	AllowedHost        string    `json:"allowed_host"` // e.g. "acmebakery.com" (origin check)
	EncryptedCredsJSON string    `json:"-"`            // AES-256 encrypted Google service account JSON
	CreatedAt          time.Time `json:"created_at"`
}

type Store struct {
	db *sql.DB
}

func NewStore(dbPath string) (*Store, error) {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return &Store{db: db}, nil
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS tenants (
			id                   TEXT PRIMARY KEY,
			name                 TEXT NOT NULL,
			sheet_id             TEXT NOT NULL,
			allowed_host         TEXT NOT NULL DEFAULT '',
			encrypted_creds_json TEXT NOT NULL DEFAULT '',
			created_at           DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	return err
}

func (s *Store) Create(name, sheetID, allowedHost, encryptedCredsJSON string) (*Tenant, error) {
	t := &Tenant{
		ID:                 uuid.NewString(),
		Name:               name,
		SheetID:            sheetID,
		AllowedHost:        allowedHost,
		EncryptedCredsJSON: encryptedCredsJSON,
		CreatedAt:          time.Now(),
	}
	_, err := s.db.Exec(
		`INSERT INTO tenants (id, name, sheet_id, allowed_host, encrypted_creds_json) VALUES (?, ?, ?, ?, ?)`,
		t.ID, t.Name, t.SheetID, t.AllowedHost, t.EncryptedCredsJSON,
	)
	if err != nil {
		return nil, fmt.Errorf("insert tenant: %w", err)
	}
	return t, nil
}

func (s *Store) GetByID(id string) (*Tenant, error) {
	row := s.db.QueryRow(
		`SELECT id, name, sheet_id, allowed_host, encrypted_creds_json, created_at FROM tenants WHERE id = ?`, id,
	)
	t := &Tenant{}
	err := row.Scan(&t.ID, &t.Name, &t.SheetID, &t.AllowedHost, &t.EncryptedCredsJSON, &t.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil // not found
	}
	if err != nil {
		return nil, fmt.Errorf("query tenant: %w", err)
	}
	return t, nil
}

func (s *Store) List() ([]*Tenant, error) {
	rows, err := s.db.Query(
		`SELECT id, name, sheet_id, allowed_host, encrypted_creds_json, created_at FROM tenants ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tenants []*Tenant
	for rows.Next() {
		t := &Tenant{}
		if err := rows.Scan(&t.ID, &t.Name, &t.SheetID, &t.AllowedHost, &t.EncryptedCredsJSON, &t.CreatedAt); err != nil {
			return nil, err
		}
		tenants = append(tenants, t)
	}
	return tenants, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}
