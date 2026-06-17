FROM golang:1.26.3-alpine AS build

WORKDIR /src
RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/worker ./cmd/worker
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/bootstrap-admin ./cmd/bootstrap-admin

FROM alpine:3.22

WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata ffmpeg wget \
	&& addgroup -S aura \
	&& adduser -S -G aura aura \
	&& mkdir -p /app/storage/uploads \
	&& chown -R aura:aura /app

COPY --from=build /out/api /app/api
COPY --from=build /out/worker /app/worker
COPY --from=build /out/bootstrap-admin /app/bootstrap-admin
COPY --chown=aura:aura docs /app/docs
COPY --chown=aura:aura migrations /app/migrations

USER aura
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
	CMD wget -qO- http://127.0.0.1:8080/health >/dev/null || exit 1

CMD ["/app/api"]
