# 多阶段构建:全部部署物二进制打进同一镜像,入口默认 server。
FROM golang:1.25-alpine AS build
ENV GOPROXY=https://goproxy.cn,direct
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download || true
COPY . .
ARG BINARIES="server collector provisioner report aaa"
RUN set -e; for b in ${BINARIES}; do CGO_ENABLED=0 go build -o /out/boss-$b ./cmd/$b; done

FROM alpine:3.20
RUN adduser -D app && apk add --no-cache ca-certificates
COPY --from=build /out/ /usr/local/bin/
# migrations 必须随镜像走:server 启动按 migrations/ 目录幂等补迁(漏拷=静默停在旧版,102 曾停于 000041)。
WORKDIR /app
COPY migrations /app/migrations
USER app
ENTRYPOINT ["boss-server"]
