# 存量开户记录导入工程计划（docs/design/2026-09-07-legacy-kaihu-import）

> 数据源：docs/开户记录_20240106.xlsx（347 条有效开户记录，2023-11-14 ~ 2024-01-09，147 行纯格式空行）
> 环境基线：102 部署库直查（boss-infra-postgres-1）+ admin API 复核，2026-09-07。
> 状态：执行中。负责人会话（goal-508a30db）统筹，T1/T2/T3 三会话并行，T4 合并后执行导入。

## 1. 数据画像（已核）

- 账号 347 全唯一：OWPAL\*×249、OWTAC\*×94、POWC_PALYYT_001（帕洛营业厅）、OWPAL19516_02（二次开户）、OWC_Office_GF、Office301（IPv6 测试）
- 带宽 20M×115 / 50M×140 / 100M×51 / 200M×36 / 30M×1 / 缺4；月数 1×272 / 3×47 / 7×18 / 15×3 / 0×3 / 缺4
- OLT003×248 / OLT001×88 / 缺11；PON口 缺16；SN 330 全唯一（缺17）；VLAN 四元组 缺35；OCC/OBD 缺16、Port 缺18、ONT号 缺34、型号 缺21
- 型号 GP8816A×233 / ZC-521×93；拆机 1 笔（OWPAL51531，2023-12-19）；尾批 8 条备注 2024/1/9（有 SN 无 VLAN，待配置形态）
- VLAN 规律：外层 VLAN 按 (OLT,PON板卡) 分组一致（OLT001≈110x、OLT003≈111x）；内层/internet/TR069 每线一对。存在 116 等离群值 → 校验按组众数偏离告警
- 业界参考（QinQ：SVLAN 按 PON 板卡/端口组规划、CVLAN 每线路、TR069 管理通道独立成对）与 web 检索结果一致：langzhichina.com GPON VLAN ISP 指南、Huawei HG8240 开局数据规划表、Kingbase 运营商数据库替换方案（staging/dry-run/对账）

## 2. 缺口清单

建模（字段层）：A1 VLAN 四元组全库无列；A2 OCC/ODB/OBD 层级无模型（ODN 域表在但空，且 Excel 三位编码不合 ODN 五位规范）；A3 ~~月数~~ orders 已有 buy_months/gift_months（复核后降级）
主数据：B1 OLT001/OLT003 未登记；B2 ports.pon_\* 仅 1 行有值（pon_onu_alloc 1 行）；B3 asset_models 无 ZC-521/GP8816A；B4 219 台 ONU 全部无 SN；B5 产品无 20M/50M 档；B6 ODN 层空
业务数据：C1 347 宽带账号（lo_accounts 仅 3 条测试）；C2 客户主档 0（Excel 无姓名/证件/电话）；C3 347 历史订单 + 1 拆机；C4 quad_links 待建
写路径：D 无 POST /resources、POST /ports、POST /lo-accounts（现有 POST /customers 注明批量导入用、POST /assets、POST /asset-models、ODN 全套写接口可用）

## 3. 裁定（详见 notes/adopted/2026-09-07-legacy-vlan-on-ports.md）

VLAN 挂端口侧（ports 加 svlan/cvlan/internet_cvlan/tr069_cvlan）；OCC/ODB/OBD/Port 以 legacy_path 单列无损承接（ODN 正式建模列 P2）；lo_accounts 加 contract_months 存档月数；不建 347 历史订单（C3 降为 P2，拆机以 lo_accounts.status=CLOSED 表达）；缺省归属 legal_entity=平台总公司、区域=空(0)、地址=单一占位节点(needs_review=true)。

## 4. 任务分解与验收规则（验收命令均为机械可执行，规格不含源码）

### T1 建模迁移（会话 A，迁移号 000202，开工前按协议重查号）
范围：ports 加 svlan/cvlan/internet_cvlan/tr069_cvlan SMALLINT 可空 + legacy_path TEXT 可空；lo_accounts 加 contract_months SMALLINT 可空；fields.md §4.2 同步三列对齐；domain 结构体补字段。
验收：① worktree 内 "go build ./..." 0 退出；② "make check" 0 退出（含 check-contract-sync 迁移号门禁）；③ up/down 成对且幂等（IF NOT EXISTS）；④ 独立 commit feat(migration)，正文写机理；⑤ push gitea。

### T2 OSS/AAA 建号写接口（会话 B）
范围：POST /resources（type OLT/SPLITTER，code 全局唯一 40900 语义，address 必填）、POST /ports（resource 归属 + port_code/quad_code 唯一）、POST /lo-accounts（loid 唯一、customer 必填、offer/qos 必填）；均挂既有域门禁（menu:resource / menu:loaccount）；routes_gen + fields.md 契约同步；handler 测试覆盖 201/422/409 三态。
验收：① "go build ./..." 0 退出；② 新增测试 go test ./internal/httpapi/admin/... -run 新测试名 0 退出；③ "make check" 0 退出；④ 独立 commit；⑤ push gitea。

### T3 导入工具（会话 C）
范围：scripts/import-kaihu/（Python3 纯 stdlib）：xlsx 解析 → 结构化中间 JSON → 校验报告（行数 347/账号 347 唯一/SN 330 唯一/缺 PON 16/缺 SN 17/缺 VLAN 35/离群 VLAN 按组众数偏离/拆机 1）→ 两段式 --dry-run（默认，只出计划与报告）/ --apply（--plan-file 消费计划：API 段 asset-models/assets 批次/products/customers；SQL 段 resources/ports/lo_accounts/quad_links，幂等 upsert 自然键）→ 导入后 reconciliation（Excel↔库 行数核对）→ POST /import-tasks 登记（clientKey 幂等，按 2026-09-03 裁定）。
验收：① 对真实 xlsx --dry-run 报告数字逐项命中（347/347/330/16/17/35/1）；② 同输入两次 dry-run 输出逐字节一致（确定性）；③ apply 幂等：第二遍 0 新建；④ 报告含 needsReview 明细清单（离群 VLAN/缺字段行）；⑤ python3 -m py_compile 全通过 + 独立 commit；⑥ push gitea。

### T4 合并执行（负责人）
合并顺序：T1 → T2（各自 rebase 后 ff-only）→ 102 CI 部署 → 迁移生效复核（psql \d）→ T3 apply 分四段执行（资产段/资源段/账号段/绑定段）→ SQL+API 双重复核（行数、抽样、四码）。
复核基线：lo_accounts 存量新增 347（1 CLOSED）；ports 存量新增 ~331 且 svlan 非空 312；assets.sn 非空 +330；product_offers 增 20M/50M；customers +347（realName PENDING）；quad_links +346；import-tasks 有登记。

### T5 收尾（负责人）
worktree remove + branch -d + push --delete（ff 失败禁止清理链）；会话归档；反思入库 self-evolving。

## 5. 数据质量红线
- 缺 PON/SN/VLAN 行照常入档，字段留空 + needsReview 清单，禁止编造补值
- 非标账号（Office301/OWC_Office_GF/POWC_PALYYT_001）白名单放行，原因写报告
- OWPAL19516_02 按同客户二次线路：独立 lo_account，quad_link 1:N
- 30M×1：建 50M 档后按 50M 归档并在报告注明（不改原始值，原始值存报告 JSON）
- 业务时区 Asia/Manila：日期列按当地自然日转 timestamptz（与 2026-08-21 裁定一致）
