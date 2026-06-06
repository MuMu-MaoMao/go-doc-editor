// Package auth 提供 JWT（JSON Web Token）的生成和验证功能。
// 使用 HS256 签名算法，令牌有效期 24 小时。
package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// jwtSecret 是 JWT 签名密钥。生产环境请通过环境变量或配置文件注入。
var jwtSecret = []byte("your-256-bit-secret-please-change-in-production")

// Claims 包含 JWT 令牌中存储的自定义声明（用户名）。
type Claims struct {
	Username string `json:"username"` // 用户名
	jwt.RegisteredClaims
}

// GenerateToken 为指定用户生成 JWT 令牌，有效期 24 小时。
func GenerateToken(username string) (string, error) {
	claims := Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ValidateToken 解析并验证 JWT 令牌，返回令牌中的声明信息。
func ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, err
}
