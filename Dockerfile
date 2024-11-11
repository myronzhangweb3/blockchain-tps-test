FROM golang:alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -a -installsuffix cgo -o generate_account ./cmd/generate_account/main.go
RUN CGO_ENABLED=0 go build -a -installsuffix cgo -o send_tx ./cmd/send_tx/main.go

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/generate_account .
COPY --from=builder /app/send_tx .

EXPOSE 8000
