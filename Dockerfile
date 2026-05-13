# ====================== Builder Stage ======================
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder

ARG TARGETPLATFORM
RUN echo "Building on ${BUILDPLATFORM} for ${TARGETPLATFORM}"

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN case ${TARGETPLATFORM} in \
      "linux/amd64")  GOARCH=amd64  ;; \
      "linux/arm64")  GOARCH=arm64  ;; \
      "linux/arm/v7") GOARCH=arm   GOARM=7 ;; \
      *) echo "Unsupported platform: ${TARGETPLATFORM}"; exit 1 ;; \
    esac && \
    CGO_ENABLED=0 GOOS=linux GOARCH=${GOARCH} GOARM=${GOARM} \
    go build -ldflags="-s -w -extldflags '-static'" -o /cfdata cfdata.go

# ====================== Runtime Stage ======================
FROM --platform=$TARGETPLATFORM alpine:latest

RUN apk add --no-cache ca-certificates tzdata \
    && mkdir -p /data \
    && chmod 777 /data

# 二进制放到系统目录
COPY --from=builder /cfdata /usr/local/bin/cfdata

# 默认工作目录设为数据目录
WORKDIR /data

VOLUME /data
EXPOSE 13335

USER root

ENTRYPOINT ["cfdata"]
CMD ["-port=13335"]
