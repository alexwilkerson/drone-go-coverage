FROM golang:1.23.6-alpine AS builder

WORKDIR /app
COPY . .
RUN go build -o /bin/drone-go-coverage ./cmd/plugin

RUN chmod +x /bin/drone-go-coverage
ENTRYPOINT ["/bin/drone-go-coverage"]
