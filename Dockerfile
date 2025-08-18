FROM golang:1.25 AS builder

ARG APP_NAME
ARG VERSION

COPY . /src
WORKDIR /src

RUN GOEXPERIMENT=greenteagc GOEXPERIMENT=jsonv2 go build -ldflags "-X main.Version=${VERSION}" ./app/${APP_NAME}/cmd/server


FROM debian:stable-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
		ca-certificates  \
        netbase \
        && rm -rf /var/lib/apt/lists/ \
        && apt-get autoremove -y && apt-get autoclean -y

COPY --from=builder /src/server /app/server

WORKDIR /app

VOLUME ["/data/conf"]

CMD ["./server", "-conf", "/data/conf"]