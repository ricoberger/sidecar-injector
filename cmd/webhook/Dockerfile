FROM golang:1.22.5 as build
WORKDIR /webhook
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN export CGO_ENABLED=0 && make build-webhook

FROM alpine:3.20.2
RUN apk update && apk add --no-cache ca-certificates
RUN mkdir /webhook
COPY --from=build /webhook/bin/webhook /webhook
WORKDIR /webhook
USER nobody
ENTRYPOINT  [ "/webhook/webhook" ]
