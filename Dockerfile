# --- build -------------------------------------------------------------------
FROM golang:1.26-alpine AS build
WORKDIR /src

# Cache modules first.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Pure-Go deps (lib/pq, zap) -> static binary, no cgo.
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/kazi-ancestry .

# --- run ---------------------------------------------------------------------
FROM alpine:3.20
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 kazi
WORKDIR /app

COPY --from=build /out/kazi-ancestry /usr/local/bin/kazi-ancestry
# Ship the SPA + the fictional sample as the seed fallback. Real data is supplied
# at runtime (mount web/family.local.json or set SEED_PATH) and is never baked in.
COPY web/ /app/web/
# No config file in the image — containers are configured via env (compose).
# The app falls back to built-in defaults + env when .configs/ is absent.

USER kazi
EXPOSE 5294
ENTRYPOINT ["kazi-ancestry"]
CMD ["serve"]
