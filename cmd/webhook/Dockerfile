FROM golang:1.18.1-alpine3.15 as build
ARG TARGETPLATFORM
ARG BUILDPLATFORM
RUN echo "Building on $BUILDPLATFORM, for $TARGETPLATFORM" > /log
RUN apk update && apk add git make
WORKDIR /webhook
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN make build-webhook

FROM alpine:3.15.4
RUN apk update && apk add --no-cache ca-certificates
RUN mkdir /webhook
COPY --from=build /webhook/bin/webhook /webhook
WORKDIR /webhook
USER nobody
ENTRYPOINT  [ "/webhook/webhook" ]
