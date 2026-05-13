# ====================== Builder Stage ======================
FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS builder

ARG TARGETPLATFORM
ARG BUILDPLATFORM

RUN echo "Building on ${BUILDPLATFORM} for ${TARGETPLATFORM}"

# 安装构建依赖
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# 复制 go mod 并下载依赖
COPY combined_refactor/go.mod combined_refactor/go.sum ./
RUN go mod download

# 复制源代码
COPY combined_refactor/ ./

# 根据目标平台设置编译参数
RUN case ${TARGETPLATFORM} in \
      "linux/amd64")  GOARCH=amd64  ;; \
      "linux/arm64")  GOARCH=arm64  ;; \
      "linux/arm/v7") GOARCH=arm   GOARM=7 ;; \
      *) echo "Unsupported platform: ${TARGETPLATFORM}"; exit 1 ;; \
    esac && \
    CGO_ENABLED=0 GOOS=linux GOARCH=${GOARCH} GOARM=${GOARM} \
    go build -ldflags="-s -w -extldflags '-static'" \
    -o /cfdata main.go

# ====================== Runtime Stage ======================
FROM --platform=$TARGETPLATFORM alpine:latest

RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -H -u 1000 cfdata

WORKDIR /app

COPY --from=builder /cfdata /usr/local/bin/cfdata

RUN mkdir -p /data && chown cfdata:cfdata /data

VOLUME /data
EXPOSE 13335

USER cfdata

ENTRYPOINT ["cfdata"]
CMD ["-port=13335"]
