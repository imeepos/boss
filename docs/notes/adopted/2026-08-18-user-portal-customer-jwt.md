# 2026-08-18 用户端门户客户 JWT 方案

## 决策

客户门户(/api/v1, api/openapi/user.yaml)的客户 JWT 复用 `auth.Manager` 签发(同密钥、同 TTL、同 issuer="boss"),
不另建密钥体系。客户身份编码约定:

- `Claims.AccountID = 0`(与 APIKeyAuth customer 主体同约定:RBAC 快照按 accountID 查询恒不命中,菜单门禁恒 403);
- `Claims.RoleCode = "customer"`(roles 表 7 角色码之一);
- `Claims.Username = "cust/<customerID>/<phone>"`(登录名位置编码客户身份,`customerIDFromToken` 回读)。

理由:auth.Manager 密钥私有不可提取,自建第二套 JWT 密钥会造成密钥配置分叉与轮换不同步;复用 Manager 使客户 token
天然被既有 `middleware.Authn` 校验,无需改 middleware/auth 包(任务约束:只允许新文件)。

## 放弃了什么

- 放弃独立 customer JWT Manager(独立密钥):密钥配置分叉,运维两套轮换,且无法经现有 Authn 校验。
- 放弃负数 AccountID 编码(曾评审计):正/负数值均可能与真实账号 id 撞车引发 RBAC 误判;AccountID=0 从机制上恒拒。
- 放弃在 Claims 增加显式 Subject 字段:需改 internal/pkg/auth(超出本任务文件边界),作为后续演进方向——
  auth.Manager 提供 SignCustomer(customerID) 扩展点后,Username 编码即废弃。

## 风险与边界

- Username 编码是过渡 hack;在此之前任何非 customer 角色 token 不能解出客户 ID(`customerIDFromToken` 恒 0 → 401)。
- 客户 token 打到 admin 菜单端点:AccountID=0 无任何 RBAC 权限,恒 403,无越权面。
