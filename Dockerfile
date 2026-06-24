# --- 阶段 1: 编译 ---
FROM --platform=$BUILDPLATFORM golang:1.25.4-alpine AS builder

ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT
# 接收从 GitHub Action 传进来的版本号，默认为 dev
ARG APP_VERSION=dev

WORKDIR /app
RUN apk add --no-cache git ca-certificates

# 拷贝依赖
COPY combined_refactor/go.mod combined_refactor/go.sum ./
RUN go mod download

# 拷贝源码
COPY combined_refactor/ ./

# 🌟 核心编译命令修改点：
# 1. 增加了 GODEBUG=netdns=go，防止 Alpine 环境下偶发的 DNS 解析失败问题。
# 2. 如果你打算用这个 Dockerfile 编译 Android 容器镜像，可以加上 -buildmode=pie（非安卓容器可不加）。
RUN GOTOOLCHAIN=local CGO_ENABLED=0 GODEBUG=netdns=go \
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
