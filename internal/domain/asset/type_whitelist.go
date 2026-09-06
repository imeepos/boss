// 资产类型写入白名单(P4-T2,Lead 裁定 2026-09-06 类型归一)。
// 背景:assets.type 此前为自由文本,方言滋生(102 实查 ONU/光猫/MI-ONU/SMOKE 并存);
// 裁定权威类型码 ONU,存量'光猫'经迁移 000191 归一,MI-ONU/SMOKE 系 e2e 残留
// 不入类型体系(由 scripts/ops/clean-asset-type-residue.sh 清理)。
// 取值依据(调研结论,见 fields.md §4.1):ONU=权威码(172 行存量);
// ROUTER=fields.md「光猫/ONU/路由器」既有业务口径;OLT=型号字典合法类别
// (域内测试 pg_model_test.go 派生路径既有取值,局端设备入台账场景)。
// 巡检侧 SQL 白名单见 scripts/ops/db-patrol-gate.sh ASSET-TYPE-UNKNOWN 查,
// 两处需同步维护(防新方言是同一口径的代码层+巡检层双闸)。

package asset

import (
	"fmt"
	"strings"
)

// ErrTypeNotAllowed 类型白名单外写入(42200);Allowed 携带当前合法集合供前端/调用方自纠。
type ErrTypeNotAllowed struct {
	Type    string
	Allowed []string
}

func (e *ErrTypeNotAllowed) Error() string {
	return fmt.Sprintf("asset: type %q 不在白名单内(合法值: %s)", e.Type, strings.Join(e.Allowed, "/"))
}

// TypeWhitelist 资产类型受控字典(新增合法值须同步:本文件+巡检 SQL+fields.md §4.1+前端 ASSET_TYPES)。
var TypeWhitelist = []string{"ONU", "ROUTER", "OLT"}

// ValidateType 类型写入闸门:建档/编辑的最终 type(含型号派生)必须命中白名单,
// 空串同样拒绝(建档 type 与 modelId 至少其一,双缺时给出可读错误而非 DB 23514)。
func ValidateType(t string) error {
	for _, ok := range TypeWhitelist {
		if t == ok {
			return nil
		}
	}
	return &ErrTypeNotAllowed{Type: t, Allowed: TypeWhitelist}
}
