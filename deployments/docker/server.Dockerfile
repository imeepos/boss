# 多阶段构建:全部部署物二进制打进同一镜像,入口默认 server。
FROM golang:1.25-alpine AS build
ENV GOPROXY=https://goproxy.cn,direct
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download || true
COPY . .
ARG BINARIES="server collector provisioner report aaa"
# 授权公钥(B 档强制门禁):仅 server 注入;空=开发态(门禁不启用,打 ALERT 日志)。
# 生产 CI 必须传 BOSS_LICENSE_PUBLIC_KEY_HEX;空公钥镜像须显式传
# ALLOW_DEV_LICENSE=1 声明开发态,否则构建即失败(0010 事故 fail-fast)。
ARG BOSS_LICENSE_PUBLIC_KEY_HEX=""
ARG ALLOW_DEV_LICENSE=""
RUN set -e; if [ -z "$BOSS_LICENSE_PUBLIC_KEY_HEX" ] && [ "$ALLOW_DEV_LICENSE" != "1" ]; then \
    echo "[server.Dockerfile] FAIL: BOSS_LICENSE_PUBLIC_KEY_HEX 为空且未声明 ALLOW_DEV_LICENSE=1——拒绝产出无门禁镜像(postmortem 0010)" >&2; \
    exit 1; \
  fi
RUN set -e; for b in ${BINARIES}; do \
    LDFLAGS="-s -w"; \
    if [ "$b" = "server" ] && [ -n "$BOSS_LICENSE_PUBLIC_KEY_HEX" ]; then \
      LDFLAGS="$LDFLAGS -X github.com/ymm-001/boss/internal/pkg/buildinfo.LicensePublicKeyHex=$BOSS_LICENSE_PUBLIC_KEY_HEX"; \
    fi; \
    CGO_ENABLED=0 go build -ldflags="$LDFLAGS" -o /out/boss-$b ./cmd/$b; \
  done

FROM alpine:3.20
RUN adduser -D app && apk add --no-cache ca-certificates
COPY --from=build /out/ /usr/local/bin/
# migrations 必须随镜像走:server 启动按 migrations/ 目录幂等补迁(漏拷=静默停在旧版,102 曾停于 000041)。
WORKDIR /app
COPY migrations /app/migrations
# 迁移文件源权限可能是 600(git/编辑器差异),容器以 app 用户运行必须可读,统一放开读权限。
RUN chmod -R a+rX /app/migrations
# 备份归档目录(BOSS_BACKUP_DIR):镜像内预建属主 app,命名卷首挂时继承该属主。
RUN mkdir -p /data/backups && chown app:app /data/backups
# 授权证书目录(BOSS_LICENSE_CERT_PATH 默认 /var/lib/boss/license.json):
# 预建属主 app——运行用户无权在 /var/lib 下建目录,激活落盘曾因此全链路失败。
RUN mkdir -p /var/lib/boss && chown app:app /var/lib/boss
USER app
ENTRYPOINT ["boss-server"]
