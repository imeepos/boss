# 2026-08-21 MinIO 密码文件化 + Bucket 策略

## why

102 上 MinIO 生产实例（`boss-infra-minio-1`，端口 29000/29001）的 root 密码以**明文**形式出现在：

- `deployments/docker-compose.102.yml`
- `deployments/docker-compose.infra.yml`
- 进程环境变量（任何可 ssh 上 102 的用户 `cat /proc/<pid>/environ` 即可读）

且三个业务 bucket（`boss-uploads` / `boss-attachments` / `boss-tmp`）建出来即默认配置（私有但无版本控制、无生命周期），上传文件被覆盖后无法找回、临时文件永不清理。

业务侧（`boss-server` 应用）通过运行时配置 (`BOSS_MINIO_SECRET_KEY`) 拿密钥，**与 compose 里的明文是两套事实**——出现不一致时只能查文件而不查 git。本 note 把 compose 侧的事实清除，强制改用文件挂载方式注入密钥。

## 决定

1. **密码走 bind-mount 文件**，不再在 compose / 进程 env 出现明文：
   - 宿主机：`/etc/minio/secrets/root_user`、`/etc/minio/secrets/root_password`（600、root）
   - 容器内：同上路径（容器内 root 默认）
   - compose 用 long-syntax `bind` 直接挂载，不用 docker secrets 块（之前试过 docker secrets，UBI Micro 镜像挂不上，简化成 bind）
   - MinIO 自动读取 `MINIO_ROOT_USER_FILE` / `MINIO_ROOT_PASSWORD_FILE` 环境变量覆盖 `MINIO_ROOT_USER` / `MINIO_ROOT_PASSWORD`

2. **当前不轮换密码**，保留 `boss12345`：
   - `boss-server` 端默认值在 `internal/pkg/config/config.go:171` 也硬编码 `boss12345`
   - `deployments/docker-compose.backup.yml` 两处 `mc alias set` 硬编码 `boss12345`
   - `deployments/config.example.yaml:24` 也是 `boss12345`
   - 改密要 4 处一起改 + 重启 minio + 热加载 boss-server 配置 + 重建备份镜像，风险/收益不匹配，留待业务侧推动

3. **三个 bucket 统一策略**：
   - 全部：`access = private` + `versioning = enabled`
   - `boss-tmp`：当前版本 7 天后删除（避免临时文件堆积）
   - `boss-attachments`：非当前版本保留 30 天、保留 3 个版本（防误删兜底，给业务侧 1 个月的回滚窗口）
   - `boss-uploads`：未设生命周期，业务方决定保留时长

## 放弃了什么

- **docker secrets 块**（`top-level secrets:` + `secrets: [..]` in service）：试过在 102 上 `docker compose config` 渲染出来挂载正常，但实际 `Recreate` 后容器内 `/run/secrets/` 不存在。原因未深入追（怀疑 docker compose v2.26.1 + UBI Micro 镜像的兼容性问题）。**已转 bind mount，简洁可靠**，唯一的代价是密码文件路径由人来保证安全（docker secrets 块本可由 docker daemon 管权限，但实际没用上）。
- **轮换密码**：见上。改了等于半套，业务侧一半。
- **公共读 bucket**：业务侧走 Presigned URL，不开 `anonymous=download`。

## 后续要做的事

- [ ] 业务方推动时一次性轮换：`/etc/minio/secrets/root_password` 改强密码 + `config.go:171` + `config.example.yaml:24` + `docker-compose.backup.yml` 两处
- [ ] 给 `boss-server` 的 storageconfig 管理页加"重置 MinIO 凭证"入口，调用 admin API 推送新密码
- [ ] Nginx 反代 `39091` 控制台目前允许 `192.168.0.0/16` + `127.0.0.1` + `10.0.0.0/8`，白名单按需收紧