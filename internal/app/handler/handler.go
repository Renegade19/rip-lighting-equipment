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

// RegisterRoutes — три GET-страницы приложения
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.GET("/", func(ctx *gin.Context) {
		ctx.Redirect(http.StatusFound, "/lighting-equipment")
	})

	// Плитка: /lighting-equipment?minPowerWatt=2&maxPowerWatt=5
	r.GET("/lighting-equipment", h.GetLightingEquipmentTile)
	// Добавление: светильник в статусе "черновик"
	r.GET("/lighting-equipment/draft", h.GetLightingEquipmentDraft)
	// Лента: с панели вкладок id не указывается, с плитки — /lighting-equipment/3,
	// кнопка "Следующий" — /lighting-equipment/3?next=true
	r.GET("/lighting-equipment/feed", h.GetLightingEquipmentFeed)
	r.GET("/lighting-equipment/:equipment_id", h.GetLightingEquipmentFeed)
}
