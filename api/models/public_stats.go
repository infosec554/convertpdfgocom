package models

type PublicStats struct {
	TotalUsers int64 `json:"total_users"`
	ToolsCount int64 `json:"tools_count"`
	TotalUsage int64 `json:"total_usage"`
}

type PublicStatsDisplay struct {
	TotalUsers string `json:"total_users"`
	ToolsCount int64  `json:"tools_count"`
	TotalUsage string `json:"total_usage"`
}
