# 后台前端体验打磨项目：批次规划与组件采用政策

日期：2026-09-08

## 决策

用户下达全局体验打磨任务（认知负担最低、步数最少、反馈完整）。立项为多会话并行项目，由负责人会话统一派发/合并/归档：

- **Wave 0（最高优先）**：全局通用选择器族专项打磨（`web/admin/src/components/pickers/` + ResourcePicker）。选择器是一切表单/筛选的公共基础，先于页面批次落地；对外 API 只做加法保持向后兼容；页面侧问题只记审计清单不改页面。
- **Wave 1（与 Wave 0 并行，页面文件互斥）**：按业务流程最重的三域逐页打磨——bss（客户/用户/产品/营销）、boss（订单/师傅/派单/消息）、billing（缴费/日结/应付/发票）。
- **Wave 2**：org+base、ams+quad+provision、intel+oss、partner+profile。
- **Wave 3**：覆盖扫尾 + 全站一致性复核（选择器口径、删除链路、反馈链路抽查）。

每页验收基线（来自用户任务书，逐条人工走查留证）：
1. 高频操作一步可达；2. 按钮完整反馈链（点击响应/等待态/成功提示/失败轻提示且原因可复制）；3. 表单标签+占位+辅助说明、语义分组、枚举与关联输入一律选择器；4. 表格操作完整、关联显示人可读名称、删除二次确认+关联风险提示、检索/分页/统计一目了然；5. 全站选择器口径一致。

组件采用政策：**有现成组件一律复用**（pickers 族/Dropdown/MultiSelect/DatePicker/RegionCascadePicker/ConfirmDialog/StatusTag/Pagination/Drawer/Card/ui-table/business data-table/page-head/EmptyState/form-field/submit-button/AttachmentManager/sonner）。页面打磨时顺手消化本页裸写的表格壳/卡片壳类串与裸 table（前次盘点：85 文件裸 table、thead 巨串 64 文件、卡片壳串 55 文件）。禁止原生 select；i18n 三语闭环；主题令牌；300/60 行红线。

## why

- 组件拆分盘点已完成（本会话 2026-09-08 前文），迁移的最省路径是并入逐页打磨：同一文件只动一次，避免「先机械迁移再体验返工」的双重触碰与冲突。
- 选择器被用户定为最高优先（决定全系统表单质量），故单列 Wave 0 并约束其只动 components/pickers，与页面批次文件互斥，保证并行安全。
- 并行会话都对 i18n locale 文件有增量：要求优先复用既有 key、新增 key 集中一次提交，合并由负责人串行执行（一次只合一个，合完 worktree-sync 其余）。

## 放弃了什么（被否决项）

- 独立的「Card/表格机械替换专项批次」先行：会与体验批次触碰同一批文件，产生并行冲突面，弃；并入逐页打磨。
- 各会话自行合并 main：违反「一次只合一个」，合并权收归负责人会话串行执行，执行会话只 push 分支汇报待合并。
- 大拆大改视觉/交互框架：用户明确不做重设计与炫技交互。

## 关联

- 组件盘点数据来源：本会话 2026-09-08 对 web/admin/src 的 grep 统计（裸 table 85 文件、thead 巨串 64 文件、卡片壳串 55 文件、pickers 消费 20 页、ConfirmDialog 65 页、sonner/toast 82 页）。
- 流程依据：AGENTS.md worktree 合并协议（2026-08-22）、止损线（2026-08-28）。
