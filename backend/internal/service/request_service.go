package service

import (
	"apitoolx/backend/internal/model"
	"gorm.io/gorm"
)

// belongsToProject 校验保存请求经其所属集合归属于指定项目（H1）。
func (s *RequestService) belongsToProject(id, projectID uint) (bool, error) {
	var projID uint
	if err := s.db.Table("saved_requests").
		Select("collections.project_id").
		Joins("JOIN collections ON collections.id = saved_requests.collection_id").
		Where("saved_requests.id = ?", id).
		Scan(&projID).Error; err != nil {
		return false, err
	}
	return projID == projectID, nil
}

// collectionBelongsToProject 校验集合归属于指定项目（H1，Save 场景）。
func (s *RequestService) collectionBelongsToProject(collectionID, projectID uint) (bool, error) {
	var projID uint
	if err := s.db.Table("collections").Select("project_id").Where("id = ?", collectionID).Scan(&projID).Error; err != nil {
		return false, err
	}
	return projID == projectID, nil
}

// RequestService 管理保存的请求与发送历史。
type RequestService struct {
	db *gorm.DB
}

func NewRequestService(db *gorm.DB) *RequestService {
	return &RequestService{db: db}
}

// Save 在集合下保存（新增）一条请求。
// projectID 用于归属校验：集合必须属于调用者有权限的项目，否则禁止越权写入（H1）。
func (s *RequestService) Save(projectID, collectionID uint, protocol, name, method, url, headers, body, preScript, testScript, extractRules, auth string) (*model.SavedRequest, error) {
	ok, err := s.collectionBelongsToProject(collectionID, projectID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrForbidden
	}
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
// projectID 用于归属校验：跨项目的集合直接返回空结果，杜绝越权读取（H1）。
func (s *RequestService) ListByCollection(projectID, collectionID uint) ([]model.SavedRequest, error) {
	var rs []model.SavedRequest
	err := s.db.
		Joins("JOIN collections ON collections.id = saved_requests.collection_id").
		Where("collections.project_id = ? AND saved_requests.collection_id = ?", projectID, collectionID).
		Order("saved_requests.id desc").Find(&rs).Error
	return rs, err
}

func (s *RequestService) Get(id, projectID uint) (*model.SavedRequest, error) {
	var r model.SavedRequest
	if err := s.db.First(&r, id).Error; err != nil {
		return nil, err
	}
	ok, err := s.belongsToProject(id, projectID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrForbidden
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

func (s *RequestService) Update(id, projectID uint, protocol, name, method, url, headers, body, preScript, testScript, extractRules, auth string) error {
	ok, err := s.belongsToProject(id, projectID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}
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

func (s *RequestService) Delete(id, projectID uint) error {
	ok, err := s.belongsToProject(id, projectID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("saved_request_id = ?", id).Delete(&model.RequestHistory{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.SavedRequest{}, id).Error
	})
}

// AddHistory 追加一条发送历史快照，供回看与回填。
// projectID 用于归属校验：历史只能写入调用者有权限项目的请求（H1）。
func (s *RequestService) AddHistory(savedRequestID, projectID uint, h *model.RequestHistory) error {
	ok, err := s.belongsToProject(savedRequestID, projectID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}
	h.SavedRequestID = savedRequestID
	return s.db.Create(h).Error
}

// ListHistory 返回某请求的历史，按时间倒序。
// projectID 用于归属校验：跨项目的请求历史直接返回空（H1）。
func (s *RequestService) ListHistory(savedRequestID, projectID uint) ([]model.RequestHistory, error) {
	ok, err := s.belongsToProject(savedRequestID, projectID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrForbidden
	}
	var hs []model.RequestHistory
	err = s.db.Where("saved_request_id = ?", savedRequestID).
		Order("created_at desc").Limit(50).Find(&hs).Error
	return hs, err
}
