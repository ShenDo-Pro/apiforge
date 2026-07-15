package model

// SavedRequest 保存的请求，归属某个集合，可含多条历史记录。
// Headers/Body 以 JSON 文本存储，便于前端反序列化与编辑。
// Protocol 标记请求类型（http/ws/mqtt/...），空值按 http 处理。
type SavedRequest struct {
	ID        uint            `gorm:"primaryKey" json:"id"`
	CollectionID uint         `gorm:"index" json:"collectionId"`
	Name      string          `gorm:"size:256" json:"name"`
	Protocol  string          `gorm:"size:16" json:"protocol"`
	Method    string          `gorm:"size:16" json:"method"`
	URL       string          `gorm:"type:text" json:"url"`
	Headers   string          `gorm:"type:text" json:"headers"`
	Body      string          `gorm:"type:text" json:"body"`
	// 脚本与提取规则（文本存储，前端在发送闭环中执行/应用）。
	PreRequestScript string        `gorm:"type:text" json:"preRequestScript"` // 预请求脚本 JS
	TestScript       string        `gorm:"type:text" json:"testScript"`       // 测试脚本 JS
	ExtractRules     string        `gorm:"type:text" json:"extractRules"`     // GUI 提取规则 JSON
	Auth             string        `gorm:"type:text" json:"auth"`             // 鉴权配置 JSON（None/Bearer/Basic/APIKey/OAuth2）
	Histories []RequestHistory `json:"histories,omitempty"`
}
