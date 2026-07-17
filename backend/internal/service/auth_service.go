package service

import (
	"errors"

	"apitoolx/backend/internal/model"
	"apitoolx/backend/internal/token"
	"apitoolx/backend/pkg/config"
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
	access, refresh, err := token.GenTokens(s.cfg, u.ID, u.Username, u.Role)
	if err != nil {
		return nil, "", "", err
	}
	return &u, access, refresh, nil
}

// Register 创建普通用户。系统默认首账号为 admin，其余注册均为普通 user。
// 在事务内先计数再创建，避免并发竞态产生重复用户（M4）；计数错误不再被吞掉。
func (s *AuthService) Register(username, password string) (*model.User, error) {
	u := &model.User{Username: username, Role: "user"}
	if err := u.SetPassword(password); err != nil {
		return nil, err
	}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var cnt int64
		if e := tx.Model(&model.User{}).Where("username = ?", username).Count(&cnt).Error; e != nil {
			return e
		}
		if cnt > 0 {
			return errors.New("username already exists")
		}
		return tx.Create(u).Error
	})
	if err != nil {
		return nil, err
	}
	return u, nil
}

// ResetPassword 修改当前登录用户密码（首次强制改密或主动修改）。
// 校验旧口令匹配后更新哈希并清除 NeedReset 标记（H6）。
func (s *AuthService) ResetPassword(userID uint, oldPwd, newPwd string) error {
	if len(newPwd) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	var u model.User
	if err := s.db.First(&u, userID).Error; err != nil {
		return errors.New("user not found")
	}
	if !u.CheckPassword(oldPwd) {
		return errors.New("invalid old password")
	}
	if err := u.SetPassword(newPwd); err != nil {
		return err
	}
	upd := map[string]interface{}{"password": u.Password, "need_reset": false}
	if err := s.db.Model(&model.User{}).Where("id = ?", userID).Updates(upd).Error; err != nil {
		return err
	}
	return nil
}

// Refresh 用 refresh token 换发新的 access token。
func (s *AuthService) Refresh(refreshToken string) (string, error) {
	claims := &token.Claims{}
	parsed, err := token.ParseToken(refreshToken, s.cfg.Secret, claims)
	if err != nil || !parsed.Valid {
		return "", errors.New("invalid refresh token")
	}
	// 注销黑名单校验：被吊销的 refresh token 不可再续期（M2）
	if token.IsRevoked(claims.JTI) {
		return "", errors.New("refresh token revoked")
	}
	access, _, err := token.GenTokens(s.cfg, claims.UserID, claims.Username, claims.Role)
	if err != nil {
		return "", err
	}
	return access, nil
}

// Logout 注销当前 refresh token：将其 JTI 加入黑名单直至原有效期，
// 使该 refresh token 立即失效（M2）。access token 仍按自身短有效期自然过期。
func (s *AuthService) Logout(refreshToken string) error {
	claims := &token.Claims{}
	parsed, err := token.ParseToken(refreshToken, s.cfg.Secret, claims)
	if err != nil || !parsed.Valid {
		return errors.New("invalid refresh token")
	}
	if claims.ExpiresAt != nil {
		token.Revoke(claims.JTI, claims.ExpiresAt.Time)
	}
	return nil
}
