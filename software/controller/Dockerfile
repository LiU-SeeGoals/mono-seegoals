FROM golang:1.21-alpine

RUN apk add --no-cache \
    bash \
    git \
    gcc \
    g++ \
    musl-dev \
    pkgconf \
    linux-headers \
    sdl2-dev

WORKDIR /var/controller

COPY . .

RUN go mod download

CMD ["go", "run", "cmd/main.go"]
