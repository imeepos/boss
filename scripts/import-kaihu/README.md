# 存量开户记录 xlsx 导入工具(scripts/import-kaihu)

数据源 docs/开户记录_20240106.xlsx(347 条有效开户,2023-11-14 ~ 2024-01-09)。
纯 Python3 标准库(zipfile/xml.etree/urllib/subprocess),无第三方依赖。
依据: docs/design/2026-09-07-legacy-kaihu-import.md(T3/§5 数据质量红线)、
docs/notes/adopted/2026-09-07-legacy-vlan-on-ports.md、2026-08-21-business-timezone.md、
2026-09-03-import-task-idempotency.md。

## 用法

解析 + 校验报告(默认 dry-run,不改任何数据):

    python3 scripts/import-kaihu/import_kaihu.py --dry-run --selfcheck --source docs/开户记录_20240106.xlsx --outdir /tmp/kaihu-dry

    产出: plan.json(每行一条标准化记录) + report.json(计数/needsReview/白名单/离群 VLAN)。
    --selfcheck 对硬指标逐项断言,任何不符退出码非 0。

执行导入(仅限负责人 T4,本任务禁止执行):

    export BOSS_ADMIN_KEY=<api key>       # 不写死,从环境读
    python3 scripts/import-kaihu/import_kaihu.py --apply --plan-file /tmp/kaihu-dry/plan.json --outdir /tmp/kaihu-dry

    API 段: asset-models(查重) -> asset-batches(存量开户导入) -> assets(loid=账号 幂等,40900=已存在)
            -> products(存量宽带20M/50M 两档) -> customers(347,按 name=账号 幂等) -> POST /import-tasks 登记
            (clientKey 幂等,重复登记覆盖统计数)。
    SQL 段: 生成 outdir/apply.sql,ssh 102 docker psql --csv 执行,末尾对账复核,任一计数不符整体失败。

机械验收(工具自带):

    sh scripts/import-kaihu/acceptance.sh docs/开户记录_20240106.xlsx /tmp/kaihu-dry

## 计划 JSON 记录字段

account 账号 / bandwidth_mbps 归档带宽(30M 按 50M 归档) / bandwidth_original 原始带宽 /
months 月数(缺为 null) / olt / pon_frame/pon_slot/pon_port PON 三段 / onu_no ONU 号 / sn / model /
svlan/cvlan/internet_cvlan/tr069_cvlan 四元组(缺为 null) / legacy_path=OCCxx-ODBxx-OBDxx-Pxx(缺段跳过,无损承接) /
port_code=P-<OLT>-<PON三段>-<ONU>(OLT/PON/ONU 任一缺即无端口) / opened_date 开户日期 ISO /
closed_date 拆机日期(Excel 序列号已换算) / remark 备注 / ownership 归属缺省 / needs_review 标记列表。

## 缺省归属(2026-09-07 裁定)

legal_entity=平台总公司(id=6) / region=集团(id=1,POST /customers 需真实区域,以集团根代「不限」,假设已列报告) /
地址=单一占位节点 addresses.path=legacy_import(needs_review=true,apply 自建,治理队列消化)。
lo_accounts.region_id/region_name 同取缺省;ports.address_id 由占位节点满足 NOT NULL。

## 假设与已知口径(报告 assumptions/needsReview 同步可见)

1. VLAN 四元组缺失硬指标按外层 svlan 缺失计(=35;全四空 34 行,另 1 行 OWTAC33398 缺 internet/TR069 入 vlan_partial)。
2. 30M×1 按 50M 归档,原始值 30 存报告 bandwidth_adjustments;缺带宽 4 行 offer 落默认 50M 档并列 needsReview。
3. 产品月费为导入假设值: 20M 默认 49.00、50M 默认 69.00(--fee-20m/--fee-50m 可调),T4 前负责人可复核;100M/200M 用既有档。
4. customers phone=PENDING 哨兵(name=账号可追溯,账号-客户映射以 lo_accounts.loid 为准);regionId=1 为「不限」代理。
5. 离群 VLAN 按 (OLT, PON板卡前两段) 分组众数偏离,入报告不阻断。
6. 端口仅为 OLT/PON/ONU 三要素齐备行建(数据实测 313);quad_links 同口径;与设计预估 ~331/+346 的差异源于
   ONU 缺 34/OLT 缺 11 行无法构成 port_code,实测数以 dry-run 报告为准,T4 复核基线按实测数对账。
7. 拆机 1 笔(OWPAL51531,序列 45279 -> 2023-12-19): lo_accounts.status=CLOSED;该行无端口无四码绑定。
8. apply 前置: 迁移 000202 已生效(VLAN 四列/legacy_path/contract_months);BOSS_ADMIN_KEY 账号数据范围覆盖平台总公司。

## 文件

import_kaihu.py CLI 入口 / xlsx_reader.py xlsx 解析 / normalize.py 行标准化 / report.py 报告+selfcheck /
api_client.py admin API 客户端 / apply_api.py API 段 / apply_sql.py SQL 段+对账 / acceptance.sh 机械验收 A/B/C。