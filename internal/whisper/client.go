package whisper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"time"
)

type Client struct {
	baseURL    string
	secret     string
	httpClient *http.Client
}

type transcribeResponse struct {
	Transcript      string  `json:"transcript"`
	Language        string  `json:"language"`
	DurationSeconds float64 `json:"duration_seconds"`
}

func NewClient(baseURL, secret string) *Client {
	return &Client{
		baseURL: baseURL,
		secret:  secret,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Transcribe sends audio bytes to the Whisper microservice and returns the transcript.
// filename is used only to set the correct file extension (e.g. "audio.webm").
func (c *Client) Transcribe(audioBytes []byte, filename string) (string, error) {
	if len(audioBytes) == 0 {
		return "", nil
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filepath.Base(filename))
	if err != nil {
		return "", fmt.Errorf("create form file: %w", err)
	}
	if _, err := io.Copy(part, bytes.NewReader(audioBytes)); err != nil {
		return "", fmt.Errorf("write audio bytes: %w", err)
	}
	writer.Close()

	req, err := http.NewRequest("POST", c.baseURL+"/transcribe", body)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if c.secret != "" {
		req.Header.Set("X-Api-Secret", c.secret)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("whisper request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("whisper returned %d: %s", resp.StatusCode, string(b))
	}

	var result transcribeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode whisper response: %w", err)
	}

	return result.Transcript, nil
}
