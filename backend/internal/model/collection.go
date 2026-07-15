package model

// Collection 请求集合，支持嵌套（ParentID）以组织文件夹结构。
type Collection struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	ProjectID uint           `gorm:"index" json:"projectId"`
	ParentID  *uint          `gorm:"index" json:"parentId"`
	Name      string         `gorm:"size:256" json:"name"`
	SortOrder int            `json:"sortOrder"`
	// Variables 集合级变量（JSON []EnvVar），默认生效，无启用/激活概念。
	Variables string         `gorm:"type:text" json:"variables"`
	Requests  []SavedRequest `json:"requests,omitempty"`
}
