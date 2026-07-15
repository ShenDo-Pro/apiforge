package service

import (
	"encoding/json"
	"errors"

	"apiforge/backend/internal/model"
	"gorm.io/gorm"
)

// ProjectService 管理项目与项目成员的授权关系。
type ProjectService struct {
	db *gorm.DB
}

func NewProjectService(db *gorm.DB) *ProjectService {
	return &ProjectService{db: db}
}

// Create 新建项目，并自动将创建者登记为 owner 成员，保证创建者初始全权限。
func (s *ProjectService) Create(ownerID uint, name, description string) (*model.Project, error) {
	p := &model.Project{Name: name, Description: description, OwnerID: ownerID}
	if err := s.db.Create(p).Error; err != nil {
		return nil, err
	}
	member := model.ProjectMember{
		ProjectID:   p.ID,
		UserID:      ownerID,
		Role:        "owner",
		Permissions: `{"add":true,"edit":true,"delete":true}`,
	}
	if err := s.db.Create(&member).Error; err != nil {
		return nil, err
	}
	return p, nil
}

// ListForUser 返回用户可见项目：自己创建的 + 作为成员加入的（去重）。
func (s *ProjectService) ListForUser(userID uint) ([]model.Project, error) {
	var projects []model.Project
	err := s.db.
		Where("owner_id = ?", userID).
		Or("id IN (?)", s.db.Model(&model.ProjectMember{}).Select("project_id").Where("user_id = ?", userID)).
		Order("created_at desc").
		Find(&projects).Error
	return projects, err
}

func (s *ProjectService) Get(id uint) (*model.Project, error) {
	var p model.Project
	if err := s.db.First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *ProjectService) Update(id uint, name, description string) error {
	return s.db.Model(&model.Project{}).Where("id = ?", id).
		Updates(map[string]interface{}{"name": name, "description": description}).Error
}

func (s *ProjectService) Delete(id uint) error {
	// 级联删除成员与集合，保持数据整洁
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("project_id = ?", id).Delete(&model.ProjectMember{}).Error; err != nil {
			return err
		}
		if err := tx.Where("project_id = ?", id).Delete(&model.Collection{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.Project{}, id).Error
	})
}

// MemberView 是成员列表的返回结构，额外带上成员用户名便于前端检索展示。
type MemberView struct {
	ID          uint   `json:"id"`
	UserID      uint   `json:"userId"`
	Username    string `json:"username"`
	Role        string `json:"role"`
	Permissions string `json:"permissions"`
}

// ListMembers 返回项目成员清单，关联 users 表取回用户名。
func (s *ProjectService) ListMembers(projectID uint) ([]MemberView, error) {
	var ms []MemberView
	err := s.db.
		Table("project_members").
		Select("project_members.id, project_members.user_id, project_members.role, project_members.permissions, users.username as username").
		Joins("left join users on users.id = project_members.user_id").
		Where("project_members.project_id = ?", projectID).
		Order("project_members.id").
		Find(&ms).Error
	return ms, err
}

// AddMember 邀请用户加入项目并设定角色与权限。
func (s *ProjectService) AddMember(projectID, userID uint, role string, perms map[string]bool) error {
	if role == "" {
		role = "developer"
	}
	if perms == nil {
		perms = map[string]bool{"add": true, "edit": true, "delete": true}
	}
	permJSON, _ := json.Marshal(perms)
	m := model.ProjectMember{
		ProjectID:   projectID,
		UserID:      userID,
		Role:        role,
		Permissions: string(permJSON),
	}
	// 重复加入则忽略，避免唯一冲突
	var cnt int64
	s.db.Model(&model.ProjectMember{}).Where("project_id = ? AND user_id = ?", projectID, userID).Count(&cnt)
	if cnt > 0 {
		return errors.New("member already exists")
	}
	return s.db.Create(&m).Error
}

func (s *ProjectService) UpdateMember(projectID, userID uint, role string, perms map[string]bool) error {
	permJSON, _ := json.Marshal(perms)
	return s.db.Model(&model.ProjectMember{}).
		Where("project_id = ? AND user_id = ?", projectID, userID).
		Updates(map[string]interface{}{"role": role, "permissions": string(permJSON)}).Error
}

func (s *ProjectService) RemoveMember(projectID, userID uint) error {
	return s.db.Where("project_id = ? AND user_id = ?", projectID, userID).
		Delete(&model.ProjectMember{}).Error
}
