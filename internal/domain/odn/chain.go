package odn

// W3 ODN 资源表批量导入(P-INFRA-1 W3;需求源 docs/ODN网络资源表模板.xlsx):模板一行=一条完整资源链。
// 说明页六规则即校验规则:
// ① 层级顺序 OCC-ODB-OBD-一级分光-SDB-SBD-二级分光,断链拒绝;
// ② 前 N 数据行为示例行,过滤不导入(批次层负责);
// ③ 未知字段留空不猜填:留空=不入库该字段,枚举未知值拒绝;
// ④ 示例行/规划态默认 PLANNED,绝不当作已安装/在网;
// ⑤ 总分光比=一级×二级自动计算,与填报列不一致拒绝该行;
// ⑥ 一行一条资源关系,指纹去重并报告(批次层负责)。

import (
	"fmt"
	"strconv"
)

// 链行结果状态(逐行可观测)。
const (
	ChainRowImported       = "imported"
	ChainRowFailed         = "failed"
	ChainRowSkippedExample = "skipped_example"
	ChainRowDuplicate      = "duplicate"
)

// ChainImportInput 一次导入批次(exampleRowCount=说明页约定的示例行数,正式录入从其后开始)。
type ChainImportInput struct {
	ExampleRowCount int                `json:"exampleRowCount"`
	Rows            []ResourceChainRow `json:"rows"`
}

// ResourceChainRow 模板 ODN资源 sheet 一行(23 列;枚举列=模板中文标签原文)。
type ResourceChainRow struct {
	RowNo          int    `json:"rowNo"`
	ResourceStatus string `json:"resourceStatus"`
	SiteCode       string `json:"siteCode"`
	SiteName       string `json:"siteName"`
	OltCode        string `json:"oltCode"`
	OdfCode        string `json:"odfCode"`
	OdfPort        string `json:"odfPort"`
	OccCode        string `json:"occCode"`
	OdbCode        string `json:"odbCode"`
	ObdCode        string `json:"obdCode"`
	Split1Ratio    string `json:"split1Ratio"`
	Split1Port     string `json:"split1Port"`
	SdbCode        string `json:"sdbCode"`
	SbdCode        string `json:"sbdCode"`
	Split2Ratio    string `json:"split2Ratio"`
	Split2Port     string `json:"split2Port"`
	TotalSplit     string `json:"totalSplit"`
	FiberCode      string `json:"fiberCode"`
	FrTo           string `json:"frTo"`
	PortStatus     string `json:"portStatus"`
	LayingMethod   string `json:"layingMethod"`
	RowStatus      string `json:"rowStatus"`
	PeceStatus     string `json:"peceStatus"`
	Remark         string `json:"remark"`
}

// ChainRecord 归一化链记录(入库口径;枚举已映射状态码;空串/0=该字段不入库)。
type ChainRecord struct {
	LineNo       int
	Lifecycle    string
	SiteCode     string
	SiteName     string
	OltCode      string
	OdfCode      string
	OdfPort      string
	OccCode      string
	OdbCode      string
	ObdCode      string
	Split1       int
	Split1Port   string
	SdbCode      string
	SbdCode      string
	Split2       int
	Split2Port   string
	TotalSplit   int
	FiberCode    string
	FrTo         string
	PortStatus   string
	LayingMethod string
	RowStatus    string
	PeceStatus   string
	Remark       string
	Fingerprint  string
}

// 说明页枚举字典(枚举 sheet):模板中文标签 → 系统状态码。
// 资源状态映射(裁定④):规划→PLANNED;已安装/已测试→IN_BUILD;在用→IN_SERVICE;已报废→RETIRED。
var chainResourceStatus = map[string]string{
	"规划": "PLANNED", "已安装": "IN_BUILD", "已测试": "IN_BUILD", "在用": "IN_SERVICE", "已报废": "RETIRED",
}

// 端口状态 → terms.md 端口四态。
var chainPortStatus = map[string]string{"可用": "IDLE", "已使用": "USED", "已预留": "RESERVED", "已封锁": "DISABLED"}

// 敷设方式 / ROW 状态 / PECE 状态(W4 路权与许可工作流的数据基础)。
var chainLayingMethod = map[string]string{"架空": "AERIAL", "地下": "UNDERGROUND", "海缆": "SUBMARINE", "微沟槽": "MICROTRENCH", "室内": "INDOOR"}
var chainRowStatus = map[string]string{"未开始": "NOT_STARTED", "待处理": "PENDING", "已批准": "APPROVED", "已过期": "EXPIRED", "不适用": "NA"}
var chainPeceStatus = map[string]string{"待签署": "PENDING_SIGN", "已签署": "SIGNED", "已盖章": "STAMPED", "不适用": "NA"}

// splitRatioEnum 枚举 sheet 分光比字典(1:N)。
var splitRatioEnum = map[string]int{"1:2": 2, "1:4": 4, "1:8": 8, "1:16": 16, "1:32": 32, "1:64": 64, "1:128": 128}

// ValidateChainRow 归一化并校验一行;返回 (记录, 拒绝原因),原因空串=通过。
func ValidateChainRow(in ResourceChainRow, lineNo int) (*ChainRecord, string) {
	r := in
	trimRow(&r)
	rec := &ChainRecord{LineNo: lineNo}
	// ④ 生命周期:资源状态映射;留空=PLANNED(状态原则:绝不当作已安装/在网)
	if r.ResourceStatus == "" {
		rec.Lifecycle = "PLANNED"
	} else {
		v, ok := chainResourceStatus[r.ResourceStatus]
		if !ok {
			return nil, "资源状态非法: " + r.ResourceStatus
		}
		rec.Lifecycle = v
	}
	rec.SiteCode, rec.SiteName, rec.OltCode = r.SiteCode, r.SiteName, r.OltCode
	if reason := validateChainEnums(&r, rec); reason != "" {
		return nil, reason
	}
	if reason := validateChainCodes(&r, rec); reason != "" {
		return nil, reason
	}
	if reason := checkChainHierarchy(rec); reason != "" {
		return nil, reason
	}
	if reason := applyTotalSplit(rec, r.TotalSplit); reason != "" {
		return nil, reason
	}
	rec.Fingerprint = chainFingerprint(rec)
	return rec, ""
}

// checkChainHierarchy 规则①:后继节点出现而前置缺失即断链(说明页层级顺序)。
func checkChainHierarchy(r *ChainRecord) string {
	switch {
	case r.OdbCode != "" && r.OccCode == "":
		return "层级断裂: ODB 缺上级 OCC"
	case r.ObdCode != "" && r.OdbCode == "":
		return "层级断裂: OBD 缺上级 ODB"
	case r.Split1 > 0 && r.ObdCode == "":
		return "层级断裂: 一级分光比缺 OBD"
	case r.Split1Port != "" && r.Split1 == 0:
		return "半链: 一级分光端口缺一级分光比"
	case r.SdbCode != "" && r.Split1Port == "":
		return "层级断裂: SDB 缺一级分光端口"
	case r.SbdCode != "" && r.SdbCode == "":
		return "层级断裂: SBD 缺上级 SDB"
	case r.Split2 > 0 && r.SbdCode == "":
		return "层级断裂: 二级分光比缺 SBD"
	case r.Split2Port != "" && r.Split2 == 0:
		return "半链: 二级分光端口缺二级分光比"
	case r.TotalSplit > 0 && r.Split1 == 0:
		return "层级断裂: 总分光比缺一级分光比"
	}
	return ""
}

// applyTotalSplit 规则⑤:总分光比=一级×二级;填报不一致拒绝;留空自动计算。
func applyTotalSplit(r *ChainRecord, filled string) string {
	computed := r.Split1 * ratioFactor(r.Split2)
	if filled != "" {
		v, err := strconv.Atoi(filled)
		if err != nil || v <= 0 {
			return "总分光比非法: " + filled
		}
		if computed > 0 && v != computed {
			return fmt.Sprintf("总分光比不一致: 填报 %d,一级×二级=%d", v, computed)
		}
		r.TotalSplit = v
		return ""
	}
	r.TotalSplit = computed
	return ""
}

// ratioFactor 二级分光缺省时乘一级(单级链总分光比=一级)。
func ratioFactor(split2 int) int {
	if split2 > 0 {
		return split2
	}
	return 1
}
