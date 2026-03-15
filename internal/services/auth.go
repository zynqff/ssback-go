package services

import (
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/ssback/internal/config"
	"golang.org/x/crypto/bcrypt"
)

// virtualAdminState stores in-memory state for virtual admins
var (
	virtualAdminReadPoems   = make(map[string][]string)
	virtualAdminPinnedPoems = make(map[string]*string)
	vaMu                    sync.Mutex
)

type Claims struct {
	Username string `json:"sub"`
	IsAdmin  bool   `json:"is_admin"`
	jwt.RegisteredClaims
}

func CreateAccessToken(username string, isAdmin bool) (string, error) {
	claims := Claims{
		Username: username,
		IsAdmin:  isAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
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

func GetVirtualAdminReadPoems(username string) []string {
	vaMu.Lock()
	defer vaMu.Unlock()
	r := virtualAdminReadPoems[username]
	if r == nil {
		return []string{}
	}
	return r
}

func GetVirtualAdminPinnedPoem(username string) *string {
	vaMu.Lock()
	defer vaMu.Unlock()
	return virtualAdminPinnedPoems[username]
}

func ToggleVirtualAdminRead(username, title string) string {
	vaMu.Lock()
	defer vaMu.Unlock()
	reads := virtualAdminReadPoems[username]
	for i, t := range reads {
		if t == title {
			virtualAdminReadPoems[username] = append(reads[:i], reads[i+1:]...)
			return "unmarked"
		}
	}
	virtualAdminReadPoems[username] = append(reads, title)
	return "marked"
}

func ToggleVirtualAdminPinned(username, title string) (string, *string) {
	vaMu.Lock()
	defer vaMu.Unlock()
	cur := virtualAdminPinnedPoems[username]
	if cur != nil && *cur == title {
		virtualAdminPinnedPoems[username] = nil
		return "unpinned", nil
	}
	t := title
	virtualAdminPinnedPoems[username] = &t
	return "pinned", &t
}
