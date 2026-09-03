-- 下发日志驱动来源留痕:记录产生本次结果的设备通道(log 桩/telnet 仿真链/tl1 北向)。
-- 审计可溯源:SUCCESS 语义随通道不同(仿真链仅代表仿真器认可),详情页必须可区分。
ALTER TABLE provision_logs ADD COLUMN IF NOT EXISTS driver TEXT NOT NULL DEFAULT '';