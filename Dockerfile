FROM golang:1.22-alpine AS builder

# gcc needed for go-sqlite3 (cgo)
RUN apk add --no-cache gcc musl-dev

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o feedbackpulse ./cmd/server

# --- Final minimal image ---
FROM alpine:3.19
RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /app/feedbackpulse .

# Data dir for SQLite
RUN mkdir -p /app/data

ENV PORT=8080
EXPOSE $PORT

CMD ["./feedbackpulse"]
