# deploy-registry-config.json 说明(2026-09-06 固化)

## 用途

deploy-runner 镜像内 /root/.docker/config.json 的仓库内固化副本。deploy-102 workflow
的 job 容器里,docker CLI 执行 push/pull 时读取该文件构造 X-Registry-Auth 请求头发给
宿主 daemon——102 私有注册表 192.168.0.102:5000 为 htpasswd 鉴权,匿名 pull/push 一律
401(2026-09-06 事故中 runner 侧拉镜像即撞 "no basic auth credentials")。

两处消费方:

1. scripts/deploy-runner.Dockerfile 构建时 COPY 进镜像(重建后的镜像开箱可用);
2. scripts/ops/deploy-runner-guard.sh 运行时经 DOCKER_CONFIG 注入,供守护路径的
   docker pull(基座)/docker push(推回)使用。

## 网段边界(Lead 裁定 2026-09-06)

- 本文件只允许出现内网私有注册表 192.168.0.102:5000 的凭据;该注册表仅监听内网,
  凭据泄露面=能进 102 网段的机器。
- 本仓库托管于同网段 gitea(192.168.0.102:3001,私有),不经公网分发。
- **严禁**把任何公网/云厂商凭据写入本文件。102 宿主 ~/.docker/config.json 中另有
  volces 云仓库凭据,固化时已刻意剔除;若未来需要云仓库凭据,必须走 secret 通道,
  不得入库。
- 凭据格式为 docker config 标准 auth 字段(base64(user:pass)),轮换时同步更新此处,
  守护 job 下次重建 deploy-runner 镜像即自动带上新值。

## 同目录 compose 二进制

docker-compose-linux-x86_64(63MB,v2 静态二进制)同为构建上下文依赖,原只存在于
102 的 /tmp/drctx2(会消失的临时位置),现固化入仓库。sha256:

383ce6698cd5d5bbf958d2c8489ed75094e34a77d340404d9f32c4ae9e12baf0
