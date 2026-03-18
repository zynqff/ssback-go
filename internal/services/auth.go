package services

import (
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/ssback/internal/config"
	"golang.org/x/crypto/bcrypt"
)

var (
	virtualAdminReadPoems   = make(map[string][]int64)
	virtualAdminPinnedPoems = make(map[string]*int64)
	vaMu                    sync.Mutex
)

type Claims struct {
	IsAdmin bool `json:"is_admin"`
	jwt.RegisteredClaims
}

func (c *Claims) Username() string {
	return c.Subject
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

func IsVirtualAdmin(username string) bool {
	_, ok := config.C.AdminsMap()[username]
	return ok
}

func CheckVirtualAdmin(username, password string) bool {
	p, ok := config.C.AdminsMap()[username]
	return ok && p == password
}

func GetVirtualAdminReadPoems(username string) []int64 {
	vaMu.Lock()
	defer vaMu.Unlock()
	r := virtualAdminReadPoems[username]
	if r == nil {
		return []int64{}
	}
	return r
}

func GetVirtualAdminPinnedPoem(username string) *int64 {
	vaMu.Lock()
	defer vaMu.Unlock()
	return virtualAdminPinnedPoems[username]
}

func ToggleVirtualAdminRead(username string, poemID int64) string {
	vaMu.Lock()
	defer vaMu.Unlock()
	reads := virtualAdminReadPoems[username]
	for i, id := range reads {
		if id == poemID {
			virtualAdminReadPoems[username] = append(reads[:i], reads[i+1:]...)
			return "unmarked"
		}
	}
	virtualAdminReadPoems[username] = append(reads, poemID)
	return "marked"
}

func ToggleVirtualAdminPinned(username string, poemID int64) (string, *int64) {
	vaMu.Lock()
	defer vaMu.Unlock()
	cur := virtualAdminPinnedPoems[username]
	if cur != nil && *cur == poemID {
		virtualAdminPinnedPoems[username] = nil
		return "unpinned", nil
	}
	id := poemID
	virtualAdminPinnedPoems[username] = &id
	return "pinned", &id
}
