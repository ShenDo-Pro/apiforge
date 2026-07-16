package database

import (
	"apiforge/backend/internal/model"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"os"

	gormpostgres "gorm.io/driver/postgres"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormsqlite "github.com/glebarez/sqlite"
)

// Open 按 driver 选择方言初始化 GORM，并打开对应数据库。
// sqlite 走纯 Go 驱动（无 cgo 依赖）；pg/mysql 使用官方 GORM 驱动，便于后续切换。
func Open(driver, dsn string) (*gorm.DB, error) {
	switch driver {
	case "pg", "postgres":
		return gorm.Open(gormpostgres.Open(dsn), &gorm.Config{})
	case "mysql":
		return gorm.Open(gormmysql.Open(dsn), &gorm.Config{})
	default:
		// sqlite 默认：确保目录存在，避免首次启动因目录缺失而失败
		if dir := dirOf(dsn); dir != "" {
			_ = os.MkdirAll(dir, 0o755)
		}
		return gorm.Open(gormsqlite.Open(dsn), &gorm.Config{})
	}
}

// Migrate 自动建表，保持模型与库结构一致。
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.User{},
		&model.Project{},
		&model.ProjectMember{},
		&model.Collection{},
		&model.SavedRequest{},
		&model.Environment{},
		&model.RequestHistory{},
		&model.Pipeline{},
		&model.PipelineStep{},
		&model.PipelineRun{},
		&model.PipelineStepResult{},
		&model.AuditLog{},
	)
}

func dirOf(dsn string) string {
	for i := len(dsn) - 1; i >= 0; i-- {
		if dsn[i] == '/' || dsn[i] == '\\' {
			return dsn[:i]
		}
	}
	return ""
}

// SeedAdmin 在库无用户时写入默认管理员，方便首次启动登录。
// 密码为空时生成随机强口令并记录到日志（不再硬编码弱口令 admin123），
// 强烈建议部署时通过 APIFORGE_ADMIN_PASSWORD 环境变量指定（H6）。
func SeedAdmin(db *gorm.DB, username, password string) {
	var count int64
	if err := db.Model(&model.User{}).Count(&count).Error; err != nil {
		slog.Error("统计用户失败", "err", err)
		return
	}
	if count > 0 {
		return
	}
	if password == "" {
		b := make([]byte, 16)
		if _, err := rand.Read(b); err != nil {
			slog.Error("生成随机管理员密码失败", "err", err)
			return
		}
		password = hex.EncodeToString(b)
		slog.Warn("已创建默认管理员，未设置 APIFORGE_ADMIN_PASSWORD，临时密码见下，请尽快通过界面修改", "username", username, "password", password)
	}
	u := &model.User{Username: username, Role: "admin", NeedReset: true}
	if err := u.SetPassword(password); err != nil {
		slog.Error("设置默认管理员密码失败", "err", err)
		return
	}
	if err := db.Create(u).Error; err != nil {
		slog.Error("创建默认管理员失败", "err", err)
		return
	}
	slog.Info("已创建默认管理员账号", "username", username)
}
