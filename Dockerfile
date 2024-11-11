FROM golang:alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -a -installsuffix cgo -o tps-test main.go

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/tps-test .

EXPOSE 8000