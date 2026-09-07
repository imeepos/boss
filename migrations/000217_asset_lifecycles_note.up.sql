-- 000217: asset_lifecycles 增 note 列(W8 数据修复留痕;terms.md §4 修复裁定 2026-09-07)。
-- 背景:开户记录导入(LINKED 但资产 IN_STOCK)归属修复需在状态轨迹行带 fix 来源标记,
-- 轨迹表原无自由文本列;通用留痕列,后续人工修复/盘点修正同样受益。

ALTER TABLE asset_lifecycles ADD COLUMN IF NOT EXISTS note VARCHAR(255);
