# 资产标签 P4 波计划(2026-09-06): 身份规范化与字典归一

前情: P3 波四任务已全验收合并(分页/身份列+EPC/报废三要素/事件回填), main=b8c9bf76。本波针对 P3 复盘遗留与新查明的数据现状。

## 一、数据现状盘点(102 实查, 裁定依据)

- assets.type 方言: ONU 172 / 光猫 49 / MI-ONU 5 / SMOKE 1。经查 MI-ONU 全部为 A-RK-E2E 造数残留、SMOKE 为当日冒烟残留, 均非真实业务类型。
- tags.epc_code 223/225 不符合 24-hex, 样本 E2E-SCAN-* 为历次 e2e 造数标签。物理 EPC 须与实物一致, 严禁程序生成重写。
- 其他域规模极小(provlog 7 / replacements 14 / stocktake-items 17 / batches 206): 无界增长场景不存在。

## 二、Lead 裁定(四条)

1. 类型归一: 权威码 ONU;存量 49 行「光猫」UPDATE 归一;type 写入走白名单(禁自由文本新方言);MI-ONU/SMOKE 按 e2e 残留清理, 不入类型体系。
2. MAC 规范化: 应用层写入统一为大写冒号形(AA:BB:CC:DD:EE:FF);唯一索引改规范化表达式(upper(regexp_replace(mac,'[:. -]','','g'))), 双保险防直写 SQL 绕过;存量 mac 现为 0 行, 迁移空转无负担。
3. 分页推广: 不做(其他域无界场景不存在, 做了即过度设计);十万级再议 keyset。
4. 非法 EPC 处置: 不程序造假重写;巡检暴露计数+清单, 交运营贴标真值回填或确认废弃。

研究出处: Postgres FM caSe-inSENsiTive(表达式唯一索引 vs citext)、eui48(EUI-48 canonical 表示)、FactVerse MDM(权威码+别名治理)。

## 三、任务清单(单一会话粒度, 机械验收)

| # | 任务 | 会话 | 迁移号 | 验收命令(102 部署后, exit 0) |
|---|---|---|---|---|
| T1 | MAC 规范化存储+表达式唯一 | P4-A | 000190 | bash scripts/e2e/verify-asset-mac-normalize-e2e.sh |
| T2 | 类型归一 ONU+白名单校验 | P4-B | 000191 | bash scripts/e2e/verify-asset-type-canonical-e2e.sh |
| T3 | 巡检两查+造数残留清理 | P4-B | 无 | bash scripts/ops/verify-patrol-extended.sh |
| T4 | 102 每日 e2e 冒烟 cron+runner 凭据卷持久化 | P4-C | 无 | bash scripts/ops/verify-smoke-cron.sh |

全部另需: make check 全绿; e2e 幂等+造数自清; BASE_URL 缺省 102; 契约变更带 fields.md 同步。

## 四、需求与边界

### T1 MAC 规范化(P4-A, 000190)
- 应用层: mac 写入(创建/编辑)归一为大写冒号形;格式校验沿用(六组 hex, : 或 - 分隔均可输入)。
- 迁移 000190: 存量 mac 归一 UPDATE(现为 0 行, 空转但必须存在保持向前一致);删除原 uq_assets_mac 部分唯一索引, 重建为表达式唯一索引 upper(regexp_replace(mac,'[:. -]','','g')) WHERE 归一后非空;down 完整回滚。
- e2e 断言: 小写横杠输入入库为大写冒号形;同 MAC 不同格式(: 分隔 vs 裸 hex)第二次创建 409;非法格式仍 400;造数清理。

### T2 类型归一(P4-B, 000191)
- 裁定落地: 迁移 UPDATE assets SET type='ONU' WHERE type='光猫'(49 行);down 不恢复(裁定不可逆, down 注释说明)。
- 写入校验: type 白名单(以 asset 域现行合法集合为准, 会话自行调研现有代码取值集合后落常量), 白名单外 400。
- 前端类型筛选/表单选项与 fields.md 三列对齐同步;中央登记文件独立小提交。
- e2e 断言: 创建 type=光猫 被拒 400;type=ONU 创建成功;102 直查 type 无「光猫」。

### T3 巡检两查+残留清理(P4-B)
- 巡检加两查(模式随现有 patrol 脚本): ①非法 24-hex 的 epc 标签计数(输出清单前 20, 口径=贴标待回填/待清理);②非白名单 type 计数(防新方言)。
- 造数残留清理: SMOKE-P3-* 资产 1 行、A-RK-E2E-001-MI-ONU-* 资产 5 行, 确认无标签绑定/领用/生命周期后删除;有引用则只暴露不删(脚本输出留痕)。
- verify-patrol-extended.sh: 两查输出格式可 grep、残留资产数为 0(或暴露清单逻辑生效)、幂等。

### T4 ops 冒烟 cron+凭据卷(P4-C)
- 102 每日 cron: 四个资产域 e2e 脚本串行(acceptance-lock 天然串行)+日志, 口径随 docs/ops/patrol-cron.md 登记新 cron 行。
- runner 凭据 compose 卷持久化脚本化(收殓 P2 遗留 README 项): 修改 102 上 gitea-runner compose 增加对 docker config.json 的卷挂载并重建验证。
- verify-smoke-cron.sh: crontab 含 smoke 行;compose 配置含卷挂载;--check 双过。本任务在 102 宿主直接操作, 交付=脚本入库+102 落地验证记录。

## 五、协作与合并协议

同 P3: 三会话 worktree(feat/p4-a / feat/p4-b / feat/p4-c)开发;迁移号独占(A=000190, B=000191, C 无);禁 push/合并/部署;完成回报结构化结果;Lead 逐个反向同步+make check+ff-only 合并+push;部署后 Lead 跑全部验收;全绿后清理 worktree/分支;归档由用户 GUI 手工操作(archiveSession 工具已列禁用)。

## 六、销项与遗留

- 销项: MAC 规范化、光猫/ONU 归一、分页推广(裁定不做)、runner 凭据卷、非法 EPC 处置口径。
- 遗留: keyset 分页(十万级再议);真实用户验收 13 表(用户侧);贴标真值回填(运营侧, 依赖巡检清单)。