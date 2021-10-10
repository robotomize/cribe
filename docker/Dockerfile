FROM golang:1.17 AS builder

RUN apt-get -qq update && apt-get -yqq install upx

ENV GO111MODULE=on \
  CGO_ENABLED=0 \
  GOOS=linux \
  GOARCH=amd64

WORKDIR /src

COPY . .

ENV BUILD_INFO_PACKAGE=github.com/bloops-games/bloops/internal/buildinfo
ENV BUILD_NAME=cribe-bot

RUN go build \
  -trimpath \
  -ldflags "-s -w -X $BUILD_INFO_PACKAGE.BuildTag=$(git describe --tags --abbrev=0) -X $BUILD_INFO_PACKAGE.Time=$(date -u '+%Y-%m-%d-%H:%M') -X $BUILD_INFO_PACKAGE.Name=$BUILD_NAME -extldflags '-static'" \
  -installsuffix cgo \
  -tags netgo \
  -o /bin/cribebot \
  ./cmd/cribe-bot

RUN strip /bin/cribebot
RUN upx -q -9 /bin/cribebot

RUN mkdir /data

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /bin/cribebot /bin/cribebot
COPY --from=builder /data /data

VOLUME /data

ENTRYPOINT ["/app/cribebot"]
