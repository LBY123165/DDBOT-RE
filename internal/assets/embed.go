package assets

import "embed"

// DistFS 包含前端构建产物（编译时嵌入）
//
//go:embed dist
var DistFS embed.FS
