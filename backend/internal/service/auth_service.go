package service

import (
	"errors"

	"apiforge/backend/internal/middleware"
	"apiforge/backend/internal/model"
	"apiforge/backend/pkg/config"
	"gorm.io/gorm"
)

// AuthService 处理登录、注册与 token 刷新。
type AuthService struct {
	db  *gorm.DB
	cfg *config.JWTConfig
}

func NewAuthService(db *gorm.DB, cfg *config.JWTConfig) *AuthService {
	return &AuthService{db: db, cfg: cfg}
}

var ErrInvalidCredential = errors.New("invalid username or password")

// Login 校验密码并签发双 token。
func (s *AuthService) Login(username, password string) (*model.User, string, string, error) {
	var u model.User
	if err := s.db.Where("username = ?", username).First(&u).Error; err != nil {
		return nil, "", "", ErrInvalidCredential
	}
	if !u.CheckPassword(password) {
		return nil, "", "", ErrInvalidCredential
	}
	access, refresh, err := middleware.GenTokens(s.cfg, u.ID, u.Username, u.Role)
	if err != nil {
		return nil, "", "", err
	}
	return &u, access, refresh, nil
}

// Register 创建普通用户。系统默认首账号为 admin，其余注册均为普通 user。
func (s *AuthService) Register(username, password string) (*model.User, error) {
	var cnt int64
	s.db.Model(&model.User{}).Where("username = ?", username).Count(&cnt)
	if cnt > 0 {
		return nil, errors.New("username already exists")
	}
	u := &model.User{Username: username, Role: "user"}
	if err := u.SetPassword(password); err != nil {
		return nil, err
	}
	if err := s.db.Create(u).Error; err != nil {
		return nil, err
	}
	return u, nil
}

// Refresh 用 refresh token 换发新的 access token。
func (s *AuthService) Refresh(refreshToken string) (string, error) {
	claims := &middleware.Claims{}
	token, err := middleware.ParseToken(refreshToken, s.cfg.Secret, claims)
	if err != nil || !token.Valid {
		return "", errors.New("invalid refresh token")
	}
	access, _, err := middleware.GenTokens(s.cfg, claims.UserID, claims.Username, claims.Role)
	if err != nil {
		return "", err
	}
	return access, nil
}
