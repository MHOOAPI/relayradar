FROM golang:1.27-alpine AS api
WORKDIR /src
RUN apk add --no-cache git
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/relayradar ./cmd/api \
    && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/prober ./cmd/prober

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata wget \
    && addgroup -S app && adduser -S -G app app
WORKDIR /app
COPY --from=api /out/relayradar /app/relayradar
COPY --from=api /out/prober /app/prober
# Build the web bundle on the host first: cd web && npm run build
COPY web/dist /app/web
ENV WEB_DIR=/app/web \
    ADDR=:8080 \
    STORE=postgres
USER app
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=20s \
    CMD wget -qO- http://127.0.0.1:8080/healthz || exit 1
ENTRYPOINT ["/app/relayradar"]
