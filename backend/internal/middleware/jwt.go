package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"apiforge/backend/pkg/config"
	"apiforge/backend/pkg/response"
	"github.com/golang-jwt/jwt/v5"
)

// contextKey 用独立类型避免与其他包键冲突。
type contextKey string

const userKey contextKey = "claims"

// Claims 是 JWT 载荷，携带用户身份与全局角色。
type Claims struct {
	UserID   uint   `json:"uid"`
	Username string `json:"uname"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// ContextUser 从请求上下文取出已认证的 Claims。
func ContextUser(r *http.Request) *Claims {
	if c, ok := r.Context().Value(userKey).(*Claims); ok {
		return c
	}
	return nil
}

// JWT 校验 Authorization: Bearer <token>，解析后注入上下文。
// 失败一律返回 401，由前端统一处理刷新或跳转登录。
func JWT(cfg *config.JWTConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			// 浏览器 WebSocket 无法设置自定义请求头，允许通过查询参数携带 token
			if q := r.URL.Query().Get("token"); q != "" {
				auth = "Bearer " + q
			}
		}
		if !strings.HasPrefix(auth, "Bearer ") {
				response.Fail(w, http.StatusUnauthorized, 401, "missing token")
				return
			}
			tokenStr := strings.TrimPrefix(auth, "Bearer ")
			claims := &Claims{}
			token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
				// 固定使用 HMAC，避免算法混淆攻击
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(cfg.Secret), nil
			})
			if err != nil || !token.Valid {
				response.Fail(w, http.StatusUnauthorized, 401, "invalid token")
				return
			}
			ctx := context.WithValue(r.Context(), userKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ParseToken 解析并校验 token，secret 必须与服务端签发一致。
func ParseToken(tokenStr, secret string, claims *Claims) (*jwt.Token, error) {
	return jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
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
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(cfg.AccessTTLMinutes) * time.Minute)),
		},
	}
	refreshClaims := Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
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
