FROM golang:1.24-alpine

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY main.go ./

RUN go build -o financial-forecaster

EXPOSE 8080

CMD ["./financial-forecaster"]

