package model

import "time"

// Project 调试项目，参照 GitLab 命名空间概念。
// OwnerID 标记创建者，创建者天然拥有项目全部权限。
type Project struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"size:128" json:"name"`
	Description string         `gorm:"size:512" json:"description"`
	OwnerID     uint           `json:"ownerId"`
	CreatedAt   time.Time      `json:"createdAt"`
	Collections []Collection   `json:"collections,omitempty"`
	Members     []ProjectMember `json:"members,omitempty"`
}
