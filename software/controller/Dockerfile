FROM golang:1.21-alpine

# Install bash
RUN apk add --no-cache bash

COPY .bash_history /root/.bash_history
WORKDIR /var/controller

# Add rendering dependencies
RUN apk update && apk add go gcc libxcursor-dev libxrandr-dev libxinerama-dev libxi-dev linux-headers mesa-dev libstdc++ mesa-utils mesa-dri-gallium && apk add --virtual build-dependencies build-base gcc wget git

COPY . .
RUN go mod download
CMD ["go", "run", "cmd/main.go"]
