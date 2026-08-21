# JPush 推送后台配置项设计(push-config)

> 状态:设计定稿,待实现。日期:2026-08-23。
> 前置调研:`docs/plan/push-integration.md`(渠道选型裁定:极光 JPush 聚合)。
> 同构依据:完全复用 smsconfig/realidconfig 已定稿模式——`biz_params` 存储 + secretbox AES-GCM 加密 + 掩码回显 + `Dynamic` 60s 热更 + env 凭据兜底(见 adopted 2026-08-21-auth-config-secret-storage)。

## 1. 范围

本设计只覆盖**后台配置项**(admin 配置页 + 存储 + API + 运行时装配)。设备注册表、Kafka 消费、推送留痕是后续独立变更,不混装。

## 2. 配置项定义(biz_params,前缀 `push.*`)

| key | group | secret | 默认 | 说明 |
|:----|:------|:------:|:-----|:-----|
| `push.enabled` | channel | | `true` | 总开关;false 时发送链路直接降级(不外呼) |
| `push.provider` | channel | | `jpush` | 预留多供应商路由(对齐 sms.provider 形态) |
| `push.jpush.appKey` | channel | | | JPush 应用 AppKey(控制台"应用设置"页) |
| `push.jpush.masterSecret` | channel | ✓ | | 服务端密钥,**加密落库,GET 只回 hasValue** |
| `push.jpush.apiUrl` | channel | | `https://bjapi.push.jiguang.cn/v3` | REST 端点,默认官方;留空用默认 |
| `push.jpush.androidChannel` | channel | | `production` | JPush Android 通道环境:`production`/`develop`,联调期用 develop |
| `push.jpush.liveTime` | channel | | `86400` | 离线保留秒数(默认 1 天) |

group 仅一个 `channel`(无文案模板组;通知文案由业务事件携带)。

自检必填(草稿合并后校验):`push.jpush.appKey` + `push.jpush.masterSecret`。

## 3. Admin API(internal/httpapi/admin/pushconfig.go)

与 smsconfig 路由逐一同构:

- `GET /api/admin/v1/push-config` → `{fields:{key:{value,hasValue}}}`,secret 只回掩码标记。
- `PUT /api/admin/v1/push-config/channel` → `{values:{key:val}}`;secret 空串=不修改;越组 key 422。逐字段落库 + 审计(`push_config`/key 粒度,detail 只记 secretUpdated+length)。
- `POST /api/admin/v1/push-config/channel/test` → 无 `target` = 配置完整性校验(ok/missing 列表);带 `target`(registrationID 或 alias) = 用合并后配置真实调 JPush API 发一条测试通知,回 `{ok,latencyMs,message}`。

权限:`menu:pushconfig`,迁移 `000094_pushconfig_menu`(up/down 成对,授 sysadmin,模板同 000064)。

## 4. 运行时装配(internal/pkg/push + wiring_push.go)

```
internal/pkg/push/
  push.go        // Sender 接口: Send(ctx, PushRequest) (MessageID, error)
  jpush.go       // JPushSender: 直调 REST v3(POST /push,BasicAuth appKey:masterSecret)
  dynamic.go     // Dynamic: resolve 回调懒加载配置,60s 缓存热生效(同 sms.Dynamic)
```

- **选型注记:直调 REST,不引官方 Go SDK**(`jpush/jpush-api-go-client`)。对齐短信通道"零 SDK 依赖直调"裁定:JPush REST v3 仅一个 POST + BasicAuth,SDK 引入依赖树收益为零。若后续需要厂商通道回执解析再评估。
- `wiring_push.go`:`pushConfigResolver` 读 biz_params → 明文 `ChannelConfig`;读库失败回退 env `BOSS_JPUSH_APP_KEY`/`BOSS_JPUSH_MASTER_SECRET`(不阻塞装配)。
- 未配置(enabled 或凭据缺失)时装配 `LogSender`(日志打印推送载荷,仅开发联调)——同 sms LogSender 降级约定。

## 5. Admin 前端(web/admin)

- 页面 `src/pages/base/pushconfig/`(index.tsx + logic.ts,契约注释与后端 pushconfig_fields.go 对齐)。
- 路由 `/base/pushconfig`,menu.def key `pushconfig`,`基础配置·推送配置`。
- 三语言 i18n(zh-CN/en-US/ms-MY)同步补 `pages.pushconfig` 与菜单词条。
- 交互:通道卡片(appKey/masterSecret/apiUrl/androidChannel/liveTime)+ 保存 + 自检按钮(支持填 registrationID 实测一条)。secret 输入框掩码回显"已配置,留空不修改"。

## 6. 契约同步义务(实现随主变更同提交)

- `docs/contract/fields.md` 补"推送配置"节(key ↔ 列名 ↔ 枚举三列对齐)。
- `docs/contract/domain-map.md`:notify 域补 push 配置 admin 投影行。

## 7. 放弃了什么

- 独立 push_config 表:配置 <10 行、无关联查询,biz_params 热更链路现成(同 auth 裁定)。
- 官方 Go SDK:零收益依赖树,见 §4 选型注记。
- 厂商通道参数(华为/小米 AppKey 等)入库:厂商参数配在 **JPush 控制台侧**,不经 BOSS 存储;BOSS 只持 JPush 聚合凭据,这正是选聚合商的架构收益。

## 8. 分期

| 期 | 内容 |
|:---|:-----|
| 本变更 | 配置项全套(本设计 §2-§6) |
| 下一步 | App 端 JPush SDK 集成 + RegistrationID 上报接口(`push_devices` 表) |
| 再下一步 | Kafka 事件 → push consumer 定向推送 + push_records 留痕 + 派单短信兜底 |
