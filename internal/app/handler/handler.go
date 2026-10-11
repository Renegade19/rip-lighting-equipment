package handler

import (
	"net/http"

	"lighting-equipment/internal/app/config"
	"lighting-equipment/internal/app/repository"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	Repository *repository.Repository
	Config     *config.Config
}

func New(r *repository.Repository, cfg *config.Config) *Handler {
	return &Handler{Repository: r, Config: cfg}
}

// RegisterRoutes — шесть HTTP-методов приложения: три GET и три POST
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.GET("/", func(ctx *gin.Context) {
		ctx.Redirect(http.StatusFound, "/lighting-equipment")
	})

	// Плитка: /lighting-equipment?minPowerWatt=2&maxPowerWatt=5
	r.GET("/lighting-equipment", h.GetLightingEquipmentTile)
	// Добавление: черновик текущего проектировщика
	r.GET("/lighting-equipment/draft", h.GetLightingEquipmentDraft)
	// Лента: с панели вкладок id не указывается, с плитки — /lighting-equipment/3,
	// кнопка "Следующий" — /lighting-equipment/3?next=true
	r.GET("/lighting-equipment/feed", h.GetLightingEquipmentFeed)
	r.GET("/lighting-equipment/:equipment_id", h.GetLightingEquipmentFeed)

	// Кнопка "Далее": создание черновика (ORM)
	r.POST("/lighting-equipment/draft", h.CreateLightingEquipmentDraft)
	// Кнопка "Опубликовать": смена статуса (ORM)
	r.POST("/lighting-equipment/publish", h.PublishLightingEquipmentDraft)
	// Кнопка "Удалить" на плитке: логическое удаление (SQL-курсор, без ORM)
	r.POST("/lighting-equipment/delete", h.DeleteLightingEquipment)
}

func (h *Handler) errorHandler(ctx *gin.Context, errorStatusCode int, err error) {
	ctx.String(errorStatusCode, err.Error())
}
