package handler

import (
	"math"
	"net/http"
	"strconv"

	"lighting-equipment/internal/app/ds"

	"github.com/gin-gonic/gin"
)

// Границы шкалы фильтра мощности (Вт) для двойного слайдера на плитке
const (
	PowerScaleMin = 0.0
	PowerScaleMax = 10.0
)

// lightingEquipmentCard — светильник в том виде, в котором он уходит в шаблон:
// ключи Minio развёрнуты в url, количество лайков посчитано
type lightingEquipmentCard struct {
	EquipmentID      int
	EquipmentName    string
	ShortDescription string
	ImageURL         string
	VideoURL         string
	PowerWatt        float64
	BeamAngleDeg     int
	LikesTotal       int
}

func (h *Handler) toCard(equipment ds.LightingEquipment) lightingEquipmentCard {
	return lightingEquipmentCard{
		EquipmentID:      equipment.EquipmentID,
		EquipmentName:    equipment.EquipmentName,
		ShortDescription: equipment.ShortDescription,
		ImageURL:         h.Config.ImageURL(equipment.ImageKey),
		VideoURL:         h.Config.VideoURL(equipment.VideoKey),
		PowerWatt:        equipment.PowerWatt,
		BeamAngleDeg:     equipment.BeamAngleDeg,
		// Количество лайков нигде не хранится — считается по коллекции
		LikesTotal: len(equipment.LikedUserIDs),
	}
}

// Значение ползунка; если параметра нет или он не число — берём край шкалы
func parsePowerWatt(raw string, fallback float64) float64 {
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return fallback
	}
	return value
}

// GetLightingEquipmentTile — страница "Плитка":
// опубликованные светильники + фильтр по диапазону мощности (Вт)
func (h *Handler) GetLightingEquipmentTile(ctx *gin.Context) {
	minPowerWatt := parsePowerWatt(ctx.Query("minPowerWatt"), PowerScaleMin)
	maxPowerWatt := parsePowerWatt(ctx.Query("maxPowerWatt"), PowerScaleMax)
	// Ползунки независимы, поэтому нижнюю границу можно увести выше верхней —
	// в этом случае меняем границы местами, чтобы диапазон остался осмысленным
	if minPowerWatt > maxPowerWatt {
		minPowerWatt, maxPowerWatt = maxPowerWatt, minPowerWatt
	}

	lightingEquipment := h.Repository.GetPublishedLightingEquipment(minPowerWatt, maxPowerWatt)
	cards := make([]lightingEquipmentCard, 0, len(lightingEquipment))
	for _, equipment := range lightingEquipment {
		cards = append(cards, h.toCard(equipment))
	}

	ctx.HTML(http.StatusOK, "lighting-equipment-tile.html", gin.H{
		"Title":     "Плитка светильников",
		"ActiveTab": "tile",
		// Значения возвращаются в шаблон, чтобы поля фильтра сохранялись после запроса
		"MinPowerWatt":      minPowerWatt,
		"MaxPowerWatt":      maxPowerWatt,
		"PowerScaleMin":     PowerScaleMin,
		"PowerScaleMax":     PowerScaleMax,
		"LightingEquipment": cards,
	})
}

// GetLightingEquipmentDraft — страница "Добавление":
// показывает единственный светильник в статусе "черновик"
func (h *Handler) GetLightingEquipmentDraft(ctx *gin.Context) {
	data := gin.H{
		"Title":     "Добавление светильника",
		"ActiveTab": "draft",
	}
	if draft, ok := h.Repository.GetDraftLightingEquipment(); ok {
		data["Draft"] = h.toCard(draft)
	}
	ctx.HTML(http.StatusOK, "lighting-equipment-draft.html", data)
}

// GetLightingEquipmentFeed — страница "Лента": один светильник.
// Без id открывается первый опубликованный, с ?next=true — следующий после id.
// Удалённые светильники и черновики в ленте не открываются
func (h *Handler) GetLightingEquipmentFeed(ctx *gin.Context) {
	equipmentID := 0
	if raw := ctx.Param("equipment_id"); raw != "" {
		id, err := strconv.Atoi(raw)
		if err != nil || id <= 0 {
			ctx.String(http.StatusNotFound, "Светильник не найден")
			return
		}
		equipmentID = id
	}
	next := ctx.Query("next") == "true"

	equipment, ok := h.Repository.GetLightingEquipmentForFeed(equipmentID, next)
	if !ok {
		ctx.String(http.StatusNotFound, "Светильник не найден")
		return
	}

	ctx.HTML(http.StatusOK, "lighting-equipment-feed.html", gin.H{
		"Title":     equipment.EquipmentName,
		"ActiveTab": "feed",
		"Equipment": h.toCard(equipment),
	})
}
