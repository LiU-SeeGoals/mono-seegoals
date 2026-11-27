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

# Build context is software/ (see docker/docker-compose.yml). go.mod replaces
# proto_go with ../proto-messages/proto_go, so mirror that layout.
COPY proto-messages/proto_go /var/proto-messages/proto_go
COPY controller/go.mod controller/go.sum ./
RUN go mod download
COPY controller/ .

CMD ["go", "run", "cmd/main.go"]
