# odn-v2 合并前交叠预警简报(picker 改造轮)

日期:2026-09-08 · 发起:选择器改造负责人会话 · 供:feat/oss-odn-v2-ui 会话

## 背景

2026-09-08 picker 改造轮(odn-pickers / 资源GIS附件 / 采购计费发布 三线)已合入 main(A 线基线 49762c69,后续 main 新进提交均为 skill 文档,不涉前端代码)。odn-pickers 会话收尾时核实:feat/oss-odn-v2-ui(本地 39c15439,远端无此分支)未进 main,且与 odn-pickers 已合入改动存在文件交叠。

## 交叠文件(5 个,合并时大概率文本冲突)

1. web/admin/src/pages/oss/odn/AssetsPanel.tsx —— 对方改动:设备/资产/施工项目三处改为 picker(约 +40/-4)
2. web/admin/src/pages/oss/odn/permits.tsx —— 对方改动:许可单关联设施、列表项目 ID 筛选两处改为 picker(约 +34)
3. web/admin/src/pages/oss/odn/ConstructionDetail.tsx —— 对方已重构拆分出 MaterialIssuesCard.tsx(新增文件,若 odn-v2 也改本文件需按 main 现状对齐新结构)
4. web/admin/src/i18n/locales/zh-CN.ts / en-US.ts / ms-MY.ts —— 对方 append-only 新增 pages.odn 域 pick*/projStatus/issue* 共 16 键
5. web/admin/src/i18n/types.ts —— 对方已同步上述键的 Translations 类型

## 消解口径(在 odn-v2 feature 侧执行)

- 按工作树合并协议先 merge main 反向同步,以 main 现状(含 picker 改造)为基消解冲突,不得回退 picker 改造。
- i18n 同位置 append 冲突:保留双方键集(两边的键都留),types.ts 必须补齐 odn-v2 新增键,否则 keys.test 与 TS 编译双红。
- 消解后门禁必须复跑:cd web/admin && pnpm typecheck && pnpm test && pnpm build。

## 参考基线

- odn-pickers 线最终门禁绿基线:main@49762c69(typecheck 0 错 / vitest 534 passed / build 通过)。
- 质疑或需要联动时,可向工作区内发起 picker 改造的协调会话投递消息。