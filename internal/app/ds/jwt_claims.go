package ds

import "github.com/golang-jwt/jwt/v5"

// JWTClaims — содержимое токена проектировщика.
// RegisteredClaims дают стандартные поля: срок жизни, время выдачи, издателя
type JWTClaims struct {
	jwt.RegisteredClaims
	UserID int    `json:"userId"`
	Login  string `json:"login"`
}
