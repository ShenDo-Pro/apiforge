package model

// ProjectMember 项目成员中间表，承载 GitLab 式授权模型。
// Role 控制项目级角色：owner 全权限、maintainer 管集合/请求、developer 受 Permissions 约束。
// Permissions 为 JSON：{"add":bool,"edit":bool,"delete":bool}，仅对 developer 生效。
type ProjectMember struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	ProjectID   uint   `gorm:"uniqueIndex:uniq_proj_member" json:"projectId"`
	UserID      uint   `gorm:"uniqueIndex:uniq_proj_member" json:"userId"`
	Role        string `gorm:"size:16;default:developer" json:"role"`
	Permissions string `gorm:"type:text" json:"permissions"`
}
