# 资产与标签 P2 波次计划(2026-09-06,多会话协作)

> 上波 P1 已上线(3ce4150):联动/回收/字典/巡检四件套 + 000185-187 迁移生效。
> 本波聚焦四件:CI 部署守护、漂移处置(已消解)、端到端实测、事件消费面。
> 派发模式:主会话(project lead)创建 3 个新专属会话并行开发,需求+验收规则派发
> (不含源码),各自 worktree 短命分支,合并走 fetch+ff-only 协议,完成即归档。

## T1 CI 部署守护固化(会话 B)
背景:deploy-runner job 镜像被 prune 后 6 次 push 零部署,静默停摆(详见 ISSUE.md 2026-09-06)。
需求:workflow 首步自检 job 镜像存在性、缺失就地重建(构建上下文来源必须可持久化可重复,
二进制/凭据不得只依赖会消失的 /tmp 与未守护的宿主文件);run 拉起后无成功部署的超时告警
(接既有 Alertmanager 或最低限度写盘+巡检比对);不动 runtime 代码。
验收:①守护自检幂等可重复(删镜像→自检→恢复);②workflow 语法过、push 到分支不炸;
③真 push 端到端部署成功(healthz 新指纹);④make check 全绿。

## T2 漂移处置(主会话,已消解)
000185 随 P1 部署自动补账:巡检在案 5 条 LINKED 漂移全部 DEPLOYED(轨迹 +5),
SCRAPPED-未回收=0。无需额外处置,仅在波次 note 记录闭环。

## T3 装机联动端到端实测(会话 A)
需求:按既有真库验收脚本惯例,新增「新装 12 环节 + 资产联动断言」可重复脚本:
造数前缀隔离、收尾清理、失败留可 grep 信号;断言集=扫码 MATCH 后资产 DEPLOYED+
地址绑定+轨迹 DEPLOYED 行+scan_logs MATCH;目标 102 真实部署,禁 mock。
验收:①脚本 102 跑通 exit 0,逐环节输出断言结果;②故意注入一次断言失败能红;
③脚本入 repo 惯例位置并提交。

## T4 标签事件消费面(会话 C)
需求:查询端点(按标签/资产倒序+limit 上限,统一信封与错误映射);前端标签页
解绑/报废入口(二次确认:影响面+不可逆提示+原因必填)+事件时间轴抽屉(仿资产轨迹);
openapi/生成器/fields.md/i18n 同步。明确不做:全局事件大屏、游标分页、跨域聚合。
验收:①go build/test+前端 build/test+三生成器 --check 全过;②102 冒烟:解绑→
事件+1 且状态 UNBOUND,报废→RECYCLE 事件,查询倒序正确;③fields.md 契约同步。

## 协作与合并纪律
- 各会话独立 worktree 短命分支;合并前 fetch,main 已前进则分支侧 merge main 重跑门禁;
- 每任务=独立可 revert 提交;契约变更 fields.md 随提交同步;
- 主会话负责按序合并(T1→T4→T3 部署后终验)、冲突消化协调、102 终验、会话归档。

## 检索结论回填与协作冲突处置

### R-A runner 镜像守护(已补发会话 B)
事故机理坐实:job 镜像本地存在时零 registry 交互;被 prune 后拉取读 act_runner 进程用户的
~/.docker/config.json(宿主 docker login 不自动传递),401 秒取消且默认日志级别无错误行。
社区最稳组合=「本地标签+定时保活」:label 改指无前缀本地镜像,cron 定期 pull+tag 保活,
runner 用户写 auths 兜底;prune 用 --filter label!=ci-keep 防误伤。告警走 dead-man-switch
(部署成功写 last_success,超 26h 告警),不对错误计数。分工:仓库内文件由会话 B 提交;
runner label 与宿主 cron 属宿主操作,由会话 B 产出脚本+操作清单,Lead 在 102 执行。

### R-C 事件消费面(已补发会话 C,后因 P2-W1 冲突转待命)
端点:按聚合 id 倒序+limit(上限 100)即够,append-only 且 id 单调,首版不需要游标;
action 白名单多值过滤。UI:一行一事件(时间/操作人/徽标/对象),changed 只渲染变化键,
不 dump 全量 JSON;危险确认三要素=影响面+不可逆说明+红色确认键默认禁用且原因必填
(报废要求输入资产编码)。明确不做:全局大屏/全文搜索/SSE 推送/导出。

### 协作冲突处置(2026-09-06 09:55)
发现并行波次 P2-W1(资产台账 CRUD 补全:后端 CRUD+前端操作列含报废/删除+verify-asset-crud-e2e)
正在同一批文件施工(loop-state.json 已由该波次接管,状态 doing)。处置:本波会话 C(事件消费面)
转待命,待 P2-W1 合入 main 后以增量方式重派(复用其操作列/表单成果,只做 events 查询端点+
事件抽屉);会话 A(端到端联动实测)与 B(CI 守护)文件面无冲突,继续并行。
合并顺序:B、A 先行合入;P2-W1 合入后再重派 C。

### R-A3 端到端实测落地(会话 A,2026-09-06)
脚本:scripts/e2e/verify-asset-linkage-e2e.sh(+伴生 ...-residue.sql;首参 BASE_URL,默认 102),102 真实部署
可重复执行(acceptance-lock 防并行)。断言集:L0 基线/S1-S12 环节(order.stage、
order_stages、四码、认证账号)/L1-L4 装机联动(资产 DEPLOYED+绑地址、asset_lifecycles
DEPLOYED 行、scan_logs MATCH、quad_links LINKED)/D1-D3 拆机联动(IN_STOCK+清地址、
IN_STOCK 轨迹行、UNLINKED;不可达输出 SKIP+原因)。收尾 acceptance-cleanup --apply+
残留 SQL 逐类为零+db-patrol-gate。102 实测:连续两轮 exit 0(40 断言/轮,约 46s/轮),
E2E_INJECT_FAIL=1 注入变红可定位。环境发现(移交处置):1)runtime bindTagEvent 以
[]byte 写 json 列(tag_events 22P02),CreateTag/Asset 绑定路径 500,标签资产预绑定暂走
夹具 SQL;2)TL1 端点 env 172.26.0.1:13027 拒连(oltsim 在 2323/23333),09-03 后零下发
SUCCESS,provisioner 完成度降为 S7b WARN 留痕;3)addresses.label 拒绝连字符,造数后缀
必须纯数字。