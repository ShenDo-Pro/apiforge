package service

import (
	"errors"

	"apiforge/backend/internal/model"
	"gorm.io/gorm"
)

// ErrForbidden 表示资源不属于当前项目（越权访问）。
var ErrForbidden = errors.New("forbidden: resource does not belong to the project")

// CollectionService 管理请求集合的树形结构（支持嵌套文件夹）。
type CollectionService struct {
	db *gorm.DB
}

func NewCollectionService(db *gorm.DB) *CollectionService {
	return &CollectionService{db: db}
}

// Create 新建集合，可指定父节点形成层级。
func (s *CollectionService) Create(projectID uint, parentID *uint, name string, sortOrder int) (*model.Collection, error) {
	c := &model.Collection{
		ProjectID: projectID,
		ParentID:  parentID,
		Name:      name,
		SortOrder: sortOrder,
	}
	if err := s.db.Create(c).Error; err != nil {
		return nil, err
	}
	return c, nil
}

// List 返回项目下全部集合（扁平），由前端按 parentID 组装成树。
func (s *CollectionService) List(projectID uint) ([]model.Collection, error) {
	var cs []model.Collection
	err := s.db.Where("project_id = ?", projectID).Order("sort_order asc, id asc").Find(&cs).Error
	return cs, err
}

func (s *CollectionService) Update(id, projectID uint, name string, sortOrder int, variables string) error {
	var c model.Collection
	if err := s.db.First(&c, id).Error; err != nil {
		return err
	}
	// 归属校验：集合必须属于当前项目，防止越权改写他人项目资源（H1）
	if c.ProjectID != projectID {
		return ErrForbidden
	}
	updates := map[string]interface{}{"name": name, "sort_order": sortOrder}
	// variables 为空字符串时表示调用方未改动该字段，避免清空已有变量。
	if variables != "" {
		updates["variables"] = variables
	}
	return s.db.Model(&model.Collection{}).Where("id = ?", id).Updates(updates).Error
}

// Delete 递归删除集合及其子集合、请求，避免残留孤立数据。
// projectID 用于逐层校验归属，杜绝跨项目删除（H1）。
func (s *CollectionService) Delete(id, projectID uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var c model.Collection
		if err := tx.First(&c, id).Error; err != nil {
			return err
		}
		if c.ProjectID != projectID {
			return ErrForbidden
		}
		var children []model.Collection
		if err := tx.Where("parent_id = ?", id).Find(&children).Error; err != nil {
			return err
		}
		for _, child := range children {
			if err := s.Delete(child.ID, projectID); err != nil {
				return err
			}
		}
		if err := tx.Where("collection_id = ?", id).Delete(&model.SavedRequest{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.Collection{}, id).Error
	})
}
