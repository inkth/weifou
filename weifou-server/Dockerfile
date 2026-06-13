# —— 构建阶段 ——————————————————————————————————————————
FROM golang:1.25-alpine AS builder
WORKDIR /app

ENV GOPROXY="https://goproxy.cn,https://proxy.golang.org,direct"
COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/server ./cmd/server

# —— 运行阶段 ——————————————————————————————————————————
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata wget && \
    addgroup -g 1000 app && \
    adduser -u 1000 -G app -D app
WORKDIR /home/app

COPY --from=builder /out/server ./server
COPY .env.example ./.env.example

USER app
EXPOSE 8080
# 用 GET(不带 --spider,后者发 HEAD,而 /healthz 仅注册 GET → 会 404)
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 -O /dev/null http://localhost:8080/healthz || exit 1

CMD ["./server"]
