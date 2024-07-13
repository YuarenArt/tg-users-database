ARG GO_VERSION=1.21
ARG ALPINE_VERSION=latest
FROM golang:${GO_VERSION}-alpine as builder

RUN apk add --no-cache git gcc musl-dev

WORKDIR /usr/src/tg-database

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ENV CGO_ENABLED=1
RUN go build -o /main .

FROM alpine:${ALPINE_VERSION}

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /main .

ENV GIN_MODE=release

EXPOSE 8082/tcp

CMD ["./main"]