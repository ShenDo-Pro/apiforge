package model

import (
	"golang.org/x/crypto/bcrypt"
	"time"
)

// User 系统用户。Role 为全局角色：admin 拥有系统全部权限，user 依赖项目授权。
type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Username  string    `gorm:"uniqueIndex;size:64" json:"username"`
	Password  string    `gorm:"size:255" json:"-"` // bcrypt 哈希，绝不外泄
	Role      string    `gorm:"size:16;default:user" json:"role"`
	CreatedAt time.Time `json:"createdAt"`
}

// SetPassword 用 bcrypt 计算哈希并写入，cost=10 兼顾安全与性能。
func (u *User) SetPassword(plain string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hash)
	return nil
}

// CheckPassword 校验明文密码是否匹配哈希。
func (u *User) CheckPassword(plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(plain)) == nil
}
