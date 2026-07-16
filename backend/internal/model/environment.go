package model

// EnvVar 是环境/全局/集合变量的原子单元，以 JSON 数组形式存入
// Environment.Values 或 Collection.Variables。
// Enabled 控制是否参与替换（集合/全局变量恒为 true）；
// Secret 标记机密变量，界面以掩码显示，数据库仍明文存储（仅展示层保护）。
type EnvVar struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Enabled bool   `json:"enabled"`
	Secret  bool   `json:"secret"`
}

// Environment 环境定义（含全局变量），挂在项目下、多人共享。
// Kind 区分普通环境与全局变量单例：global 每项目仅一行。
type Environment struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	ProjectID uint   `gorm:"index:idx_env_proj_kind,priority:1" json:"projectId"`
	Kind      string `gorm:"size:8;index:idx_env_proj_kind,priority:2" json:"kind"` // "env" | "global"
	Name      string `gorm:"size:256" json:"name"`
	Values    string `gorm:"type:text" json:"values"` // JSON: []EnvVar
	SortOrder int    `json:"sortOrder"`
}
