# 部署专用 runner 镜像:git(基座自带) + docker CLI + compose 插件,挂宿主 docker.sock 用。
# 构建上下文 = scripts/ops/(compose 二进制与注册表凭据 JSON 均固化在仓库内,不再依赖
# 102 的 /tmp/drctx2 与宿主 ~/.docker/config.json——2026-09-06 镜像被 prune 后手工重建
# 即因这两样缺失而三步才修通,见 ISSUE.md CI/deploy-102):
#   docker build -f scripts/deploy-runner.Dockerfile scripts/ops
# 重建由 deploy workflow 首步 runner-image-guard job 自动触发
# (scripts/ops/deploy-runner-guard.sh,缺失即建、存在即 no-op)。
FROM 192.168.0.102:5000/runner:bookworm
RUN apt-get update && apt-get install -y --no-install-recommends docker.io \
    && rm -rf /var/lib/apt/lists/*
COPY docker-compose-linux-x86_64 /usr/local/bin/docker-compose
RUN chmod +x /usr/local/bin/docker-compose \
    && mkdir -p /usr/libexec/docker/cli-plugins /usr/local/lib/docker/cli-plugins /root/.docker/cli-plugins \
    && for d in /usr/libexec/docker/cli-plugins /usr/local/lib/docker/cli-plugins /root/.docker/cli-plugins; \
       do ln -sf /usr/local/bin/docker-compose $d/docker-compose; done

# 注册表凭据烤入:job 容器里的 docker CLI push/pull 时读 ~/.docker/config.json 构造
# X-Registry-Auth 发给宿主 daemon(私有仓库 htpasswd 鉴权,匿名一律 401)。凭据仅限
# 内网网段 192.168.0.102:5000,内容与轮换说明见 scripts/ops/deploy-registry-config.README.md;
# 2026-09-06 事故中重建镜像缺此文件曾致无法推送注册表。
COPY deploy-registry-config.json /root/.docker/config.json
