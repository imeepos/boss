// Package openapi 内嵌 REST 契约源码树(YAML),供运行时聚合为自包含 JSON 文档
// (GET /api/admin/v1/docs/openapi,页面 /base/apidocs)。文档数据单一事实源
// 就是仓库内契约本身,不再另维护文档站数据源。
package openapi

import "embed"

// FS 四端契约源码树:聚合根(*.yaml)+ 各端域文件目录。
//
//go:embed *.yaml admin open user worker
var FS embed.FS
