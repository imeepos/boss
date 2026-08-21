// Package clock 业务时区统一口径。
//
// 裁定(见 docs/notes/adopted/2026-08-21-business-timezone.md):
//   - 存储/DB会话/容器时区固定 UTC(timestamptz 全量);
//   - API 传递绝对时刻(RFC3339 带偏移);
//   - 展示时区归客户端;
//   - 自然日/月边界一律按业务时区(配置 BOSS_TIMEZONE,默认 Asia/Manila),
//     禁止散落 time.Local / 裸 now() 格式化日期。
package clock

import (
	"fmt"
	"sync"
	"time"

	_ "time/tzdata" // 嵌入 IANA 库,distroless 容器无 /usr/share/zoneinfo 也可用
)

var (
	mu  sync.RWMutex
	loc = time.UTC
)

// Set 设定业务时区(IANA 名);空串保持现状(测试/降级场景)。
func Set(name string) error {
	if name == "" {
		return nil
	}
	l, err := time.LoadLocation(name)
	if err != nil {
		return fmt.Errorf("clock: load timezone %s: %w", name, err)
	}
	mu.Lock()
	loc = l
	mu.Unlock()
	return nil
}

// Location 业务时区。
func Location() *time.Location {
	mu.RLock()
	defer mu.RUnlock()
	return loc
}

// Now 当前时刻,墙钟落在业务时区。
func Now() time.Time {
	return time.Now().In(Location())
}

// DayBounds 返回 t 所在业务日的绝对时间区间 [start, end);
// t 任意偏移均可,先归一业务时区再切日界。
func DayBounds(t time.Time) (time.Time, time.Time) {
	l := Location()
	y, m, d := t.In(l).Date()
	start := time.Date(y, m, d, 0, 0, 0, 0, l)
	return start, start.AddDate(0, 0, 1)
}
