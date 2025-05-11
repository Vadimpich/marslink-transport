FROM golang:1.24

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod tidy

COPY . .

RUN go build -o transport-service ./cmd

CMD ["./transport-service"]
