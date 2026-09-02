-- 下发日志留痕增强:记录完整设备指令与原始应答(后台日志详情页)。
-- commands: 顺序执行的设备指令(换行分隔);device_response: 设备原始应答/错误。
ALTER TABLE provision_logs ADD COLUMN IF NOT EXISTS commands TEXT NOT NULL DEFAULT '';
ALTER TABLE provision_logs ADD COLUMN IF NOT EXISTS device_response TEXT NOT NULL DEFAULT '';
