FROM golang:1.24-alpine3.20 AS builder

ARG Version

RUN apk --no-cache add git alpine-sdk

WORKDIR /sources
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -mod vendor -ldflags "-X main.Version=sources-$Version -s -w" -o monday /sources/cmd

FROM alpine:3.21.3

LABEL name="monday"
LABEL description="A dev tool for microservice developers to run local applications and/or forward others from/to Kubernetes SSH or TCP"

WORKDIR /

COPY --from=builder /sources/monday monday

ENTRYPOINT ["/monday"]
