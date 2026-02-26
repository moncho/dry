# Multi-stage build for local development
FROM golang:alpine AS builder

RUN apk add --no-cache git

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /dry .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /dry /app/dry

CMD ["sleep", "1", "&&", "/app/dry"]
