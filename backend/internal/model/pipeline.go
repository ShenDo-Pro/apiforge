package model

import "time"

// Pipeline 测试流水线，归属某个项目，含有序步骤与运行历史。
// WebhookToken 为每个流水线生成的独立触发密钥，对外通过 Webhook URL 暴露。
type Pipeline struct {
	ID           uint            `gorm:"primaryKey" json:"id"`
	ProjectID    uint            `gorm:"index" json:"projectId"`
	Name         string          `gorm:"size:256" json:"name"`
	Description  string          `gorm:"size:512" json:"description"`
	WebhookToken string          `gorm:"size:64;index" json:"-"`
	CreatedAt    time.Time       `json:"createdAt"`
	UpdatedAt    time.Time       `json:"updatedAt"`
	Steps        []PipelineStep  `json:"steps,omitempty"`
	Runs         []PipelineRun   `json:"runs,omitempty"`
}

// PipelineStep 流水线步骤：可引用已有「保存请求」(SavedRequestID)，
// 也可内联定义 (Method/URL/Headers/Body)。两者互斥——引用模式下内联字段忽略。
// 断言始终以步骤自身定义为准，便于复用同一请求但按不同流水线分别校验。
type PipelineStep struct {
	ID             uint    `gorm:"primaryKey" json:"id"`
	PipelineID     uint    `gorm:"index" json:"pipelineId"`
	SortOrder      int     `json:"sortOrder"`
	Name           string  `gorm:"size:256" json:"name"`
	Enabled        bool    `json:"enabled"`
	SavedRequestID *uint   `gorm:"index" json:"savedRequestId,omitempty"`
	Method         string  `gorm:"size:16" json:"method"`
	URL            string  `gorm:"type:text" json:"url"`
	Headers        string  `gorm:"type:text" json:"headers"`
	Body           string  `gorm:"type:text" json:"body"`
	Assertions     string  `gorm:"type:text" json:"assertions"`
}

// PipelineRun 一次流水线运行记录，触发来源 manual/webhook。
type PipelineRun struct {
	ID         uint                `gorm:"primaryKey" json:"id"`
	PipelineID uint                `gorm:"index" json:"pipelineId"`
	Trigger    string              `gorm:"size:16" json:"trigger"`
	Status     string              `gorm:"size:16" json:"status"`
	StartedAt  time.Time           `json:"startedAt"`
	FinishedAt time.Time           `json:"finishedAt"`
	Summary    string              `gorm:"size:512" json:"summary"`
	Results    []PipelineStepResult `json:"results,omitempty"`
}

// PipelineStepResult 单步执行结果，含状态码、耗时、响应与逐条断言明细。
type PipelineStepResult struct {
	ID              uint   `gorm:"primaryKey" json:"id"`
	PipelineRunID   uint   `gorm:"index" json:"runId"`
	StepID          uint   `gorm:"index" json:"stepId"`
	StepName        string `gorm:"size:256" json:"stepName"`
	Status          string `gorm:"size:16" json:"status"`
	Method          string `gorm:"size:16" json:"method"`
	URL             string `gorm:"type:text" json:"url"`
	StatusCode      int    `json:"statusCode"`
	DurationMs      int64  `json:"durationMs"`
	ResponseHeaders string `gorm:"type:text" json:"responseHeaders"`
	ResponseBody    string `gorm:"type:text" json:"responseBody"`
	Error           string `gorm:"type:text" json:"error"`
	AssertionResults string `gorm:"type:text" json:"assertionResults"`
}

// Assertion 步骤断言定义，序列化为 PipelineStep.Assertions 的 JSON 数组。
// Type 取值：status / body_contains / header_equals / max_duration_ms。
type Assertion struct {
	Type     string `json:"type"`
	Expected string `json:"expected"`
	Header   string `json:"header,omitempty"`
	Invert   bool   `json:"invert,omitempty"`
}

// AssertionResult 单条断言的评估输出，序列化为 PipelineStepResult.AssertionResults。
type AssertionResult struct {
	Type     string `json:"type"`
	Expected string `json:"expected"`
	Header   string `json:"header,omitempty"`
	Actual   string `json:"actual"`
	Passed   bool   `json:"passed"`
}
