FROM golang:1.21

WORKDIR /app

COPY . .

RUN go mod tidy
RUN go build -o transport-service ./cmd

CMD ["./transport-service"]
