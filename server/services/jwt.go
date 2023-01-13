package services

import (
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

type JWTManager struct {
	key        []byte
	keyfunc    jwt.Keyfunc
	expiration time.Duration
}

func NewJWTManager(key string, expiration time.Duration) *JWTManager {
	m := new(JWTManager)
	m.key = []byte(key)
	m.keyfunc = func(t *jwt.Token) (interface{}, error) { return m.key, nil }
	m.expiration = expiration

	return m
}

type UserClaims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// Creates JWT token for provided user.
// Returns token string representation and its expiration time.
func (m *JWTManager) IssueToken(username string) (string, time.Time, error) {
	expiresAt := time.Now().Add(m.expiration)
	claims := UserClaims{
		username,
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString(m.key)
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expiresAt, nil
}

// Tries to extract JWT token from Authorization header and parse it.
func (m *JWTManager) ParseToken(ctx *gin.Context) (*jwt.Token, UserClaims, error) {
	header := ctx.GetHeader("Authorization")
	if header == "" {
		return nil, UserClaims{}, errors.New("'Authorization' header is missing")
	}

	headerSplit := strings.Split(header, " ")
	if len(headerSplit) != 2 || headerSplit[0] != "Bearer" {
		return nil, UserClaims{}, errors.New("'Authorization' header value is malformed")
	}

	tokenString := headerSplit[1]
	claims := UserClaims{}
	token, err := jwt.ParseWithClaims(tokenString, &claims, m.keyfunc)
	return token, claims, err
}
