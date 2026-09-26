# syntax=docker/dockerfile:1

# ---- Build stage ----
FROM golang:1.26-bookworm AS builder

WORKDIR /app

# 複製 go.mod / go.sum 並下載依賴（放前面以利用 layer cache）
COPY go.mod go.sum ./
RUN go mod download

# 複製專案全部檔案
COPY . .

# CGO_ENABLED=0: the production SQLite driver (modernc.org/sqlite) is pure
# Go, so no C toolchain is needed at runtime. A fully static binary lets us
# ship on a minimal distroless base with no shell or package manager - much
# smaller image, much smaller attack surface. See docs/AUDIT.md Q5.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /wallet-api ./main

# ---- Runtime stage ----
# distroless "nonroot" images run as a non-root user (uid 65532) by
# default and contain no shell, package manager, or other tools an
# attacker could use after a compromise - just enough libc-free runtime
# to execute a static Go binary.
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=builder /wallet-api ./wallet-api

EXPOSE 8080

ENTRYPOINT ["./wallet-api"]
