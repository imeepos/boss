package odn

// W3 资源链导入校验单测:说明页六规则(①③④⑤ + 枚举字典 + 指纹去重基础)。
// ②示例行过滤与⑥批次去重在 ImportResourceChains 存储层,依赖 DB,由 102 验收覆盖。

import (
	"strings"
	"testing"
)

const tdq = ""

func chainRowFull() ResourceChainRow {
	return ResourceChainRow{
		RowNo: 6, ResourceStatus: "在用", SiteCode: "SITE001", SiteName: "示例机房", OltCode: "SITE001_OLT001",
		OdfCode: "SITE001_ODF001_A", OdfPort: "ODF-A-01", OccCode: "OCC001", OdbCode: "ODB001",
		ObdCode: "OBD001", Split1Ratio: "1:8", Split1Port: "P01", SdbCode: "SDB001", SbdCode: "SBD001",
		Split2Ratio: "1:8", Split2Port: "P01", TotalSplit: "64", FiberCode: "F01", PortStatus: "可用",
		LayingMethod: "架空", RowStatus: "待处理", PeceStatus: "待签署", Remark: "备注",
	}
}

func TestValidateChainRowOK(t *testing.T) {
	rec, reason := ValidateChainRow(chainRowFull(), 6)
	if reason != tdq {
		t.Fatalf("完整链行应通过,实际拒绝: %s", reason)
	}
	if rec.Lifecycle != "IN_SERVICE" {
		t.Errorf("在用应映射 IN_SERVICE,实际 %s", rec.Lifecycle)
	}
	if rec.Split1 != 8 || rec.Split2 != 8 || rec.TotalSplit != 64 {
		t.Errorf("分光比解析错误: %d/%d/%d", rec.Split1, rec.Split2, rec.TotalSplit)
	}
	if rec.PortStatus != "IDLE" || rec.LayingMethod != "AERIAL" || rec.RowStatus != "PENDING" || rec.PeceStatus != "PENDING_SIGN" {
		t.Errorf("枚举映射错误: %s/%s/%s/%s", rec.PortStatus, rec.LayingMethod, rec.RowStatus, rec.PeceStatus)
	}
	if rec.OccCode != "OCC001" || rec.OdbCode != "ODB001" || rec.ObdCode != "OBD001" || rec.SdbCode != "SDB001" || rec.SbdCode != "SBD001" {
		t.Errorf("箱体编码解析错误")
	}
}

func TestValidateChainRowRules(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*ResourceChainRow)
		want   string
	}{
		{name: "总分光比不一致(规则⑤)", mutate: func(r *ResourceChainRow) { r.TotalSplit = "32" }, want: "总分光比不一致"},
		{name: "ODB缺OCC(规则①)", mutate: func(r *ResourceChainRow) { r.OccCode = tdq }, want: "层级断裂"},
		{name: "SBD缺SDB(规则①)", mutate: func(r *ResourceChainRow) { r.SdbCode = tdq }, want: "层级断裂"},
		{name: "二级端口缺二级比(规则①)", mutate: func(r *ResourceChainRow) { r.Split2Ratio = tdq }, want: "半链"},
		{name: "端口状态非法(规则③)", mutate: func(r *ResourceChainRow) { r.PortStatus = "已烧毁" }, want: "端口状态非法"},
		{name: "分光比非枚举(规则③)", mutate: func(r *ResourceChainRow) { r.Split1Ratio = "1:3" }, want: "一级分光比非法"},
		{name: "资源状态非法(规则③)", mutate: func(r *ResourceChainRow) { r.ResourceStatus = "已丢失" }, want: "资源状态非法"},
		{name: "OCC列填ODB前缀", mutate: func(r *ResourceChainRow) { r.OccCode = "ODB009" }, want: "编码前缀须为 OCC"},
	}
	for _, c := range cases {
		row := chainRowFull()
		c.mutate(&row)
		_, reason := ValidateChainRow(row, 6)
		if reason == tdq {
			t.Errorf("%s: 期望拒绝(%s),实际通过", c.name, c.want)
			continue
		}
		if !strings.Contains(reason, c.want) {
			t.Errorf("%s: 拒绝原因 %q 不含 %q", c.name, reason, c.want)
		}
	}
}

func TestValidateChainRowLifecycleDefault(t *testing.T) {
	// 规则④:资源状态留空=PLANNED,绝不当作已安装/在网;规划态同。
	for _, status := range []string{tdq, "规划"} {
		row := chainRowFull()
		row.ResourceStatus = status
		rec, reason := ValidateChainRow(row, 6)
		if reason != tdq {
			t.Fatalf("状态 %q 应通过: %s", status, reason)
		}
		if rec.Lifecycle != "PLANNED" {
			t.Errorf("状态 %q 应落 PLANNED,实际 %s", status, rec.Lifecycle)
		}
	}
	row := chainRowFull()
	row.ResourceStatus = "已安装"
	rec, _ := ValidateChainRow(row, 6)
	if rec.Lifecycle != "IN_BUILD" {
		t.Errorf("已安装应映射 IN_BUILD,实际 %s", rec.Lifecycle)
	}
}

func TestChainFingerprintDedup(t *testing.T) {
	// 规则⑥:全等行同指纹;尾随空格归一后同指纹;差异行不同指纹。
	a, _ := ValidateChainRow(chainRowFull(), 6)
	bRow := chainRowFull()
	bRow.SiteName = "示例机房 "
	b, _ := ValidateChainRow(bRow, 7)
	if a.Fingerprint != b.Fingerprint {
		t.Errorf("尾随空格归一后指纹应一致")
	}
	cRow := chainRowFull()
	cRow.Remark = "不同备注"
	c, _ := ValidateChainRow(cRow, 8)
	if a.Fingerprint == c.Fingerprint {
		t.Errorf("备注差异行指纹不应相同")
	}
}

func TestValidateDeviceCodeBoxKinds(t *testing.T) {
	// 六类箱体设备码(扩展裁定):3 位编号 + 同址扩容后缀 -N;OBD/SBD 为新增类型。
	for _, code := range []string{"OCC001", "ODB001", "OBD001", "SDB001", "SBD001", "ODF002", "ODB001-2"} {
		if _, err := ValidateDeviceCode(code); err != nil {
			t.Errorf("%s 应为合法设备码: %v", code, err)
		}
	}
	if _, err := ValidateDeviceCode("OBD001-1"); err == nil {
		t.Errorf("扩容后缀 -1 应非法(规范 3.1)")
	}
	if RequiredParentKind(DevOBD) != DevODB || RequiredParentKind(DevSBD) != DevSDB {
		t.Errorf("箱内部件归属链错误")
	}
}

func TestValidateChainAttribution(t *testing.T) {
	// W3 收口 F2:prv/city/grid 三列一体;任一非空即三者必须齐全;grid 1~99。
	row := chainRowFull()
	row.PrvCode, row.CityPrefix, row.GridCode = "PHL001", "MNL", "5"
	rec, reason := ValidateChainRow(row, 6)
	if reason != "" {
		t.Fatalf("完整归属应通过: %s", reason)
	}
	if rec.PrvCode != "PHL001" || rec.CityPrefix != "MNL" || rec.GridCode != 5 {
		t.Errorf("归属解析错误: %s/%s/%d", rec.PrvCode, rec.CityPrefix, rec.GridCode)
	}

	cases := []struct{ name, prv, city, grid, want string }{
		{name: "只填网格", prv: "", city: "", grid: "5", want: "网格归属不完整"},
		{name: "缺城市", prv: "PHL001", city: "", grid: "5", want: "网格归属不完整"},
		{name: "网格码0", prv: "PHL001", city: "MNL", grid: "0", want: "网格码非法"},
		{name: "网格码100", prv: "PHL001", city: "MNL", grid: "100", want: "网格码非法"},
		{name: "网格码非数字", prv: "PHL001", city: "MNL", grid: "ab", want: "网格码非法"},
	}
	for _, c := range cases {
		rw := chainRowFull()
		rw.PrvCode, rw.CityPrefix, rw.GridCode = c.prv, c.city, c.grid
		_, reason := ValidateChainRow(rw, 6)
		if reason == "" || !strings.Contains(reason, c.want) {
			t.Errorf("%s: 期望拒绝(%s),实际 %q", c.name, c.want, reason)
		}
	}
}

func TestChainFingerprintAttribution(t *testing.T) {
	// 归属并入指纹:同链同归属=同指纹;同链无归属/异归属=异指纹(归属是资源关系一部分)。
	aRow := chainRowFull()
	aRow.PrvCode, aRow.CityPrefix, aRow.GridCode = "PHL001", "MNL", "5"
	a, _ := ValidateChainRow(aRow, 6)
	bRow := chainRowFull()
	bRow.PrvCode, bRow.CityPrefix, bRow.GridCode = "PHL001", "MNL", "5"
	b, _ := ValidateChainRow(bRow, 7)
	if a.Fingerprint != b.Fingerprint {
		t.Errorf("同链同归属指纹应一致")
	}
	cRow := chainRowFull()
	cRow.PrvCode, cRow.CityPrefix, cRow.GridCode = "PHL001", "MNL", "6"
	c, _ := ValidateChainRow(cRow, 8)
	if a.Fingerprint == c.Fingerprint {
		t.Errorf("异归属指纹不应一致")
	}
	d, _ := ValidateChainRow(chainRowFull(), 9)
	if a.Fingerprint == d.Fingerprint {
		t.Errorf("带归属与不带归属指纹不应一致")
	}
}
