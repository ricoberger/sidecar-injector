FROM golang:1.23.2 AS build
WORKDIR /githubauth
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN export CGO_ENABLED=0 && make build-githubauth

FROM alpine:3.20.3
RUN apk update && apk add --no-cache ca-certificates
RUN mkdir /githubauth
COPY --from=build /githubauth/bin/githubauth /githubauth
WORKDIR /githubauth
USER nobody
ENTRYPOINT  [ "/githubauth/githubauth" ]
