package service

import (
	"encoding/json"

	"apiforge/backend/internal/model"
	"gorm.io/gorm"
)

// EnvironmentService 管理环境与全局变量的 CRUD，环境挂在项目下、多人共享。
type EnvironmentService struct {
	db *gorm.DB
}

func NewEnvironmentService(db *gorm.DB) *EnvironmentService {
	return &EnvironmentService{db: db}
}

// List 返回项目下全部环境（含 global 单例），按 SortOrder 升序。
func (s *EnvironmentService) List(projectID uint) ([]model.Environment, error) {
	var envs []model.Environment
	err := s.db.Where("project_id = ?", projectID).
		Order("sort_order asc, id asc").
		Find(&envs).Error
	return envs, err
}

// GetGlobal 取项目全局变量单例；不存在则创建空单例，保证每项目仅一行。
func (s *EnvironmentService) GetGlobal(projectID uint) (*model.Environment, error) {
	var g model.Environment
	err := s.db.Where("project_id = ? AND kind = ?", projectID, "global").First(&g).Error
	if err == gorm.ErrRecordNotFound {
		g = model.Environment{ProjectID: projectID, Kind: "global", Name: "Globals", Values: "[]"}
		if cerr := s.db.Create(&g).Error; cerr != nil {
			return nil, cerr
		}
		return &g, nil
	}
	if err != nil {
		return nil, err
	}
	return &g, nil
}

// UpsertGlobal 覆盖写入全局变量（前端增量编辑后整体提交），保证每项目仅一行。
func (s *EnvironmentService) UpsertGlobal(projectID uint, values []model.EnvVar) error {
	b, err := json.Marshal(values)
	if err != nil {
		return err
	}
	g, err := s.GetGlobal(projectID)
	if err != nil {
		return err
	}
	return s.db.Model(&model.Environment{}).Where("id = ?", g.ID).
		Update("values", string(b)).Error
}

// Create 新建普通环境。
func (s *EnvironmentService) Create(projectID uint, name string, values []model.EnvVar) (*model.Environment, error) {
	b, _ := json.Marshal(values)
	env := &model.Environment{ProjectID: projectID, Kind: "env", Name: name, Values: string(b), SortOrder: 0}
	if err := s.db.Create(env).Error; err != nil {
		return nil, err
	}
	return env, nil
}

// Find 取单条环境（含 global），并校验归属当前项目（H1）。
func (s *EnvironmentService) Find(id, projectID uint) (*model.Environment, error) {
	var env model.Environment
	if err := s.db.First(&env, id).Error; err != nil {
		return nil, err
	}
	if env.ProjectID != projectID {
		return nil, ErrForbidden
	}
	return &env, nil
}

// Update 修改环境名称与变量（values 整体覆盖），并校验归属（H1）。
func (s *EnvironmentService) Update(id, projectID uint, name string, values []model.EnvVar) error {
	var env model.Environment
	if err := s.db.First(&env, id).Error; err != nil {
		return err
	}
	if env.ProjectID != projectID {
		return ErrForbidden
	}
	b, err := json.Marshal(values)
	if err != nil {
		return err
	}
	return s.db.Model(&model.Environment{}).Where("id = ?", id).
		Updates(map[string]interface{}{"name": name, "values": string(b)}).Error
}

// Reorder 按给定 id 顺序重排 SortOrder（拖拽后提交）。
func (s *EnvironmentService) Reorder(ids []uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		for i, id := range ids {
			if err := tx.Model(&model.Environment{}).Where("id = ?", id).
				Update("sort_order", i).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// Delete 删除环境；global 单例不允许删除（调用方应避免），此处直接忽略。
// projectID 用于校验归属，防止跨项目删除（H1）。
func (s *EnvironmentService) Delete(id, projectID uint) error {
	var env model.Environment
	if err := s.db.First(&env, id).Error; err != nil {
		return err
	}
	if env.ProjectID != projectID {
		return ErrForbidden
	}
	if env.Kind == "global" {
		return nil
	}
	return s.db.Delete(&env).Error
}
