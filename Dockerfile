# ====================== Builder Stage ======================
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder

ARG TARGETPLATFORM
RUN echo "Building on ${BUILDPLATFORM} for ${TARGETPLATFORM}"

RUN apk add --no-cache git ca-certificates

WORKDIR /app

# 复制 go mod 并下载依赖
COPY go.mod go.sum ./
RUN go mod download

# 复制全部源代码
COPY . .

# 编译多架构
RUN case ${TARGETPLATFORM} in \
      "linux/amd64")  GOARCH=amd64  ;; \
      "linux/arm64")  GOARCH=arm64  ;; \
      "linux/arm/v7") GOARCH=arm   GOARM=7 ;; \
      *) echo "Unsupported platform: ${TARGETPLATFORM}"; exit 1 ;; \
    esac && \
    CGO_ENABLED=0 GOOS=linux GOARCH=${GOARCH} GOARM=${GOARM} \
    go build -ldflags="-s -w -extldflags '-static'" -o /cfdata main.go

# ====================== Runtime Stage ======================
FROM --platform=$TARGETPLATFORM alpine:latest

RUN apk add --no-cache ca-certificates tzdata \
    && mkdir -p /root/cfdata-web \
    && chmod 777 /root/cfdata-web

# 把二进制放到数据目录
COPY --from=builder /cfdata /root/cfdata-web/cfdata

WORKDIR /root/cfdata-web

VOLUME /root/cfdata-web
EXPOSE 13335

USER root

ENTRYPOINT ["./cfdata"]
CMD ["-port=13335"]
