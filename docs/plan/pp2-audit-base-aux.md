# PP2-W4 base+辅助页 Phase A 审计(2026-09-07)

- 分支:feat/pp2-w4-base-aux(worktree /Users/imeepos/ext512/ymm-001/boss-pp2-w4,自 main@bef4e5ad 切出)
- 负责会话:PP2-W4 | 方法:102 部署版(http://192.168.0.102:5180)cdp-admin-capture 实截 + DOM 断言 + console/网络采集 + 源码逐组扫描 + 路由核对;Phase A 只读,未改动 102 数据
- 范围:base 全域 16 组 + profile/backup/dashboard/news/placeholder,共 21 组 22 条路由
- 边界:base 卫生红线 PP1-A 已清零,本审计不重复登记;聚焦反馈链/表单/表格/选择器四维

## 一、机械基线(全绿)

| 项 | 结果 |
|---|---|
| 原生 select / 裸 alert | 0 / 0(全域 grep) |
| 单文件行数 | 全部 ≤300(geo 843 已拆 6 文件最大 167;importer 目录 2093 为多文件合计) |
| 走查渲染 | 22/22 路由真实渲染(chars 297-2251),0 登录页误入 |
| console err/warn | 22 路由全 0 |
| network ≥400 | 22 路由全 0 |

## 二、逐页走查表(102 真实数据)

截图 /tmp/pp2-w4/pg*.png(22 张)+ 交互断言 w2-*.png;DOM 断言为 eval 返回值。

| 路由 | 渲染断言 | 四维备注 |
|---|---|---|
| /base/account | 表 10 行、分页文案、28 个行内操作钮 | DataTable ✓;行内启停操作的成功反馈待接 toast(P2-1) |
| /base/authconfig | 表单页 26 钮 | toast ✓(标杆六页之一) |
| /base/smsconfig | 表单 + 渠道测试入口 | FormField/SecretInput/已配置不回显/toast.success+error/渠道 test 全齐(标杆) |
| /base/pushconfig | 同上模式 | 标杆 |
| /base/realidconfig | 同上模式 | 标杆 |
| /base/stripeconfig | 表单 + 支付渠道测试 | 标杆 |
| /base/storageconfig | 表单 + SecretInput + rotate-secret | 标杆;防御性 isLogin=true 为密码型输入误报,已核实非登录页 |
| /base/servers | 本地 CRUD 表 | localStorage 本地态(页面自注 localOnly,无后端端点);useConfirm(danger) ✓;flash 为局部 span(P2-2);手搓编辑框(P3-3) |
| /base/apidocs | swagger 文档页 | 只读,无表单 |
| /base/license | 授权页渲染 | 组件实为 pages/boss/license(跨域挂载,登记事实,非缺陷) |
| /base/params | 表 10 行可编辑网格;行内改值 → 「已修改」徽标 → 保存钮启用(现场断言 ✓) | 见 P1-1/P2-3 |
| /base/address | 地址树 + 挂接抽屉(RegionCascadePicker) | 树加载失败无页面级错误条(P2-4);RegionCascade 自身 loading/空态/重试已合规 |
| /base/geo | 国家 + 区划两级面板 46 钮 | 拆分合规;CountryForm/SubdivForm 表单就地错误 ✓ |
| /base/audit | 表 10 行 + 1 输入 + 3 筛选 Dropdown | DataTable ✓ 筛选齐 |
| /base/crashlogs | 日志表渲染 | 无分页(P2-5);ErrorBanner/EmptyState ✓(PP1-A) |
| /base/importer | 导入中心(抽屉+面板) | PP1-A 已拆分;批量导入长任务过程反馈形态登记复核(P3-6) |
| /base/realname-review | 审核队列(102 现 1 条待审) | 拒绝理由必填校验 ✓、catch 置错 ✓;行内无按钮,审核入口在行展开(路径较深,P3-7);队列无分页(P2-5 同项) |
| /base/backup | DataTable 10 行 | 备份/恢复任务成功反馈待接 toast(P2-1 同批) |
| /profile | /ucenter/overview 外壳 | 子页(personal/password/apikey/audit/myData)源码有 loadFail/empty 文案;子页逐页走查留 Phase B 开工时补 |
| /dashboard | 总览表/图 42 钮 | 渲染 ✓ |
| /news | /news 仅 /news/:slug 详情路由,无列表路由(入口待核,P3-8) | 112 行只读 |
| /placeholder | 占位页 | 正常 |

## 三、四维发现清单(Phase B 候选,勿与 PP1-A 已修项重复)

| # | 优先级 | 维度 | 发现 | 建议 |
|---|---|---|---|---|
| P1-1 | 高 | 反馈链 | base/params 批量保存:失败信息渲染在 success 绿色 span(错误误用成功色);Promise.all 一败全报败,实际可能部分已保存,口径失真 | 失败走 toast.error + 逐项结果口径(成功 N/失败 M 可复制);文案色分离 |
| P2-1 | 中 | 反馈链 | account 行内启停、backup 任务创建等写操作成功反馈未走 toast(全域 toast 仅配置六页) | 对齐路线图规则 2:toast.success/toast.error,失败原因可复制 |
| P2-2 | 中 | 反馈链 | params/servers 用局部 span flash(2.5s)代替全局 toast | 统一接 sonner 体系 |
| P2-3 | 中 | 表单 | params 详情弹层为手搓 fixed 定位 div | 换 ui/dialog 或 Drawer(一致性) |
| P2-4 | 中 | 反馈链 | address 页国家列表加载失败仅 console.warn,页面级无错误提示 | 补 ErrorBanner + 重试(对齐 W0 契约 3) |
| P2-5 | 中 | 表格 | crashlogs、realname-review 列表无分页 | 接 Pagination(对齐路线图规则 4) |
| P3-3 | 低 | 表单 | servers 编辑框手搓 | 换 ui/dialog |
| P3-6 | 低 | 反馈链 | importer 批量导入长任务的过程反馈形态 | Phase B 复核是否需要进度态 |
| P3-7 | 低 | 表单 | realname-review 审核入口交互路径较深(行展开) | 评估行内直达 |
| P3-8 | 低 | 路由 | /news 无列表路由,详情页入口依赖外链 | 核对入口完整性 |

选择器维度:全域仅 params(静态三态 Dropdown)与 address(RegionCascadePicker)使用选择器,均继承 W0 契约;配置六页为纯表单无数值枚举下拉,无违规。文案当 value 的 W1 裁定防御在本域 102 走查期间 grep '[Dropdown] value 不在 options 值域' 0 命中。

## 四、路由与契约核对

- 页面 ↔ 端点:27 个去重端点(accounts/roles/departments/posts/legal-entities/regions/menu-perms/api-keys/audit-logs/params/auth-config/sms-config/push-config/realid-config/stripe-config/storage-config(含 channel/test)/backup(jobs/restore/tables)/crash-logs/dashboard/geo.countries/verifications)均有对应页面消费,未发现假入口;/base/servers 为 localStorage 本地态(页面自注 localOnly),/base/license 挂载 boss/license 组件(登记事实,非缺陷)。
- openapi 契约逐条对账留 Phase B 与后端路由核对细目(Phase A 以页面↔端点面为主)。

## 五、Phase B 建议顺序

1. P1-1 params 反馈色与部分失败口径(小,独立可验)。
2. P2-1 写操作 toast 全域对齐(account/backup 先行,再 params/servers flash 统一)。
3. P2-4/P2-5 address 错误条 + crashlogs/realname-review 分页。
4. P3 批次(手搓 modal/dialog 化、realname 入口、news 入口核对)。

等 Phase B 指令。
