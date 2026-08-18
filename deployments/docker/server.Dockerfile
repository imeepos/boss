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
USER app
ENTRYPOINT ["boss-server"]
