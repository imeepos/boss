# 2026-08-21 hostctl sidecar + MinIO 密钥热轮换

## why

102 上 MinIO root 密码（`boss12345`）轮换需要同时修改:
- 宿主机 `/etc/minio/secrets/root_password`
- 重启 minio 容器
- boss-server 侧 `biz_params` 中的 `minio.secretKey`（AES-GCM 加密）

手工 SSH 操作有 4 个文件 + 1 个容器要改，窗口期长、易遗漏。
需要一个一键 API 让管理员在后台页面完成整个轮换。

## 决定

1. **hostctl sidecar**（`cmd/hostctl/main.go`）:
   - 静态编译 Go 二进制，~6MB，systemd unit 管理
   - 监听 `0.0.0.0:39093`（docker bridge 内 boss-server 可达）
   - HMAC-SHA256 鉴权（预共享 key `/etc/boss-host-ctl/hmac.key`，0600）
   - `POST /v1/minio/rotate-secret` → 原子写密码文件 + `docker compose up -d --force-recreate --no-deps minio` + 健康检查
   - `GET /healthz` 无鉴权

2. **boss-server 侧**:
   - `internal/pkg/hostctl/client.go`: HMAC 客户端
   - `POST /admin/storage-config/rotate-secret`: 校验密码 → 加密存 biz_params → 调 hostctl
   - HMAC key 通过 bind-mount 注入容器（`/run/hostctl-hmac/hmac.key`），不进 git/env
   - `config.go` 新增 `readEnvOrFile()` helper:env 优先、文件兜底

3. **前端**:
   - storageconfig 页面加"轮换 MinIO 根密码"按钮 + 对话框
   - 三语言（中/英/马）

## 放弃了什么

- **docker secrets 顶层块**:UBI Micro 镜像挂不上（已验证），改 bind mount
- **给 boss-server 挂 docker.sock**:等于给业务容器 root 权限，安全风险不可接受
- **ssh 远程脚本方案**:boss-server 容器无 ssh 客户端，凭据管理更复杂
- **HMAC key 内联 env**:key 会进 git history 和 `docker inspect` 输出，改 bind-mount 文件

## 端到端验证

```
POST /api/admin/v1/storage-config/rotate-secret {"secret":"e2e_test_pass"}
→ {"ok":true,"rotatedAt":"2026-08-21T12:42:51Z","duration":"6.834387425s"}

mc alias set t http://192.168.0.102:29000 boss e2e_test_pass → 成功，看到 3 个 bucket
mc alias set old http://192.168.0.102:29000 boss boss12345 → 失败（旧密码已失效）

POST .../rotate-secret {"secret":"boss12345"} → 恢复成功
```

## 后续

- [ ] 生产环境首次使用时生成真正强密码替换 `boss12345`
- [ ] nginx 39093 端口加限源（当前 `0.0.0.0:39093` 对外暴露，仅 HMAC 保护）
- [ ] hostctl 加 metrics 端点（轮换次数、耗时）