FROM golang:1.22-alpine AS build

WORKDIR /app

COPY go.mod ./
COPY main.go ./

RUN go build -o monitor

FROM alpine:latest
WORKDIR /app
COPY --from=build /app/monitor .
CMD ["./monitor"]