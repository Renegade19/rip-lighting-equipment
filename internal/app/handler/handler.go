package handler

import (
	"errors"
	"net/http"

	"lighting-equipment/internal/app/config"
	"lighting-equipment/internal/app/repository"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type Handler struct {
	Repository *repository.Repository
	Config     *config.Config
}

func New(r *repository.Repository, cfg *config.Config) *Handler {
	return &Handler{Repository: r, Config: cfg}
}

// RegisterRoutes — десять методов веб-сервиса, все адреса начинаются с /api
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	// Неизвестные адреса и методы тоже отвечают JSON, а не текстом
	r.HandleMethodNotAllowed = true
	r.NoRoute(func(ctx *gin.Context) {
		h.errorHandler(ctx, http.StatusNotFound, errors.New("такого адреса в веб-сервисе нет"))
	})
	r.NoMethod(func(ctx *gin.Context) {
		h.errorHandler(ctx, http.StatusMethodNotAllowed, errors.New("метод HTTP для этого адреса не поддерживается"))
	})

	api := r.Group("/api")

	// Домен "светильники" (услуги)
	lightingEquipment := api.Group("/lighting-equipment")
	{
		// Список опубликованных с фильтром: ?minPowerWatt=4.5&maxPowerWatt=10
		lightingEquipment.GET("", h.GetLightingEquipmentList)
		// Лента — один метод: без параметров — первый опубликованный,
		// ?equipmentId=3 — по ид, ?equipmentId=3&next=true — следующий после него
		lightingEquipment.GET("/feed", h.GetLightingEquipmentFeed)
		// Черновик текущего проектировщика, ид не указывается
		lightingEquipment.GET("/draft", h.GetLightingEquipmentDraft)
		// Добавление: название + файлы фото и видео (multipart/form-data)
		lightingEquipment.POST("", h.CreateLightingEquipment)
		// Публикация: черновик -> опубликован
		lightingEquipment.PUT("/:equipment_id/publish", h.PublishLightingEquipment)
		// Логическое удаление, только своих светильников
		lightingEquipment.DELETE("/:equipment_id", h.DeleteLightingEquipment)
		// Лайк от текущего проектировщика: 1 ставит, 0 отменяет
		lightingEquipment.POST("/:equipment_id/like", h.SetLightingEquipmentLike)
	}

	// Домен "проектировщики" (пользователи)
	lightingUsers := api.Group("/lighting-users")
	{
		lightingUsers.POST("/register", h.RegisterLightingUser)
		// Заглушки до лабораторной №4
		lightingUsers.POST("/login", h.LoginLightingUser)
		lightingUsers.POST("/logout", h.LogoutLightingUser)
	}
}

// Ошибка в едином для всех методов формате JSON
func (h *Handler) errorHandler(ctx *gin.Context, errorStatusCode int, err error) {
	ctx.JSON(errorStatusCode, gin.H{
		"status":      "error",
		"description": err.Error(),
	})
}

var errInternal = errors.New("внутренняя ошибка сервера")

// Ошибки репозитория: бизнес-ошибки получают свои коды HTTP,
// остальные пишутся в журнал, а клиенту уходит 500 без подробностей
func (h *Handler) repositoryErrorHandler(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, repository.ErrLightingEquipmentNotFound):
		h.errorHandler(ctx, http.StatusNotFound, err)
	case errors.Is(err, repository.ErrForeignLightingEquipment):
		h.errorHandler(ctx, http.StatusForbidden, err)
	case errors.Is(err, repository.ErrLightingEquipmentNotDraft),
		errors.Is(err, repository.ErrDraftAlreadyExists),
		errors.Is(err, repository.ErrLoginAlreadyTaken):
		h.errorHandler(ctx, http.StatusConflict, err)
	default:
		log.Error(err)
		h.errorHandler(ctx, http.StatusInternalServerError, errInternal)
	}
}
