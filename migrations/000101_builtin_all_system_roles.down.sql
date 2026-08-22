BEGIN;

-- down 无法区分哪些 false 行原本就是误标,保持不动(仅数据修复,无结构变更)。

COMMIT;
