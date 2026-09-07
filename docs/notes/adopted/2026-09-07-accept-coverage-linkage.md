# 竣工验收联动覆盖新裁定(ACCEPTED→覆盖 SERVED)

- 日期:2026-09-07
- 场景:P-INFRA-1 W4 捎带审查 F6:项目 ACCEPTED 与地址覆盖 PENDING→SERVED 无联动,覆盖门控(BOSS_ODN_COVERAGE_GATE)开闸后出现「已竣工仍拒单」口径冲突。
- 需求源:docs/reviews/2026-09-07-invest-build-phase-design-review.md F6(新裁定,随 W4 落)。

## 决策

1. 项目竣工验收(BUILDING→ACCEPTED)与覆盖联动同事务:单内设施 as-built 回填 IN_SERVICE 的同时,
   address_coverage 中 facility_code 命中单内明细且 status=PENDING 的行翻转为 SERVED。
2. 联动数(coverageServed)随竣工响应与审计(odn.construction.accept 明细 coverageServed 字段)透出,
   失败随事务回滚并留 [odn-coverage] ACCEPT LINK FAILED 可 grep 日志,成功留 ACCEPT LINK proj=x served=n 日志。
3. 开关 BOSS_ODN_ACCEPT_COVERAGE_LINK=off 显式关闭;默认开(新裁定即业务正确口径,存量项目行为不变——
   只有「走竣工动作」才触发联动,历史 ACCEPTED 项目不回填)。

## why

- 同事务而非事后任务:覆盖状态与设施生命周期同源(as-built 同刻成立),拆开会有「设施已在网、覆盖仍 PENDING」的脏窗口,
  门控开闸时该窗口直接拒单,正是 F6 指出的口径冲突。
- 只翻 PENDING 不碰 UNSERVED/SERVED:PENDING=规划在建(有目标挂接),竣工即兑现;UNSERVED 无目标挂接,翻 SERVED 会违反
  address_coverage_target_chk(SERVED 须挂目标),交由覆盖页人工/导入链维护。

## 放弃了什么(被否决项)

- 事后异步任务补偿:引入脏窗口与重试语义,且需新调度面,放弃。
- 翻转 UNSERVED:违反覆盖目标约束,语义错误。
- 竣工时自动建覆盖行:无地址映射依据(项目不直挂地址),目标缺失的覆盖行不可造,放弃。

## 关联

- 实现 internal/domain/odn/pg_construction_flow.go(linkCoverageOnAccept/AcceptProject)、internal/app/wiring.go 开关;
  契约 fields.md 1.5.13 头注;审计 action=odn.construction.accept。
- F3 许可工作流另见 adopted 2026-09-07-odn-permit-workflow。