# ===== Build Stage =====
FROM golang:1.22-alpine AS builder

# 安装依赖
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# 下载依赖（利用缓存）
COPY go.mod go.sum ./
RUN go mod download

# 复制源码
COPY . .

# 编译
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s" -o /app/server ./cmd/server

# ===== Runtime Stage =====
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# 复制二进制和配置
COPY --from=builder /app/server .
COPY --from=builder /app/configs ./configs

# 时区
ENV TZ=Asia/Shanghai

# 非root用户运行
RUN adduser -D -u 1000 appuser
USER appuser

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health/live || exit 1

CMD ["./server"]
