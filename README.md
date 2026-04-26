# FeedbackPulse Backend

Go backend for FeedbackPulse — receives widget feedback, transcribes voice via Whisper microservice, and appends rows to Google Sheets.

---

## Environment Variables

| Variable | Required | Description |
|---|---|---|
| `GOOGLE_CREDENTIALS_JSON` | ✅ | Full JSON of your Google service account key |
| `ADMIN_KEY` | ✅ | Secret key to protect `/admin` routes |
| `WHISPER_URL` | ✅ | URL of your Whisper microservice |
| `WHISPER_SECRET` | ✅ | `X-Api-Secret` value for the Whisper service |
| `DB_PATH` | optional | SQLite file path (default: `./data/feedbackpulse.db`) |
| `PORT` | optional | HTTP port (default: `8080`) |

---

## Google Sheets Setup

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a project → Enable **Google Sheets API**
3. Create a **Service Account** → Download the JSON key
4. Set `GOOGLE_CREDENTIALS_JSON` to the full contents of that JSON
5. When registering a tenant, share their Google Sheet with the service account email (found in the JSON as `client_email`) with **Editor** access

---

## Local Development

```bash
# Set required env vars
export GOOGLE_CREDENTIALS_JSON='{ ...your service account json... }'
export ADMIN_KEY=localadminsecret
export WHISPER_URL=https://whisper-microservice.onrender.com
export WHISPER_SECRET=your-whisper-secret

# Run
go run ./cmd/server
```

---

## API Reference

### `GET /health`
```json
{ "status": "ok", "service": "feedbackpulse-backend" }
```

---

### `POST /feedback`
Submit feedback from the widget.

**Body:** `multipart/form-data`

| Field | Required | Description |
|---|---|---|
| `site_id` | ✅ | Tenant site ID |
| `emoji` | ✅ | `happy`, `neutral`, or `sad` |
| `email` | ❌ | Logged-in user's email |
| `audio` | ❌ | Audio file (webm/wav/mp3, max 5MB) |

**Response:**
```json
{ "status": "ok", "transcript": "Your service was great!" }
```

---

### `POST /admin/tenants`
Register a new business tenant.

**Header:** `X-Admin-Key: <your-admin-key>`

**Body:** `application/json`
```json
{
  "name": "Acme Bakery",
  "sheet_id": "1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgVE2upms",
  "allowed_host": "acmebakery.com"
}
```

**Response:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Acme Bakery",
  "sheet_id": "1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgVE2upms",
  "allowed_host": "acmebakery.com",
  "created_at": "2024-01-15T10:30:00Z"
}
```

Save the returned `id` — that's the `site_id` to use in the widget script tag.

---

### `GET /admin/tenants`
List all registered tenants.

**Header:** `X-Admin-Key: <your-admin-key>`

---

## Deploy to Render

1. Push repo to GitHub
2. Render → New → Blueprint → connect repo (reads `render.yaml`)
3. After first deploy, go to **Environment** tab and manually set:
   - `GOOGLE_CREDENTIALS_JSON` (paste the full service account JSON)
   - `WHISPER_SECRET` (from your Whisper service dashboard)
4. Hit **Save** → service redeploys automatically

> **Note:** Render free tier spins down after inactivity. The SQLite disk persists across restarts.
