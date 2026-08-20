# 认证配置页实现规格（auth-config-v1.spec.md）

> 配对设计稿：`designs/auth-config-v1.png`。本文可直接转发给前端工程师或 AI 编码代理。
> 页面归属：后台 `web/admin`，基础设置分组下的新页面 `pages/base/authconfig/`，菜单项「认证配置」。

## 1. 页面结构（与设计稿区块一一对应）

### 1.1 外壳（复用现有，不新做）
- 顶栏：深 navy `--shell-topbar-bg #0F1E3B`，左 logo + 产品名、搜索框、右侧通知 + 用户头像。
- 侧栏：`--shell-side-bg #FCFCFD`，「基础设置」分组展开，子项「认证配置」激活态：
  背景 `--shell-menu-active-bg #FBF5E8`，文字 `--shell-menu-active-text #172744`。
- 内容区：背景 `--shell-content-bg #F4F5F7`，内边距 24px。

### 1.2 页头
- 用 `components/business/page-head` 的 `PageHead`：
  title「号码认证配置」，desc「管理本机号码一键登录与短信验证渠道，失败自动降级」。

### 1.3 卡片一：中国区 · 一键登录（极光认证 JVerification）
- `Card`（`--shell-card-bg` 白底、`radius-md 8px`、`--shell-card-shadow`）。
- 标题行：左侧标题「中国区 · 一键登录（极光认证 JVerification）」；
  右侧状态徽标 + 总开关（`ui/switch`）：
  - 开关 ON → `Badge` variant success 文案「已启用」；OFF → 灰色「已停用」。
- 表单（label 灰色 `--color-text-secondary` 13px，输入框 `ui/input`）：

  | 字段 key | 标签 | 控件 | 说明 |
  |---|---|---|---|
  | `auth.cn.appKey` | AppKey | Input | 极光控制台应用 AppKey |
  | `auth.cn.appSecret` | AppSecret | Input(type=password) + 眼睛切换 | **脱敏回显**：接口只返回掩码，用户未修改则提交时不带此字段 |
  | `auth.cn.packageName` | Android 包名 | Input | 极光后台登记的包名，如 `com.ymm.boss.user` |
  | `auth.cn.preloadTimeoutMs` | 预取号超时（毫秒） | Input(number) | 默认 5000，范围 2000–10000 |

- 底部按钮（右对齐，间距 8px）：
  - `ToolbarButton`「测试连通性」：用当前配置调后端自检接口，结果用 sonner toast。
  - `ToolbarButton primary`「保存」：`--primary #273F70`，loading 态禁用。

### 1.4 卡片二：马来西亚/海外 · 号码认证
- 标题右侧：OFF 时徽标「待配置」（灰色），ON 时「已启用」（绿）+ Switch。
- 表单：

  | 字段 key | 标签 | 控件 | 说明 |
  |---|---|---|---|
  | `auth.my.provider` | 认证渠道 | **Dropdown 组件**（禁止原生 select） | 选项：`opengateway` = GSMA Open Gateway（IPification）、`none` = 不启用（仅短信） |
  | `auth.my.smsProvider` | 短信兜底渠道 | Dropdown | 选项：`engagelab`（极光海外 EngageLab）、`twilio`、`vonage` |
  | `auth.my.apiKey` | API Key | Input(password) + 眼睛 | 同 1.3 脱敏规则 |
  | `auth.my.countryCode` | 默认国家码 | Input | 默认 `+60` |
  | `auth.my.smsSign` | 短信签名 | Input | 如 `YMMBOSS` |

- 底部按钮同 1.3。

### 1.5 卡片三：降级与合规策略（通栏）
- 三个 `ui/switch` 横排（间距 32px）：
  - `auth.fallback.smsOnFail` 认证失败自动降级为短信验证码（默认 ON）
  - `auth.fallback.billingAlert` 仅认证成功时计费告警（默认 ON）
  - `auth.fallback.autoRegister` 未注册号码自动创建账号（默认 OFF）
- 两个输入：
  - `auth.compliance.privacyVersion` 隐私协议版本（如 `v2026.02`）
  - `auth.compliance.agreementUrl` 授权页协议链接（URL）
- 底部说明行：`--color-text-tertiary` 12px，ⓘ 图标 +「授权页须展示协议勾选框，SDK 初始化前须取得用户隐私协议同意（工信部合规要求）」。

## 2. 接口契约建议（后端 Go，前缀 `/api/admin/v1`）

- `GET /auth-config` → 返回全部配置，secret 字段只回掩码（如 `••••••••` + `hasValue: true`）。
- `PUT /auth-config/{group}`（group ∈ `cn` / `my` / `fallback`）→ 部分更新；secret 字段值为空表示不修改。
- `POST /auth-config/{group}/test` → 用草稿配置做连通性自检，返回 `{ok, latencyMs, error}`。
- 存储复用现有 params 机制（`GET/PUT /params`）或独立表，secret 必须加密存储，落库决策当天写 `docs/notes/adopted/`。
- 契约变更同步 `api/openapi/sys.yaml` 与 `docs/contract/fields.md`。

## 3. 注意事项（红线提醒）

1. **下拉一律用 `components/Dropdown.tsx`**，禁止原生 `<select>`（弹层无法随主题定制，已被点名 2 次）。
2. **i18n**：所有文案进 `i18n/locales/{zh-CN,en-US,ms-MY}.ts` 三语，key 挂 `pages.authconfig.*`。
3. **双主题**：所有颜色引用 tokens.css 变量，不得写死 hex；完成门禁后双主题截图验证。
4. **secret 脱敏**：页面任何状态（含接口报错、详情弹层）不得回显完整 AppSecret/API Key。
5. 门禁：`pnpm typecheck && pnpm test && pnpm build` 全绿 + git commit 收尾。
6. 权限：该页面挂「基础设置」分组，需管理员角色（参考 `pages/base/params` 的路由注册方式 `router/menu.def.ts`）。
7. 移动端 App 端联动：配置开启后，用户端登录页「本机号码一键登录」按钮才展示；预取号失败时隐藏按钮（不展示报错）。
