package handler

import (
	"errors"
	"net/http"
	"strings"

	"lighting-equipment/internal/app/ds"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	log "github.com/sirupsen/logrus"
)

const jwtPrefix = "Bearer "

// Ключи, под которыми middleware кладёт данные токена в контекст запроса
const (
	contextJWTClaimsKey = "jwtClaims"
	contextJWTKey       = "jwt"
)

var (
	errAuthRequired = errors.New("метод доступен только авторизованному проектировщику: передайте заголовок Authorization: Bearer <токен>")
	errInvalidToken = errors.New("токен недействителен или истёк, выполните вход заново")
	errTokenRevoked = errors.New("с этим токеном проектировщик уже вышел из системы, выполните вход заново")
	// Redis недоступен — проверить чёрный список нельзя
	errBlacklistUnavailable = errors.New("не удалось проверить токен")
)

// parseAuthorization читает JWT из заголовка Authorization: проверяет префикс,
// подпись, срок жизни и отсутствие токена в чёрном списке Redis
func (h *Handler) parseAuthorization(ctx *gin.Context) (*ds.JWTClaims, string, error) {
	jwtStr := ctx.GetHeader("Authorization")
	if !strings.HasPrefix(jwtStr, jwtPrefix) {
		return nil, "", errAuthRequired
	}
	// отрезаем префикс
	jwtStr = strings.TrimSpace(jwtStr[len(jwtPrefix):])

	claims := &ds.JWTClaims{}
	_, err := jwt.ParseWithClaims(jwtStr, claims, func(token *jwt.Token) (any, error) {
		return []byte(h.Config.JWT.SecretKey), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}), jwt.WithExpirationRequired())
	if err != nil || claims.UserID <= 0 {
		return nil, "", errInvalidToken
	}

	revoked, err := h.Redis.CheckJWTInBlacklist(ctx.Request.Context(), jwtStr)
	if err != nil {
		log.Error(err)
		return nil, "", errBlacklistUnavailable
	}
	if revoked {
		return nil, "", errTokenRevoked
	}
	return claims, jwtStr, nil
}

// WithAuthCheck — middleware методов создателя: без действующего токена ответ 401
func (h *Handler) WithAuthCheck() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		claims, jwtStr, err := h.parseAuthorization(ctx)
		if errors.Is(err, errBlacklistUnavailable) {
			h.errorHandler(ctx, http.StatusInternalServerError, errInternal)
			ctx.Abort()
			return
		}
		if err != nil {
			ctx.Header("WWW-Authenticate", `Bearer realm="lumicalc"`)
			h.errorHandler(ctx, http.StatusUnauthorized, err)
			ctx.Abort()
			return
		}
		ctx.Set(contextJWTClaimsKey, claims)
		ctx.Set(contextJWTKey, jwtStr)
		ctx.Next()
	}
}

// WithOptionalAuth — middleware методов чтения: доступны и гостю.
// С действующим токеном в ответе заполняются признаки текущего проектировщика,
// без токена или с недействительным — запрос выполняется как от гостя
func (h *Handler) WithOptionalAuth() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if claims, jwtStr, err := h.parseAuthorization(ctx); err == nil {
			ctx.Set(contextJWTClaimsKey, claims)
			ctx.Set(contextJWTKey, jwtStr)
		}
		ctx.Next()
	}
}
