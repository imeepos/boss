package geo

import (
	"context"
	"errors"
)

// 国际地理基础数据域(ISO 3166-1/2 + 多语言译名 + 时区/货币/电话码关联)。
// 数据模型见 migrations/000038,字段口径见 docs/contract/fields.md 1.5.1。

// ErrNotFound 国家/区划不存在。
var ErrNotFound = errors.New("geo: not found")

// ErrDuplicate 唯一键冲突(alpha2/alpha3/numeric/完整区划码/译名唯一键)。
var ErrDuplicate = errors.New("geo: duplicate")

// Country 国家(ISO 3166-1 + UN M49)。
type Country struct {
	Alpha2        string `json:"alpha2"`        // 'PH',全局外联主键
	Alpha3        string `json:"alpha3"`        // 'PHL'
	NumericCode   string `json:"numericCode"`   // '608',=UN M49 国家码
	ShortName     string `json:"shortName"`     // English short name
	FullName      string `json:"fullName"`      // 全称,可空
	Status        string `json:"status"`        // INDEPENDENT / DISCONTINUED
	ContinentCode string `json:"continentCode"` // AS/EU/NA/SA/AF/OC/AN
	M49Region     string `json:"m49Region"`     // M49 洲/子区域码,可空
	PostalRegex   string `json:"postalRegex"`   // 邮编正则,可空
	IsActive      bool   `json:"isActive"`      // 停用码软删除保留
	DisplayName   string `json:"displayName"`   // 按 locale 联译名,空回落 short_name
}

// CountryName 国家译名(独立译名表,GeoNames alternatenames 范式)。
type CountryName struct {
	Locale   string `json:"locale"` // 'zh-Hans'/'en'
	Name     string `json:"name"`
	NameType string `json:"nameType"` // STANDARD/SHORT/ALIAS/HISTORIC
}

// Currency 国家货币(ISO 4217)。
type Currency struct {
	Currency  string `json:"currency"` // 'PHP'
	IsPrimary bool   `json:"isPrimary"`
	MinorUnit int16  `json:"minorUnit"` // 小数位:JPY=0,USD=2
}

// CountryAttrs 国家关联属性(一对多集合,整体替换)。
type CountryAttrs struct {
	TimeZones    []string   `json:"timeZones"` // IANA 时区
	Currencies   []Currency `json:"currencies"`
	CallingCodes []string   `json:"callingCodes"` // E.164 国家码
}

// CountryDetail 国家详情:主档 + 译名 + 关联属性。
type CountryDetail struct {
	Country
	Names []CountryName `json:"names"`
	Attrs CountryAttrs  `json:"attrs"`
}

// Subdivision 行政区划(ISO 3166-2 完整码,自引用树)。
type Subdivision struct {
	Code          string `json:"code"`          // 'PH-NCR',主键
	CountryCode   string `json:"countryCode"`   // 所属国家 alpha-2
	ParentCode    string `json:"parentCode"`    // 自引用父码,可空
	Level         int16  `json:"level"`         // 1~4,各国深度不同
	Category      string `json:"category"`      // state/province/region/municipality...
	OSMAdminLevel int16  `json:"osmAdminLevel"` // OSM 行政层级标尺 2~10
	GeonameID     int64  `json:"geonameId"`     // GeoNames 挂接,0=未挂
	IsActive      bool   `json:"isActive"`
	DisplayName   string `json:"displayName"` // 按 locale 联译名,空回落 code
}

// SubdivisionName 区划译名。
type SubdivisionName struct {
	Locale   string `json:"locale"`
	Name     string `json:"name"`
	NameType string `json:"nameType"` // STANDARD/SHORT/ALIAS/HISTORIC/PINYIN/ROMANIZED
}

// CountryNameRow 导入用译名行(带所属国家码)。
type CountryNameRow struct {
	CountryCode string      `json:"countryCode"`
	Name        CountryName `json:"name"`
}

// SubdivisionNameRow 导入用区划译名行(带所属区划码)。
type SubdivisionNameRow struct {
	SubdivisionCode string          `json:"subdivisionCode"`
	Name            SubdivisionName `json:"name"`
}

// ImportData 批量导入载荷:全量或增量 upsert,已有主键/唯一键则更新。
type ImportData struct {
	Countries        []Country            `json:"countries"`
	CountryNames     []CountryNameRow     `json:"countryNames"`
	Subdivisions     []Subdivision        `json:"subdivisions"`
	SubdivisionNames []SubdivisionNameRow `json:"subdivisionNames"`
}

// ImportCounts 导入结果计数。
type ImportCounts struct {
	Countries        int `json:"countries"`
	CountryNames     int `json:"countryNames"`
	Subdivisions     int `json:"subdivisions"`
	SubdivisionNames int `json:"subdivisionNames"`
}

// GeoService 地理基础数据维护接口(增删改查;停用为软删除;Import 批量 upsert)。
type GeoService interface {
	ListCountries(ctx context.Context, locale string) ([]Country, error)
	GetCountry(ctx context.Context, alpha2 string) (*CountryDetail, error)
	CreateCountry(ctx context.Context, c Country) error
	UpdateCountry(ctx context.Context, alpha2 string, c Country) error
	SetCountryActive(ctx context.Context, alpha2 string, active bool) error
	AddCountryName(ctx context.Context, alpha2 string, n CountryName) error
	RemoveCountryName(ctx context.Context, alpha2, locale, nameType string) error
	ReplaceCountryAttrs(ctx context.Context, alpha2 string, attrs CountryAttrs) error

	ListSubdivisions(ctx context.Context, countryCode, locale string) ([]Subdivision, error)
	GetSubdivision(ctx context.Context, code string) (*Subdivision, error)
	ListSubdivisionNames(ctx context.Context, code string) ([]SubdivisionName, error)
	CreateSubdivision(ctx context.Context, s Subdivision) error
	UpdateSubdivision(ctx context.Context, code string, s Subdivision) error
	SetSubdivisionActive(ctx context.Context, code string, active bool) error
	AddSubdivisionName(ctx context.Context, code string, n SubdivisionName) error
	RemoveSubdivisionName(ctx context.Context, code, locale, nameType string) error

	Import(ctx context.Context, data ImportData) (ImportCounts, error)

	// GetAddress 按 id 查地址(path/name/level);未命中返回 nil。
	GetAddress(ctx context.Context, id int64) (*AddressInfo, error)
}

// AddressInfo 地址简要信息(四码 addrCode 展示用)。
type AddressInfo struct {
	ID    int64  `json:"id"`
	Path  string `json:"path"`  // LTREE 路径如 root.luzon.ncr.manila
	Name  string `json:"name"`
	Level int    `json:"level"` // 1市 2区 3街道 4小区 5楼栋
}
