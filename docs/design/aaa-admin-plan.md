# 3A/AAA 管理与配置页面开发计划

## 目标

在现有真实 AAA 后端能力之上，形成可运维的后台管理闭环，不伪造 NAS、在线会话或运行指标数据源。

能力域声明：`【能力域】AAA 【Agent】AG-05 【internal 包】aaa 【阶段】7 【页面】aaa/aaadashboard、oss/loaccount、aaa/aaalog`。

## 一期交付范围

1. **AAA 总览**：基于真实 `/lo-accounts`、`/cdrs`、`/auth-logs` 聚合当前账号数、ACTIVE/SUSPENDED/CLOSED、话单数量、未入账数量、认证成功/失败数量；展示数据更新时间和刷新操作。
2. **认证账号管理**：在现有 LOID 列表上增加状态筛选、账号详情抽屉，展示客户/企业/区域/套餐/QoS/付费模式；停复机继续通过既有 billing 业务接口，页面只呈现已有后端能力，不直接改库。
3. **话单与认证日志**：保留现有双页签和契约路径，增加结果/入账状态筛选、详情抽屉和可识别的真实字段展示。
4. **配置边界**：本期不新增虚假的 AAA 配置存储。RADIUS 端口、共享密钥、Kafka topic 属部署环境配置，继续由环境变量/Secret 管理；页面不回显密钥。若后续要求热配置，另立契约和 adopted note。

## 实施顺序

1. 页面路由、菜单、角色可见性和 i18n 类型闭环。
2. AAA 总览页面，复用真实 admin API。
3. LOID 状态筛选和详情抽屉。
4. CDR/认证日志筛选与详情抽屉。
5. 页面单测、类型检查、生产构建。
6. 102 真实登录后通过页面/API 冒烟，核对网络请求、错误和真实数据。
7. 独立提交、推送、反向同步 main、门禁、ff-only 合并和 worktree 清理。

## 明确不在本期

- NAS/RADIUS 客户端 CRUD：当前没有持久化实体、密钥轮换契约和真实设备源。
- 在线会话/强制下线：当前没有 session 表或 NAS Disconnect 链路。
- 伪造 RADIUS 运行指标：应接入 Prometheus 真实指标后再做。
- 将环境 Secret 写入 `biz_params` 或前端。
