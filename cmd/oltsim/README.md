# OLT 设备仿真接入程序(oltsim)

> 目的:用软件模拟"真实厂家 OLT/光猫",对接 BOSS 的网元侧契约,
> 用于测试 ①连通性(provisioner→设备) ②业务完整性(12 环节+预下发+RADIUS 鉴权闭环)。
> 定位:联调/验收工具,非生产服务;随仓库维护,新设备协议面适配在此扩展。

## 双端角色(社区模式:设备仿真 = 协议双端)

| 端 | 模拟对象 | 协议 | 契约面 |
|---|---|---|---|
| Telnet CLI 服务端 | 厂家 OLT 管理面 | TCP 行协议(login:/password:/provision apply) | 镜像 `internal/domain/provision/telnet.go` 的 `TelnetExecutor` 交互面 |
| RADIUS 客户端 | 光猫(ONT)上线 | RFC 2865 Access-Request | 对齐 `internal/domain/aaa/radius` 服务端(User-Name=LOID) |

## 用法

```bash
# 起模拟器(默认 Telnet :2323,HTTP :8081)
go run ./cmd/oltsim

# 故障注入:登录必拒(测 TelnetExecutor 登录失败路径)
go run ./cmd/oltsim --deny-login

# 强制下发全失败(测 FAILED 留痕+重试)
go run ./cmd/oltsim --force-err

# 光猫上线(向 boss-aaa 1812 发 Access-Request,User-Name=LOID)
curl -XPOST localhost:8081/ont/online -d '{"loid":"LOID-xxx"}'
```

## 与生产代码的对应

- 下发命令格式与 `TelnetExecutor.Exec` 完全一致(template=%d task=%s event=%s)
- 回执 OK/ERR 语义与 provision 域 `telnet.go` 的 readUntil("OK") 校验一致
- Access-Request 仅需 User-Name(LOID),解锁 boss-aaa `serveAuth` 分支
- 带宽/会话超时从 Access-Accept 的 FramedPool/SessionTimeout 读取,验证套餐生效

## 验收轨迹(mainchain 对接)

- provision_tasks 状态 DONE = 模拟器收到下发且回 OK
- auth_logs result=SUCCESS = 光猫上线认证放行(入网凭证有效)
- products.bandwidth 出现在 Access-Accept = 套餐带宽生效