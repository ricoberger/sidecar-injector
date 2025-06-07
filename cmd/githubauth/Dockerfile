FROM golang:1.24.4 AS build
WORKDIR /githubauth
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN export CGO_ENABLED=0 && make build-githubauth

FROM alpine:3.22.0
RUN apk update && apk add --no-cache ca-certificates
RUN mkdir /githubauth
COPY --from=build /githubauth/bin/githubauth /githubauth
WORKDIR /githubauth
USER nobody
ENTRYPOINT  [ "/githubauth/githubauth" ]
