package service

import (
	"apiforge/backend/internal/model"
	"gorm.io/gorm"
)

// AuditService 提供审计日志的写入与分页查询（C9）。
type AuditService struct {
	db *gorm.DB
}

func NewAuditService(db *gorm.DB) *AuditService {
	return &AuditService{db: db}
}

// Append 写入一条审计记录。
func (s *AuditService) Append(l *model.AuditLog) error {
	return s.db.Create(l).Error
}

// List 按时间倒序分页返回审计日志。
func (s *AuditService) List(page, perPage int) ([]model.AuditLog, int64, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	var total int64
	if err := s.db.Model(&model.AuditLog{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var logs []model.AuditLog
	offset := (page - 1) * perPage
	if err := s.db.Order("created_at desc").Offset(offset).Limit(perPage).Find(&logs).Error; err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}
