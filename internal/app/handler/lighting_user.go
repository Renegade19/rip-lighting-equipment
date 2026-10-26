package handler

import (
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"lighting-equipment/internal/app/ds"
	"lighting-equipment/internal/app/repository"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

var lightingUserLogin = regexp.MustCompile(`^[a-zA-Z0-9._-]{3,50}$`)

// Одна ошибка на неверный логин и неверный пароль: по ответу нельзя узнать,
// зарегистрирован ли такой логин
var errWrongCredentials = errors.New("неверный логин или пароль")

// RegisterLightingUser — POST регистрации нового проектировщика.
// В БД сохраняется хэш пароля, в ответе нет ни пароля, ни хэша
//
//	@Summary		Регистрация проектировщика
//	@Description	Логин 3–50 символов (латиница, цифры, . _ -), пароль от 6 символов. В БД хранится хэш bcrypt.
//	@Tags			lighting-users
//	@Accept			json
//	@Produce		json
//	@Param			request	body		ds.RegisterLightingUserSerializer	true	"Логин, пароль, ФИО"
//	@Success		201		{object}	ds.LightingUserSerializer
//	@Failure		400		{object}	ds.StatusSerializer
//	@Failure		409		{object}	ds.StatusSerializer
//	@Router			/lighting-users/register [post]
func (h *Handler) RegisterLightingUser(ctx *gin.Context) {
	var request ds.RegisterLightingUserSerializer
	if err := ctx.ShouldBindJSON(&request); err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, errInvalidJSON)
		return
	}

	login := strings.TrimSpace(request.Login)
	if !lightingUserLogin.MatchString(login) {
		h.errorHandler(ctx, http.StatusBadRequest, errors.New("login: от 3 до 50 символов — латинские буквы, цифры, точка, дефис, подчёркивание"))
		return
	}
	fullName := strings.TrimSpace(request.FullName)
	if fullName == "" || utf8.RuneCountInString(fullName) > 100 {
		h.errorHandler(ctx, http.StatusBadRequest, errors.New("fullName: укажите ФИО, не длиннее 100 символов"))
		return
	}
	// bcrypt учитывает только первые 72 байта пароля
	if utf8.RuneCountInString(request.Password) < 6 || len(request.Password) > 72 {
		h.errorHandler(ctx, http.StatusBadRequest, errors.New("password: от 6 символов и не длиннее 72 байт"))
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Error(err)
		h.errorHandler(ctx, http.StatusInternalServerError, errInternal)
		return
	}

	user, err := h.Repository.CreateLightingUser(login, string(passwordHash), fullName)
	if err != nil {
		h.repositoryErrorHandler(ctx, err)
		return
	}
	ctx.JSON(http.StatusCreated, ds.NewLightingUserSerializer(*user))
}

// LoginLightingUser — POST аутентификации: проверка логина и пароля, выдача JWT.
// В токен записываются ид и логин проектировщика, срок жизни берётся из настроек
//
//	@Summary		Аутентификация
//	@Description	Проверяет логин и пароль и выдаёт JWT. Токен передаётся в методах создателя
//	@Description	заголовком Authorization: Bearer {accessToken}.
//	@Tags			lighting-users
//	@Accept			json
//	@Produce		json
//	@Param			request	body		ds.LoginLightingUserSerializer	true	"Логин и пароль"
//	@Success		200		{object}	ds.LoginLightingUserResponseSerializer
//	@Failure		400		{object}	ds.StatusSerializer
//	@Failure		401		{object}	ds.StatusSerializer
//	@Router			/lighting-users/login [post]
func (h *Handler) LoginLightingUser(ctx *gin.Context) {
	var request ds.LoginLightingUserSerializer
	if err := ctx.ShouldBindJSON(&request); err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, errInvalidJSON)
		return
	}
	login := strings.TrimSpace(request.Login)
	if login == "" || request.Password == "" {
		h.errorHandler(ctx, http.StatusBadRequest, errors.New("укажите login и password"))
		return
	}

	user, err := h.Repository.GetLightingUserByLogin(login)
	if errors.Is(err, repository.ErrLightingUserNotFound) {
		h.errorHandler(ctx, http.StatusUnauthorized, errWrongCredentials)
		return
	}
	if err != nil {
		h.repositoryErrorHandler(ctx, err)
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(request.Password)) != nil {
		h.errorHandler(ctx, http.StatusUnauthorized, errWrongCredentials)
		return
	}

	now := time.Now()
	expiresIn := h.Config.JWT.ExpiresIn
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &ds.JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "lumicalc",
			Subject:   strconv.Itoa(user.UserID),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiresIn)),
		},
		UserID: user.UserID,
		Login:  user.Login,
	})
	accessToken, err := token.SignedString([]byte(h.Config.JWT.SecretKey))
	if err != nil {
		log.Error(err)
		h.errorHandler(ctx, http.StatusInternalServerError, errInternal)
		return
	}

	ctx.JSON(http.StatusOK, ds.LoginLightingUserResponseSerializer{
		AccessToken: accessToken,
		TokenType:   strings.TrimSpace(jwtPrefix),
		ExpiresIn:   int64(expiresIn.Seconds()),
		User:        ds.NewLightingUserSerializer(*user),
	})
}

// LogoutLightingUser — POST деавторизации: токен попадает в чёрный список Redis
// до конца своего срока жизни и больше не принимается
//
//	@Summary		Выход
//	@Description	Токен попадает в чёрный список Redis до конца срока жизни и больше не принимается.
//	@Tags			lighting-users
//	@Produce		json
//	@Success		200	{object}	ds.StatusSerializer
//	@Failure		401	{object}	ds.StatusSerializer
//	@Security		BearerAuth
//	@Router			/lighting-users/logout [post]
func (h *Handler) LogoutLightingUser(ctx *gin.Context) {
	claims := getCurrentJWTClaims(ctx)
	ttl := time.Until(claims.ExpiresAt.Time)

	err := h.Redis.WriteJWTToBlacklist(ctx.Request.Context(), ctx.GetString(contextJWTKey), claims.Login, ttl)
	if err != nil {
		log.Error(err)
		h.errorHandler(ctx, http.StatusInternalServerError, errInternal)
		return
	}
	ctx.JSON(http.StatusOK, ds.StatusSerializer{
		Status:      "ok",
		Description: "проектировщик вышел из системы, токен больше не действует",
	})
}
