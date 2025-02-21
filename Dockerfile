FROM golang:1.24 as builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o action ./cmd/main.go

FROM alpine:3.16
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/action /action
ENTRYPOINT ["/action"]
