FROM golang:1.15.6-alpine3.12 as build
ARG TARGETPLATFORM
ARG BUILDPLATFORM
RUN echo "Building on $BUILDPLATFORM, for $TARGETPLATFORM" > /log
RUN apk update && apk add git make
WORKDIR /basicauth
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN make build-basicauth

FROM alpine:3.12.0
RUN apk update && apk add --no-cache ca-certificates
RUN mkdir /basicauth
COPY --from=build /basicauth/bin/basicauth /basicauth
WORKDIR /basicauth
USER nobody
ENTRYPOINT  [ "/basicauth/basicauth" ]
