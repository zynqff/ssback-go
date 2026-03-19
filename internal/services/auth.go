package services

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/ssback/internal/config"
	"golang.org/x/crypto/bcrypt"
)

type Claims struct {
	IsAdmin bool `json:"is_admin"`
	jwt.RegisteredClaims
}

func CreateAccessToken(username string, isAdmin bool) (string, error) {
	claims := Claims{
		IsAdmin: isAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   username,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.C.SecretKey))
}

func ParseToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(config.C.SecretKey), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, jwt.ErrTokenInvalidClaims
}

func HashPassword(password string) (string, error) {
	if len(password) > 72 {
		password = password[:72]
	}
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(b), err
}

func CheckPassword(password, hash string) bool {
	if len(password) > 72 {
		password = password[:72]
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
