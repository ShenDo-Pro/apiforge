package model

import "time"

// RequestHistory 一次发送的历史快照，便于回看与回填。
// Timings/ResponseHeaders 均为 JSON 文本，对应前端计时分解与头表展示。
type RequestHistory struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	SavedRequestID uint      `gorm:"index" json:"savedRequestId"`
	Method         string    `gorm:"size:16" json:"method"`
	URL            string    `gorm:"type:text" json:"url"`
	StatusCode     int       `json:"statusCode"`
	Proto          string    `gorm:"size:16" json:"proto"`
	ResponseHeaders string   `gorm:"type:text" json:"responseHeaders"`
	ResponseBody   string    `gorm:"type:text" json:"responseBody"`
	Timings        string    `gorm:"type:text" json:"timings"`
	CreatedAt      time.Time `json:"createdAt"`
}
