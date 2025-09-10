package models

// PublicStats — bazadan olingan toza ma'lumot

// PublicStatsDisplay — foydalanuvchiga ko'rsatiladigan format
type PublicStatsDisplay struct {
	TotalUsers string `json:"total_users"`
	ToolsCount int    `json:"tools_count"`
	TotalUsage string `json:"total_usage"`
}
type PublicStats struct {
	TotalUsers int64
	ToolsCount int
	TotalUsage int64
}
