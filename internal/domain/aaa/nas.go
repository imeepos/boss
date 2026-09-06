package aaa

// AAA-A5(G7):per-NAS 客户端注册表。RADIUS 服务端按请求来源 IP 查表校验密钥
// (未注册/停用拒绝并留痕);CoA/强制下线使用目标 NAS 自己的密钥与端口;
// 全局密钥(BOSS_AAA_SECRET)降级为默认关闭的兼容开关回退项(fields.md §8J 迁移路径)。

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/ymm-001/boss/internal/domain/aaa/credential"
)

// NasVendor NAS 厂商类型(VSA 限速属性下发依据;GENERIC/未知走字符串属性兜底)。
type NasVendor string

// 厂商枚举(迁移 000196 CHECK 约束同源)。
const (
	NasVendorHuawei  NasVendor = "HUAWEI"
	NasVendorZTE     NasVendor = "ZTE"
	NasVendorGeneric NasVendor = "GENERIC"
)

// DefaultNasCoAPort NAS 动态授权默认端口(RFC 5176;注册表未配置时落库默认)。
const DefaultNasCoAPort = 3799

// 域错误(radius SecretSource/CoA 客户端按 errors.Is 分流,留痕分类依据)。
var (
	ErrNasNotFound  = errors.New("aaa: nas not registered")
	ErrNasDisabled  = errors.New("aaa: nas disabled")
	ErrNasDuplicate = errors.New("aaa: nas ip duplicated")
)

// NasClient NAS 客户端(管理面安全字段;密钥明文/密文永不进本结构)。
type NasClient struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	NasIP     string    `json:"nasIp"`
	Vendor    NasVendor `json:"vendor"`
	CoAPort   int       `json:"coaPort"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// NasAuth 协议面视角:注册表命中后的密钥材料(仅 RADIUS/CoA 链路内存使用)。
type NasAuth struct {
	Client NasClient
	Secret []byte // 解密后共享密钥(报文签名/组包用,不落日志不回显)
}

// NasResolver 协议面按来源 IP 取 NAS 密钥材料(RADIUS SecretSource/Handler/CoA 共用)。
type NasResolver interface {
	LookupNas(ctx context.Context, ip string) (*NasAuth, error)
}

// NasUpsert 管理面创建/更新载荷;SecretPlain 空=更新不改密钥(创建必填)。
type NasUpsert struct {
	Name        string
	NasIP       string
	SecretPlain string
	Vendor      NasVendor
	CoAPort     int
	Enabled     *bool // nil=更新保持原值(创建取 true)
}

// NasAdminService 管理面 CRUD(admin 路由消费;PGStore 实现)。
type NasAdminService interface {
	CreateNas(ctx context.Context, u NasUpsert) (int64, error)
	UpdateNas(ctx context.Context, id int64, u NasUpsert) error
	DeleteNas(ctx context.Context, id int64) error
	GetNas(ctx context.Context, id int64) (*NasClient, error)
	ListNasPage(ctx context.Context, q NasPage) (AdminPageResult[NasClient], error)
}

// NasPage NAS 注册表分页参数(keyword 匹配名称/IP;enabled:""/"true"/"false")。
type NasPage struct {
	Page     int
	PageSize int
	Keyword  string
	Vendor   string
	Enabled  string
}

// CredentialMaterial 密钥材料归一:CRED_KEY 非空直用;空=从全局 Secret 派生
// (derived=true,开发兜底,调用方必须打 ALERT;与 A1 cmd/aaa 口径一致)。
func CredentialMaterial(credKey, secret string) (string, bool) {
	if credKey != "" {
		return credKey, false
	}
	return "boss-aaa-cred-key|" + secret, true
}

// BuildCodec 凭据编解码器构建(NAS 密钥落库/还原复用既有密文体系,A1 同源)。
func BuildCodec(credKey, secret string) (*credential.Codec, error) {
	material, derived := CredentialMaterial(credKey, secret)
	if derived {
		log.Println("[aaa] CRED KEY ALERT: BOSS_AAA_CRED_KEY 未设置,使用 Secret 派生密钥(生产必须显式配置)")
	}
	return credential.New(material)
}