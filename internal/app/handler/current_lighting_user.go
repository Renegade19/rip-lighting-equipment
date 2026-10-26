package handler

import (
	"lighting-equipment/internal/app/ds"

	"github.com/gin-gonic/gin"
)

// В лабораторной №3 проектировщик был зафиксирован константой через функцию-singleton.
// Теперь его ид берётся из JWT: middleware проверяет токен и кладёт его
// содержимое в контекст запроса. Вызовы в обработчиках остались прежними

// getCurrentLightingUserID — ид текущего проектировщика, 0 — гость
func getCurrentLightingUserID(ctx *gin.Context) int {
	if claims := getCurrentJWTClaims(ctx); claims != nil {
		return claims.UserID
	}
	return 0
}

// getCurrentJWTClaims — содержимое токена текущего проектировщика, nil — гость
func getCurrentJWTClaims(ctx *gin.Context) *ds.JWTClaims {
	value, ok := ctx.Get(contextJWTClaimsKey)
	if !ok {
		return nil
	}
	claims, _ := value.(*ds.JWTClaims)
	return claims
}
