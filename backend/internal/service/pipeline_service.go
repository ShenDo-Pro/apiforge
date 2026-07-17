package service

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"time"

	"apitoolx/backend/internal/model"
	"apitoolx/backend/internal/proxy"
	"gorm.io/gorm"
)

// PipelineService 管理测试流水线的 CRUD、Webhook token 与顺序执行。
type PipelineService struct {
	db      *gorm.DB
	maxBody int64
}

func NewPipelineService(db *gorm.DB, maxBody int64) *PipelineService {
	return &PipelineService{db: db, maxBody: maxBody}
}

// genToken 生成 32 字节随机十六进制 token，用作 Webhook 触发密钥。
func genToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// List 返回项目下全部流水线（不含步骤/运行）。
func (s *PipelineService) List(projectID uint) ([]model.Pipeline, error) {
	var ps []model.Pipeline
	err := s.db.Where("project_id = ?", projectID).Order("id desc").Find(&ps).Error
	return ps, err
}

// Get 返回流水线及其有序步骤与运行历史。
func (s *PipelineService) Get(id uint) (*model.Pipeline, error) {
	var p model.Pipeline
	if err := s.db.Preload("Steps").Preload("Runs").First(&p, id).Error; err != nil {
		return nil, err
	}
	sort.Slice(p.Steps, func(i, j int) bool { return p.Steps[i].SortOrder < p.Steps[j].SortOrder })
	return &p, nil
}

type PipelineCreateReq struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Create 新建流水线并分配 Webhook token。
func (s *PipelineService) Create(projectID uint, in PipelineCreateReq) (*model.Pipeline, error) {
	p := &model.Pipeline{
		ProjectID:    projectID,
		Name:         in.Name,
		Description:  in.Description,
		WebhookToken: genToken(),
	}
	if err := s.db.Create(p).Error; err != nil {
		return nil, err
	}
	return p, nil
}

type PipelineUpdateReq struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Steps       []StepInput `json:"steps"`
}

// StepInput 是步骤的入站结构，支持引用已有请求或内联定义。
type StepInput struct {
	ID             uint    `json:"id"`
	SavedRequestID *uint   `json:"savedRequestId"`
	Name           string  `json:"name"`
	Enabled        bool    `json:"enabled"`
	Method         string  `json:"method"`
	URL            string  `json:"url"`
	Headers        string  `json:"headers"`
	Body           string  `json:"body"`
	Assertions     string  `json:"assertions"`
}

// Update 更新流水线元信息与全部步骤（全量替换步骤）。
func (s *PipelineService) Update(id uint, in PipelineUpdateReq) (*model.Pipeline, error) {
	var p model.Pipeline
	if err := s.db.First(&p, id).Error; err != nil {
		return nil, err
	}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&p).Updates(map[string]interface{}{
			"name":        in.Name,
			"description": in.Description,
		}).Error; err != nil {
			return err
		}
		// 增量更新步骤：按 ID 复用已有步骤（保留历史 StepResult 关联），
		// 新建无 ID 的步骤，删除客户端不再提交的旧步骤（M17）。
		var existing []model.PipelineStep
		if err := tx.Where("pipeline_id = ?", id).Find(&existing).Error; err != nil {
			return err
		}
		existingByID := make(map[uint]model.PipelineStep, len(existing))
		for _, e := range existing {
			existingByID[e.ID] = e
		}
		kept := make(map[uint]bool, len(in.Steps))
		for i, st := range in.Steps {
			step := model.PipelineStep{
				PipelineID:     id,
				SortOrder:      i,
				Name:           st.Name,
				Enabled:        st.Enabled,
				SavedRequestID: st.SavedRequestID,
				Method:         st.Method,
				URL:            st.URL,
				Headers:        st.Headers,
				Body:           st.Body,
				Assertions:     st.Assertions,
			}
			if st.ID > 0 {
				if _, ok := existingByID[st.ID]; ok {
					step.ID = st.ID
					kept[st.ID] = true
					if err := tx.Model(&model.PipelineStep{}).Where("id = ?", st.ID).Updates(step).Error; err != nil {
						return err
					}
					continue
				}
			}
			if err := tx.Create(&step).Error; err != nil {
				return err
			}
		}
		for _, e := range existing {
			if !kept[e.ID] {
				if err := tx.Delete(&e).Error; err != nil {
					return err
				}
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return s.Get(id)
}

// Delete 级联删除流水线及其步骤、运行与逐步结果。
func (s *PipelineService) Delete(id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var runs []model.PipelineRun
		if err := tx.Where("pipeline_id = ?", id).Find(&runs).Error; err != nil {
			return err
		}
		for _, run := range runs {
			if err := tx.Where("pipeline_run_id = ?", run.ID).Delete(&model.PipelineStepResult{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("pipeline_id = ?", id).Delete(&model.PipelineRun{}).Error; err != nil {
			return err
		}
		if err := tx.Where("pipeline_id = ?", id).Delete(&model.PipelineStep{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.Pipeline{}, id).Error
	})
}

// RegenerateToken 重置 Webhook token，旧 URL 立即失效。
func (s *PipelineService) RegenerateToken(id uint) (string, error) {
	token := genToken()
	if err := s.db.Model(&model.Pipeline{}).Where("id = ?", id).
		Update("webhook_token", token).Error; err != nil {
		return "", err
	}
	return token, nil
}

// FindByToken 按 Webhook token 定位流水线（免鉴权触发用）。
func (s *PipelineService) FindByToken(token string) (*model.Pipeline, error) {
	var p model.Pipeline
	if err := s.db.Preload("Steps").Where("webhook_token = ?", token).First(&p).Error; err != nil {
		return nil, err
	}
	sort.Slice(p.Steps, func(i, j int) bool { return p.Steps[i].SortOrder < p.Steps[j].SortOrder })
	return &p, nil
}

// Run 顺序执行流水线步骤并落库运行记录，返回完整运行结果。
// trigger 取值 manual / webhook，共用同一执行路径。
func (s *PipelineService) Run(pipelineID uint, trigger string) (*model.PipelineRun, error) {
	p, err := s.Get(pipelineID)
	if err != nil {
		return nil, err
	}

	run := &model.PipelineRun{
		PipelineID: p.ID,
		Trigger:    trigger,
		Status:     "running",
		StartedAt:  time.Now(),
	}
	if err := s.db.Create(run).Error; err != nil {
		return nil, err
	}

	passed := 0
	failed := 0
	for _, step := range p.Steps {
		if !step.Enabled {
			continue
		}
		result := s.execStep(p.ID, run.ID, &step)
		if result.Status == "passed" {
			passed++
		} else {
			failed++
		}
		if err := s.db.Create(result).Error; err != nil {
			slog.Error("写入库步骤结果失败", "run", run.ID, "step", step.ID, "err", err)
		}
	}

	run.FinishedAt = time.Now()
	if failed > 0 {
		run.Status = "failed"
	} else {
		run.Status = "passed"
	}
	run.Summary = strconv.Itoa(passed+failed) + " steps, " + strconv.Itoa(failed) + " failed"
	if err := s.db.Model(run).Updates(map[string]interface{}{
		"status":      run.Status,
		"finished_at": run.FinishedAt,
		"summary":     run.Summary,
	}).Error; err != nil {
		slog.Error("更新流水线运行状态失败", "run", run.ID, "err", err)
	}

	// 重新载入结果，便于返回完整数据
	full, _ := s.GetRun(run.ID)
	return full, nil
}

// GetRun 返回单次运行及其逐步结果。
func (s *PipelineService) GetRun(runID uint) (*model.PipelineRun, error) {
	var run model.PipelineRun
	if err := s.db.Preload("Results").First(&run, runID).Error; err != nil {
		return nil, err
	}
	return &run, nil
}

// ListRuns 返回流水线的运行历史，按时间倒序。
func (s *PipelineService) ListRuns(pipelineID uint) ([]model.PipelineRun, error) {
	var runs []model.PipelineRun
	err := s.db.Where("pipeline_id = ?", pipelineID).Order("id desc").Find(&runs).Error
	return runs, err
}

// execStep 解析步骤来源（引用/内联），执行 HTTP 请求并评估断言，返回结果记录。
func (s *PipelineService) execStep(projectID, runID uint, step *model.PipelineStep) *model.PipelineStepResult {
	res := &model.PipelineStepResult{
		PipelineRunID: runID,
		StepID:        step.ID,
		StepName:      step.Name,
	}

	method, url, headers, body, err := s.resolveStepSource(projectID, step)
	res.Method = method
	res.URL = url

	if err != nil {
		res.Status = "error"
		res.Error = err.Error()
		return res
	}

	// 解析请求头 JSON
	hdr := map[string]string{}
	if headers != "" {
		if e := json.Unmarshal([]byte(headers), &hdr); e != nil {
			res.Status = "error"
			res.Error = "invalid headers json: " + e.Error()
			return res
		}
	}

	preq := &proxy.ProxyRequest{Method: method, URL: url, Headers: hdr, Body: body}
	presp := proxy.Do(preq, s.maxBody)
	if presp.Error != "" {
		res.Status = "error"
		res.Error = presp.Error
		return res
	}

	res.StatusCode = presp.Status
	res.DurationMs = presp.Timings.Total
	if hb, e := json.Marshal(presp.Headers); e == nil {
		res.ResponseHeaders = string(hb)
	}
	// 截断过长的响应体，避免落库膨胀（与 maxBody 一致）
	if len(presp.Body) > int(s.maxBody) {
		res.ResponseBody = presp.Body[:s.maxBody]
	} else {
		res.ResponseBody = presp.Body
	}

	results := s.evaluateAssertions(step.Assertions, presp)
	if ab, e := json.Marshal(results); e == nil {
		res.AssertionResults = string(ab)
	}

	ok := true
	for _, a := range results {
		if !a.Passed {
			ok = false
			break
		}
	}
	if ok {
		res.Status = "passed"
	} else {
		res.Status = "failed"
	}
	return res
}

// resolveStepSource 返回步骤实际要执行的请求定义。
// 引用模式下加载 SavedRequest，并校验其归属当前项目，防止跨租户执行他人请求（M9）；
// 内联字段忽略；否则使用内联定义。
func (s *PipelineService) resolveStepSource(projectID uint, step *model.PipelineStep) (method, url, headers, body string, err error) {
	if step.SavedRequestID != nil {
		var sr model.SavedRequest
		// 通过集合关联校验请求归属当前项目，避免裸 ID 越权引用（M9）
		if e := s.db.
			Joins("JOIN collections ON collections.id = saved_requests.collection_id").
			Where("saved_requests.id = ? AND collections.project_id = ?", *step.SavedRequestID, projectID).
			First(&sr).Error; e != nil {
			return "", "", "", "", e
		}
		return sr.Method, sr.URL, sr.Headers, sr.Body, nil
	}
	return step.Method, step.URL, step.Headers, step.Body, nil
}

// evaluateAssertions 根据断言定义评估响应，逐条产出通过/失败结果。
func (s *PipelineService) evaluateAssertions(raw string, presp *proxy.ProxyResponse) []model.AssertionResult {
	out := []model.AssertionResult{}
	if raw == "" {
		return out
	}
	var asserts []model.Assertion
	if err := json.Unmarshal([]byte(raw), &asserts); err != nil {
		out = append(out, model.AssertionResult{Type: "parse", Actual: err.Error(), Passed: false})
		return out
	}
	for _, a := range asserts {
		r := model.AssertionResult{Type: a.Type, Expected: a.Expected, Header: a.Header}
		switch a.Type {
		case "status":
			r.Actual = strconv.Itoa(presp.Status)
			want, e := strconv.Atoi(a.Expected)
			pass := e == nil && presp.Status == want
			if a.Invert {
				pass = !pass
			}
			r.Passed = pass
		case "body_contains":
			r.Actual = "len=" + strconv.Itoa(len(presp.Body))
			pass := strings.Contains(presp.Body, a.Expected)
			if a.Invert {
				pass = !pass
			}
			r.Passed = pass
		case "header_equals":
			got := presp.Headers[a.Header]
			r.Actual = got
			pass := got == a.Expected
			if a.Invert {
				pass = !pass
			}
			r.Passed = pass
		case "max_duration_ms":
			r.Actual = strconv.FormatInt(presp.Timings.Total, 10)
			want, e := strconv.ParseInt(a.Expected, 10, 64)
			// 阈值非法视为断言失败（不再静默通过），避免无效断言定义被误判通过（M19）
			r.Passed = e == nil && presp.Timings.Total <= want
		default:
			r.Actual = "unknown assertion type"
			r.Passed = false
		}
		out = append(out, r)
	}
	return out
}
