# Stage 1: Builder
FROM golang:1.21-alpine AS builder

# 安裝建置依賴
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# 複製 go mod 檔案並下載依賴
COPY go.mod go.sum ./
RUN go mod download

# 複製源碼
COPY . .

# 建置執行檔
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X main.Version=${VERSION:-dev}" \
    -a -installsuffix cgo \
    -o /ralph-loop \
    ./cmd/ralph-loop

# Stage 2: Runtime
FROM alpine:latest

# 安裝執行時依賴
RUN apk --no-cache add \
    ca-certificates \
    nodejs \
    npm \
    git \
    && npm install -g @github/copilot \
    && rm -rf /var/cache/apk/*

# 建立非 root 使用者
RUN addgroup -g 1000 ralph && \
    adduser -D -u 1000 -G ralph ralph

# 複製執行檔
COPY --from=builder /ralph-loop /usr/local/bin/ralph-loop

# 設定工作目錄
WORKDIR /workspace

# 切換到非 root 使用者
USER ralph

# 健康檢查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ralph-loop version || exit 1

# 預設入口點
ENTRYPOINT ["ralph-loop"]

# 預設命令
CMD ["--help"]
