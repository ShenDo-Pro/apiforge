package service

import (
	"apiforge/backend/internal/model"
	"gorm.io/gorm"
)

// RequestService 管理保存的请求与发送历史。
type RequestService struct {
	db *gorm.DB
}

func NewRequestService(db *gorm.DB) *RequestService {
	return &RequestService{db: db}
}

// Save 在集合下保存（新增）一条请求。
func (s *RequestService) Save(collectionID uint, protocol, name, method, url, headers, body, preScript, testScript, extractRules, auth string) (*model.SavedRequest, error) {
	r := &model.SavedRequest{
		CollectionID:     collectionID,
		Protocol:         protocol,
		Name:             name,
		Method:           method,
		URL:              url,
		Headers:          headers,
		Body:             body,
		PreRequestScript: preScript,
		TestScript:       testScript,
		ExtractRules:     extractRules,
		Auth:             auth,
	}
	if err := s.db.Create(r).Error; err != nil {
		return nil, err
	}
	return r, nil
}

// ListByCollection 返回集合下的全部保存请求。
func (s *RequestService) ListByCollection(collectionID uint) ([]model.SavedRequest, error) {
	var rs []model.SavedRequest
	err := s.db.Where("collection_id = ?", collectionID).Order("id desc").Find(&rs).Error
	return rs, err
}

func (s *RequestService) Get(id uint) (*model.SavedRequest, error) {
	var r model.SavedRequest
	if err := s.db.First(&r, id).Error; err != nil {
		return nil, err
	}
	return &r, nil
}

// ListAllByProject 返回项目下全部保存请求（跨集合），供流水线步骤引用选择。
func (s *RequestService) ListAllByProject(projectID uint) ([]model.SavedRequest, error) {
	var rs []model.SavedRequest
	err := s.db.
		Joins("JOIN collections ON collections.id = saved_requests.collection_id").
		Where("collections.project_id = ?", projectID).
		Order("saved_requests.id desc").
		Find(&rs).Error
	return rs, err
}

func (s *RequestService) Update(id uint, protocol, name, method, url, headers, body, preScript, testScript, extractRules, auth string) error {
	return s.db.Model(&model.SavedRequest{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"protocol":         protocol,
			"name":             name, "method": method, "url": url,
			"headers": headers, "body": body,
			"pre_request_script": preScript,
			"test_script":       testScript,
			"extract_rules":     extractRules,
			"auth":              auth,
		}).Error
}

func (s *RequestService) Delete(id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("saved_request_id = ?", id).Delete(&model.RequestHistory{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.SavedRequest{}, id).Error
	})
}

// AddHistory 追加一条发送历史快照，供回看与回填。
func (s *RequestService) AddHistory(savedRequestID uint, h *model.RequestHistory) error {
	h.SavedRequestID = savedRequestID
	return s.db.Create(h).Error
}

// ListHistory 返回某请求的历史，按时间倒序。
func (s *RequestService) ListHistory(savedRequestID uint) ([]model.RequestHistory, error) {
	var hs []model.RequestHistory
	err := s.db.Where("saved_request_id = ?", savedRequestID).
		Order("created_at desc").Limit(50).Find(&hs).Error
	return hs, err
}
