# PP2 前端页面体验打磨 · 路线图(2026-09-07)

> 上游任务:前端页面体验打磨(使用者视角:认知负担最低、步数最少、常用触手可及)。
> 前序:PP1(2026-09-05,三波)已完成代码卫生红线(alert/select/内联 style/300 行),本轮聚焦体验本体:反馈链路、表单、表格、选择器。
> 对接环境:http://192.168.0.102:28080(admin 前缀 /api/admin/v1,冒烟 admin/admin123);admin 前端 102 部署版 http://192.168.0.102:5180。

## 一、开工前审计(2026-09-07 实测 main@dd9acd88)

| 维度 | 事实 | 结论 |
|---|---|---|
| 红线 | 全域 native select=0、alert=0、无 >300 行文件 | PP1 卫生成果保持,本轮不回退 |
| 组件采纳 | boss 裸 button 79/裸 table 20;ams 77/15;org 70/14;oss 29/10;billing 25/10 | 体验债集中在 boss/ams/org |
| 反馈覆盖 | toast 使用:boss 4/32 文件、org 0/25、oss 0/11、billing 0/11、provision/quad/alarm 0 | 反馈链路普遍缺失 |
| 遗留选择器 | ResourcePicker 直用 16 页(alarm/ams.replace/billing×3/boss×7/oss×2/provision/quad) | W0 做兼容层,页面后续零改动受益 |
| PP1-C 登记债 | quad/check 与 alarm 内联 style+裸色值;provision 成功反馈未走 toast | 归 W3 清偿 |
| 表格基础 | DataTable 采纳 bss 10 页领先,boss 仅 3、ams/org/billing 几乎为 0 | 表格可读性是本轮主战场 |

## 二、波次划分(地基先行,域间不相交)

| 波次 | 范围 | 专属会话 | 分支 | 节奏 |
|---|---|---|---|---|
| W0 选择器基座 | components/Dropdown+MultiSelect+pickers 全家+ResourcePicker 兼容层+picker-guide | PP2-W0 | feat/pp2-w0-pickers | 即刻全量开工 |
| W1 boss 域(32 页) | order/dispatch/install-board/worker 系/message/dismantle/complaint/feedback/site/knowledge/license/callback/release/service-metrics 等 | PP2-W1 | feat/pp2-w1-boss | Phase A 只读走查→等 W0 合并→Phase B 修复 |
| W2 ams+org(58 页) | ams:asset/inventory/purchase/replace/stock/tag;org:apikey/company/datascope/department/menuperm/openplat/partner/post/region/staff | PP2-W2 | feat/pp2-w2-ams-org | 同 W1 两阶段 |
| W3 计费+运维(30 页) | billing 全域+oss 全域+provision+quad+alarm+intel 未覆盖残留+aaa-dashboard/aaalog;顺带清偿 PP1-C 登记债 | PP2-W3 | feat/pp2-w3-ops | 同 W1 两阶段 |
| W4 base+辅助页(46 页) | base 全域+profile/backup/dashboard/news/placeholder | 排队,有会话空出再派 | feat/pp2-w4-base-aux | W1-W3 收尾后 |
| 收尾整合 | 主树 make check+web-admin-check;102 部署冒烟;会话归档 | 负责人本人 | - | 全部合并后 |

## 三、统一体验验收规则(逐页走查,逐条对照)

1. 高频操作一步可达:新建/编辑/审批/反查等入口在列表行内或页头直达,无多余跳转。
2. 按钮完整反馈链:点击有响应、处理中有等待态(禁用+加载标识)、成功有 toast.success、失败有 toast.error 且原因可复制(错误文案支持复制按钮或选中复制)。
3. 表单:字段有 label+placeholder+必要辅助说明;按语义分组;可枚举/可关联输入一律选择器(SimplePicker/DialogPicker/域选择器),关联对象可搜索选定,禁自由文本臆造;必填/错误就地提示。
4. 表格:行操作完整(与 API 能力对齐,不做按钮 API 不存在的假入口);关联信息显示人类可读名称(取不到降级 #ID 并留痕),不裸露内部编号;删除必须 ConfirmDialog(danger) 且先查关联引用;检索/分页/统计(总数)齐备。
5. 全局选择器:全站表现一致(W0 统一契约后,各页不得绕过)。
6. 每页走查必须在 102 真实数据上以 cdp-admin-capture.mjs 实截+DOM 断言+console/网络零报错验证;无验证动作不得声称完成。

## 四、硬性纪律(全波次通用)

- 对接只用 102(28080);本地 vite dev 仅作新代码走查载体,proxy 指 102,结束必须关停;端口冲突换端口。
- 禁止:新增业务功能、改后端行为与数据结构、整体视觉重设计、炫技交互、新依赖、原生 select、裸 alert、playwright(仅 e2e/ 目录豁免)。
- worktree 开发,禁改主分支;branch 名按上表;commit 纪律 type(scope): subject,feat 带测试;i18n 三语键 append-only 压独立小提交;单文件 ≤300 行。
- 主树现存外来 worktree boss-odn-p9 属他人,严禁触碰。
- 完成后 push gitea 分支即报告,合并由负责人按 W0→W1→W2→W3 串行 ff-only 统一执行(防 i18n types 撞车);worktree 清理等合并确认后按协议四步走。

## 五、证据与账本

- Phase A 产物:docs/plan/pp2-audit-<域>.md(逐页走查表:页面/典型操作步数/反馈链缺口/表单问题/表格问题/选择器使用/修复清单+优先级),截图留 /tmp 不入库。
- Phase B 产物:docs/acceptance/2026-09-07-pp2-<波次>.md(机械门禁输出+逐页处置结论+走查断言记录),对齐 PP1 先例。
- 账本 .devloop/loop-state.json:U0-U6,每任务带可执行验收命令;U5 为 T14 ODN 前端强化 rider(归 W3,余力则做)。
