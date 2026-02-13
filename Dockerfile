FROM golang:1.25-alpine AS build

RUN apk add --no-cache git

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

ARG VERSION=dev
ARG BUILD_DATE=unknown

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags "-s -w -X gophkeeper/internal/version.Version=${VERSION} -X gophkeeper/internal/version.BuildDate=${BUILD_DATE}" \
    -o /out/gophkeeper-server ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/gophkeeper-server /app/gophkeeper-server

ENV GOPHKEEPER_ADDR=:8080
ENV GOPHKEEPER_DB=file:/data/gophkeeper.db?_pragma=busy_timeout(5000)
ENV GOPHKEEPER_TOKEN_TTL=24h

EXPOSE 8080
VOLUME ["/data"]

ENTRYPOINT ["/app/gophkeeper-server"]
