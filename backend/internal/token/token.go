// Package token 负责 JWT 的签发与解析，从 middleware 剥离以避免 service 层反向依赖传输层（M3）。
package token

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"apiforge/backend/pkg/config"
	"github.com/golang-jwt/jwt/v5"
)

// Claims 是 JWT 载荷，携带用户身份与全局角色。
type Claims struct {
	UserID   uint   `json:"uid"`
	Username string `json:"uname"`
	Role     string `json:"role"`
	// JTI 唯一标识，用于注销时把该 refresh token 加入黑名单（M2）。
	JTI string `json:"jti"`
	jwt.RegisteredClaims
}

// 注销黑名单：记录被吊销的 refresh token JTI 及其失效时间，过期后自动失效（M2）。
var (
	blacklistMu sync.Mutex
	blacklist   = map[string]time.Time{}
)

// Revoke 将 jti 加入黑名单，直到 expireAt 之前均视为已注销。
func Revoke(jti string, expireAt time.Time) {
	if jti == "" {
		return
	}
	blacklistMu.Lock()
	blacklist[jti] = expireAt
	blacklistMu.Unlock()
}

// IsRevoked 判断 jti 是否已被注销（M2）。
func IsRevoked(jti string) bool {
	if jti == "" {
		return false
	}
	blacklistMu.Lock()
	defer blacklistMu.Unlock()
	exp, ok := blacklist[jti]
	if !ok {
		return false
	}
	if time.Now().After(exp) {
		// 已过期，顺手清理
		delete(blacklist, jti)
		return false
	}
	return true
}

// newJTI 生成随机 JTI。
func newJTI() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return hex.EncodeToString([]byte(time.Now().Format("150405.000000000")))
	}
	return hex.EncodeToString(b)
}

// ParseToken 解析并校验 token，secret 必须与服务端签发一致。
func ParseToken(tokenStr, secret string, claims *Claims) (*jwt.Token, error) {
	return jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		// 固定使用 HMAC，避免算法混淆攻击
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(secret), nil
	})
}

// GenTokens 生成 access/refresh 双 token，refresh 长有效期用于静默续期。
func GenTokens(cfg *config.JWTConfig, userID uint, username, role string) (access, refresh string, err error) {
	now := time.Now()
	accessClaims := Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		JTI:      newJTI(),
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(cfg.AccessTTLMinutes) * time.Minute)),
		},
	}
	refreshClaims := Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		JTI:      newJTI(),
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(cfg.RefreshTTLHours) * time.Hour)),
		},
	}
	secret := []byte(cfg.Secret)
	a, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString(secret)
	if err != nil {
		return "", "", err
	}
	r, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString(secret)
	if err != nil {
		return "", "", err
	}
	return a, r, nil
}
