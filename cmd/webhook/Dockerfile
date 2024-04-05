FROM golang:1.22.1-alpine3.18 as build
RUN apk update && apk add git make
WORKDIR /webhook
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN export CGO_ENABLED=0 && make build-webhook

FROM alpine:3.19.1
RUN apk update && apk add --no-cache ca-certificates
RUN mkdir /webhook
COPY --from=build /webhook/bin/webhook /webhook
WORKDIR /webhook
USER nobody
ENTRYPOINT  [ "/webhook/webhook" ]
