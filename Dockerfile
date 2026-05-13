# --- 阶段 1: 编译 ---
FROM --platform=$BUILDPLATFORM golang:1.25.4-alpine AS builder

ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT
# 新增：接收从 GitHub Action 传进来的版本号
ARG APP_VERSION=latest

WORKDIR /app
RUN apk add --no-cache git ca-certificates

# 拷贝依赖
COPY combined_refactor/go.mod combined_refactor/go.sum ./
RUN go mod download

# 拷贝源码
COPY combined_refactor/ ./

# 核心：在 ldflags 中注入版本号
# 注意：确认你的代码里变量名是 appVersion 还是 version
RUN GOTOOLCHAIN=local CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH} \
    GOARM=${TARGETVARIANT#v} \
    go build -ldflags "-s -w -X main.appVersion=${APP_VERSION}" -o cfdata-app .

# --- 阶段 2: 运行 ---
FROM alpine:latest
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/cfdata-app .
EXPOSE 13335
ENTRYPOINT ["./cfdata-app"]
