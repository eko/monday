FROM golang:1.16.4-alpine3.12 as builder

ARG Version

RUN apk --no-cache add git alpine-sdk

WORKDIR /sources
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -mod vendor -ldflags "-X main.Version=sources-$Version -s -w" -o monday /sources/cmd

FROM alpine:3.13.5

LABEL name="monday"
LABEL description="A dev tool for microservice developers to run local applications and/or forward others from/to Kubernetes SSH or TCP"

WORKDIR /

COPY --from=builder /sources/monday monday

ENTRYPOINT ["/monday"]
