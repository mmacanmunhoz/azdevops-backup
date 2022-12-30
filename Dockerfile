FROM golang:alpine

WORKDIR /build

COPY ./src/main.go .

RUN go build -o main main.go

CMD ["./main"]