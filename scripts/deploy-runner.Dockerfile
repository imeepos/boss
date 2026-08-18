# 部署专用 runner 镜像:git(基座自带) + docker CLI + compose 插件,挂宿主 docker.sock 用。
FROM 192.168.0.102:5000/runner:bookworm
RUN apt-get update && apt-get install -y --no-install-recommends docker.io \
    && rm -rf /var/lib/apt/lists/*
COPY docker-compose-linux-x86_64 /usr/local/bin/docker-compose
RUN chmod +x /usr/local/bin/docker-compose \
    && mkdir -p /usr/libexec/docker/cli-plugins \
    && ln -s /usr/local/bin/docker-compose /usr/libexec/docker/cli-plugins/docker-compose
