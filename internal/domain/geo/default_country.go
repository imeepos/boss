package geo

// 默认国家配置(biz_params geo.default_country,值=alpha-2 如 "PH")。
// 写路径复用既有 PUT /params/:key(menu:params);本文件只负责读与值校验。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
)

// DefaultCountryKey 默认国家的 biz_params 键。
const DefaultCountryKey = "geo.default_country"

// alpha2Re 合法 ISO 3166-1 alpha-2:两个 ASCII 字母。
var alpha2Re = regexp.MustCompile("^[A-Za-z]{2}$")

// GetDefaultCountry 读默认国家;未配置返回空值对象(Configured=false)。
// 值非法(非两位字母)按未配置处理并留 [geo] 可 grep 告警。
func (s *PGStore) GetDefaultCountry(ctx context.Context) (DefaultCountry, error) {
	raw, err := getParamRaw(ctx, s, DefaultCountryKey)
	if err != nil {
		return DefaultCountry{}, fmt.Errorf("geo: get default country: %w", err)
	}
	return parseDefaultCountry(raw), nil
}

// getParamRaw 读 biz_params 原始 JSONB 文本(缺行返回空串)。
func getParamRaw(ctx context.Context, s *PGStore, key string) (string, error) {
	var raw string
	err := s.db.QueryRow(ctx,
		`SELECT COALESCE(value::text,'') FROM biz_params WHERE key=$1`, key).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return raw, nil
}

// parseDefaultCountry 参数值→读视图:JSON 字符串解包后归一为大写 alpha-2;
// 未配置/非法均返回未配置态(契约:不兜底部署主国家,由前端决定缺省态)。
func parseDefaultCountry(raw string) DefaultCountry {
	v := raw
	if json.Unmarshal([]byte(raw), &v) != nil {
		v = raw
	}
	v = strings.ToUpper(strings.TrimSpace(v))
	if !alpha2Re.MatchString(v) {
		if raw != "" {
			log.Printf("[geo] DEFAULT COUNTRY INVALID key=%s value=%q treat-as-unconfigured", DefaultCountryKey, raw)
		}
		return DefaultCountry{}
	}
	return DefaultCountry{CountryCode: v, Configured: true}
}
