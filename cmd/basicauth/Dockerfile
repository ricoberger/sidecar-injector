FROM golang:1.22.0-alpine3.18 as build
RUN apk update && apk add git make
WORKDIR /basicauth
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN export CGO_ENABLED=0 && make build-basicauth

FROM alpine:3.19.1
RUN apk update && apk add --no-cache ca-certificates
RUN mkdir /basicauth
COPY --from=build /basicauth/bin/basicauth /basicauth
WORKDIR /basicauth
USER nobody
ENTRYPOINT  [ "/basicauth/basicauth" ]
