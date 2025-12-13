package config

import "time"

const (
	AccessExpireTime  = time.Minute * 20
	RefreshExpireTime = time.Hour * 24
)
