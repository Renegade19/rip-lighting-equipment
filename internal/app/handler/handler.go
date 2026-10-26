package handler

import (
	"errors"
	"net/http"

	"lighting-equipment/internal/app/config"
	"lighting-equipment/internal/app/redis"
	"lighting-equipment/internal/app/repository"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Handler struct {
	Repository *repository.Repository
	Redis      *redis.Client
	Config     *config.Config
}

func New(r *repository.Repository, redisClient *redis.Client, cfg *config.Config) *Handler {
	return &Handler{Repository: r, Redis: redisClient, Config: cfg}
}

// RegisterRoutes — десять методов веб-сервиса, все адреса начинаются с /api.
// Гостю доступно только чтение: список, лента, регистрация и вход.
// Остальные методы — создателя, им нужен JWT в заголовке Authorization
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	// Документация Swagger: http://localhost:3000/swagger/index.html
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	optionalAuth := h.WithOptionalAuth()
	authRequired := h.WithAuthCheck()

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
		lightingEquipment.GET("", optionalAuth, h.GetLightingEquipmentList)
		// Лента: без ид — первый опубликованный, /3 — по ид, /3?next=true — следующий
		lightingEquipment.GET("/feed", optionalAuth, h.GetLightingEquipmentFeedFirst)
		lightingEquipment.GET("/:equipment_id", optionalAuth, h.GetLightingEquipmentFeed)
		// Черновик текущего проектировщика, ид не указывается
		lightingEquipment.GET("/draft", authRequired, h.GetLightingEquipmentDraft)
		// Добавление: название + файлы фото и видео (multipart/form-data)
		lightingEquipment.POST("", authRequired, h.CreateLightingEquipment)
		// Публикация: черновик -> опубликован
		lightingEquipment.PUT("/:equipment_id/publish", authRequired, h.PublishLightingEquipment)
		// Логическое удаление, только своих светильников
		lightingEquipment.DELETE("/:equipment_id", authRequired, h.DeleteLightingEquipment)
		// Лайк от текущего проектировщика: 1 ставит, 0 отменяет
		lightingEquipment.POST("/:equipment_id/like", authRequired, h.SetLightingEquipmentLike)
	}

	// Домен "проектировщики" (пользователи)
	lightingUsers := api.Group("/lighting-users")
	{
		lightingUsers.POST("/register", h.RegisterLightingUser)
		// Вход: выдаёт JWT
		lightingUsers.POST("/login", h.LoginLightingUser)
		// Выход: JWT попадает в чёрный список Redis
		lightingUsers.POST("/logout", authRequired, h.LogoutLightingUser)
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
