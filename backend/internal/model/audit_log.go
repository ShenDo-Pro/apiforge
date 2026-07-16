package model

import "time"

// AuditLog 记录已认证用户的写操作，用于安全审计与行为追溯（C9）。
type AuditLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"userId"`
	Username  string    `gorm:"size:64;index" json:"username"`
	Method    string    `gorm:"size:8" json:"method"`
	Path      string    `gorm:"size:255;index" json:"path"`
	Status    int       `json:"status"`
	CreatedAt time.Time `gorm:"index" json:"createdAt"`
}
