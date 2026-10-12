package handler

import (
	"errors"
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"

	"lighting-equipment/internal/app/ds"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

var lightingUserLogin = regexp.MustCompile(`^[a-zA-Z0-9._-]{3,50}$`)

var errAuthNotImplemented = errors.New("метод появится в лабораторной №4 вместе с авторизацией")

// RegisterLightingUser — POST регистрации нового проектировщика.
// В БД сохраняется хэш пароля, в ответе нет ни пароля, ни хэша
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

// LoginLightingUser — POST аутентификации. Заглушка до лабораторной №4
func (h *Handler) LoginLightingUser(ctx *gin.Context) {
	h.errorHandler(ctx, http.StatusNotImplemented, errAuthNotImplemented)
}

// LogoutLightingUser — POST деавторизации. Заглушка до лабораторной №4
func (h *Handler) LogoutLightingUser(ctx *gin.Context) {
	h.errorHandler(ctx, http.StatusNotImplemented, errAuthNotImplemented)
}
