FROM golang:1.24.0 AS build
WORKDIR /basicauth
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN export CGO_ENABLED=0 && make build-basicauth

FROM alpine:3.21.3
RUN apk update && apk add --no-cache ca-certificates
RUN mkdir /basicauth
COPY --from=build /basicauth/bin/basicauth /basicauth
WORKDIR /basicauth
USER nobody
ENTRYPOINT  [ "/basicauth/basicauth" ]
