-- 000191: 资产类型归一(P4-T2,Lead 裁定 2026-09-06)。
-- 裁定:权威类型码 ONU;'光猫'为历史方言(102 实查 49 行),一律 UPDATE 归一。
-- 配套:assets.type 写入白名单(internal/domain/asset/type_whitelist.go,
-- ONU/ROUTER/OLT)拦截新方言;MI-ONU/SMOKE 系 e2e 残留不入类型体系,
-- 由 scripts/ops/clean-asset-type-residue.sh 按"无引用才删"清理,不经迁移。
-- down 不恢复:方言归一属不可逆裁定(把 ONU 行回写'光猫'只会复活方言),
-- 故 down 仅留注释,不执行任何 SQL。

BEGIN;

UPDATE assets SET type = 'ONU' WHERE type = '光猫';

COMMIT;
