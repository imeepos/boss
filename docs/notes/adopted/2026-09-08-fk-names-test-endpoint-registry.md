# 复盘整改批次：FK 人读名服务端化 /test 动作端点 /registry 收编

日期：2026-09-08

## 决策

体验打磨收官复盘的待办与缺陷，整改方案经社区实践检索后裁定：

1. **FK 人读名一律服务端 JOIN 进响应**：customers 列表/详情行补 `legal_entity_name`，worker 行补 `group_name`/`region_name`，complaints 列表补 `customer_name`。管理面板社区标准（AdminJS/Django admin 系）即服务端解析外键显示名，客户端批量装配是 N+1 反模式；否决 `?expand=` 参数化（单一消费方无需泛化）与前端批量查询端点。
2. **`POST /storage-config/test` 零副作用动作端点**：Stripe 式 validate 模式——用现存凭据对 MinIO 做最小真实探测（HEAD bucket），短超时，返回结构化诊断 `{ok, step, message}`（非布尔），任何路径不落密钥日志。对比 auth/sms/push/realid/stripe 五组配置页既有 `*/test` 惯例对齐。
3. **StatusTag registry 收编**：页内三语状态映射（diffKind 等 9 处建议项）迁入 registry 单一事实源；UserPicker 全站零消费判定删除（YAGNI，git 历史可恢复），否决"保留备用"。
4. **JSON 输入只做校验反馈**：importer/params 粘贴 JSON 在 blur 时解析并给行:列定位错误（ErrorBanner），不引 CodeMirror/Monaco 类高亮依赖——管理台场景校验+精确定位优于高亮（克制裁定，高亮在有真实痛点后再议）。
5. **未验证动作链补真**：devseed 扩 PENDING 工单/师傅主档/催收任务，102 端到端实走派单指派/师傅调队/催收执行，acc_ 前缀当页清理；交付人工签收走查指南 docs/admin/acceptance-walkthrough.md。

## why

- 复盘（本会话前文）确认：核心页面体验已收官，剩余欠账集中在「关联信息人可读」的最后一段（后端不回名称）与验证空洞（数据缺失导致未真提交）。社区实践为本项目既有直觉提供了权威背书，方案可一次做对。
- 执行纪律沿用：worktree 隔离、每项一提交一验证、门禁全绿才合并、合并成功后清理分支（用户明令免确认）。

## 放弃了什么（被否决项）

- `?expand=` 通用展开参数：只有两三个消费方，泛化是过度设计。
- 客户端 `/legal-entities` 全量查找兜底的现状保留：整表扫描兜底是债务本体，字段落地后删除。
- CodeMirror/Monaco 高亮：重依赖，收益未证实。
- workers 补删除接口：属新功能非债务清偿，不在本批。

## 关联

- 检索依据：AdminJS/Piccolo admin FK 显示值讨论（服务端 related query 即标准）、Stripe 密钥校验（零副作用真实探测）、设计系统 status token 单一事实源模式。
- 上游：docs/contract/data-relations.md §6（本批落地后逐条销账）、docs/admin/ux-final-audit.md 迭代建议节。
