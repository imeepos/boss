package httpx

// MaskPhone 手机号脱敏:保留前 3 后 4。
func MaskPhone(p string) string {
	if len(p) >= 11 {
		return p[:3] + "****" + p[len(p)-4:]
	}
	return p
}
