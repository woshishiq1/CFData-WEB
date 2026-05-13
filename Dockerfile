# ====================== Builder Stage ======================
# 1. 明确指定 1.25.4 版本，防止拉取到更新的 1.25.x 导致行为变化
FROM --platform=$BUILDPLATFORM golang:1.25.4-alpine AS builder

ARG TARGETPLATFORM
# 2. 强制使用本地编译器，不自动升级工具链，解决 go.mod 版本冲突问题
ENV GOTOOLCHAIN=local

RUN apk add --no-cache git ca-certificates

WORKDIR /app

# 复制 go.mod 并根据需要下载依赖
COPY combined_refactor/go.mod combined_refactor/go.sum ./
RUN go mod download

COPY combined_refactor/ ./

# 3. 优化构建逻辑：确保 GOARM 只在 arm/v7 时生效，避免干扰其他架构
RUN export GOOS=linux && \
    case ${TARGETPLATFORM} in \
      "linux/amd64")  export GOARCH=amd64 GOARM="" ;; \
      "linux/arm64")  export GOARCH=arm64 GOARM="" ;; \
      "linux/arm/v7") export GOARCH=arm   GOARM=7   ;; \
      *) echo "Unsupported platform: ${TARGETPLATFORM}"; exit 1 ;; \
    esac && \
    echo "Building for $GOOS/$GOARCH with GOARM=$GOARM" && \
    CGO_ENABLED=0 go build -ldflags="-s -w -extldflags '-static'" -o /cfdata .

# ====================== Runtime Stage ======================
FROM --platform=$TARGETPLATFORM alpine:latest

# 安装必要依赖
RUN apk add --no-cache ca-certificates tzdata \
    && mkdir -p /data && chmod 777 /data

# 从编译阶段拷贝二进制文件
COPY --from=builder /cfdata /usr/local/bin/cfdata

WORKDIR /data
VOLUME /data
EXPOSE 13335

# 赋予执行权限（保险起见）
RUN chmod +x /usr/local/bin/cfdata

USER root

ENTRYPOINT ["cfdata"]
CMD ["-port=13335"]
