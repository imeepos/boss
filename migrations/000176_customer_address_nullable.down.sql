-- 回退:重建 NOT NULL。若存在 address_id IS NULL 的存量行,本语句按预期失败,
-- 需先人工归位(回填地址或移除档案)再执行。
ALTER TABLE customers ALTER COLUMN address_id SET NOT NULL;
