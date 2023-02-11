FROM golang:1.18-alpine as base

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./

RUN go build -o /hnf


FROM alpine:3.17

COPY --from=base /hnf /hnf

EXPOSE 8080

CMD ["/hnf"]