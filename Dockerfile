FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/kiwis-worker ./cmd/kiwis-worker/main.go

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Run as non-root user
RUN addgroup -S app && adduser -S -G app app

COPY --from=builder /out/kiwis-worker /app/kiwis-worker
COPY migrations /app/migrations

RUN chown -R app:app /app
USER app

ENTRYPOINT ["/app/kiwis-worker"]
