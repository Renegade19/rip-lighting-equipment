package handler

import (
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"

	"lighting-equipment/internal/app/ds"
	"lighting-equipment/internal/app/repository"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Границы шкалы фильтра мощности (Вт) для двойного слайдера на плитке
const (
	PowerScaleMin = 0.0
	PowerScaleMax = 10.0
)

var errLightingEquipmentNotFound = errors.New("Светильник не найден")

// lightingEquipmentCard — светильник в том виде, в котором он уходит в шаблон:
// вместо пустых и нерабочих url подставлены фото и видео по умолчанию
type lightingEquipmentCard struct {
	ds.LightingEquipment
	LikesTotal int64
	ImageURL   string
	VideoURL   string
}

func toCard(equipment ds.LightingEquipment, likesTotal int64) lightingEquipmentCard {
	return lightingEquipmentCard{
		LightingEquipment: equipment,
		LikesTotal:        likesTotal,
		ImageURL:          resolveImageURL(equipment.ImageURL),
		VideoURL:          resolveVideoURL(equipment.VideoURL),
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

	lightingEquipment, err := h.Repository.GetPublishedLightingEquipment(minPowerWatt, maxPowerWatt)
	if err != nil {
		log.Error(err)
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}
	cards := make([]lightingEquipmentCard, 0, len(lightingEquipment))
	for _, equipment := range lightingEquipment {
		cards = append(cards, toCard(equipment.LightingEquipment, equipment.LikesTotal))
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

// GetLightingEquipmentDraft — страница "Добавление".
// Черновик у проектировщика не более одного; если его нет,
// страница открывается пустой с кнопкой "Далее"
func (h *Handler) GetLightingEquipmentDraft(ctx *gin.Context) {
	data := gin.H{
		"Title":           "Добавление светильника",
		"ActiveTab":       "draft",
		"DefaultImageURL": DefaultImageURL,
		"DefaultVideoURL": DefaultVideoURL,
	}

	draft, err := h.Repository.GetDraftLightingEquipment(getCurrentLightingUserID())
	switch {
	case err == nil:
		data["Draft"] = toCard(*draft, 0)
	case !errors.Is(err, gorm.ErrRecordNotFound):
		log.Error(err)
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.HTML(http.StatusOK, "lighting-equipment-draft.html", data)
}

// GetLightingEquipmentFeed — страница "Лента": один светильник.
// Без id открывается первый опубликованный, с ?next=true — следующий после id.
// Удалённые светильники и черновики в ленте не открываются
func (h *Handler) GetLightingEquipmentFeed(ctx *gin.Context) {
	equipmentID := 0
	next := ctx.Query("next") == "true"

	if raw := ctx.Param("equipment_id"); raw != "" {
		id, err := strconv.Atoi(raw)
		if err != nil || id <= 0 {
			h.errorHandler(ctx, http.StatusNotFound, errLightingEquipmentNotFound)
			return
		}
		equipmentID = id
	} else {
		// Лента без id: следующий после нуля — первый опубликованный
		next = true
	}

	equipment, err := h.Repository.GetLightingEquipmentForFeed(equipmentID, next)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		h.errorHandler(ctx, http.StatusNotFound, errLightingEquipmentNotFound)
		return
	}
	if err != nil {
		log.Error(err)
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.HTML(http.StatusOK, "lighting-equipment-feed.html", gin.H{
		"Title":     equipment.EquipmentName,
		"ActiveTab": "feed",
		"Equipment": toCard(equipment.LightingEquipment, equipment.LikesTotal),
	})
}

// CreateLightingEquipmentDraft — POST кнопки "Далее": создание черновика через ORM.
// Форма отправляется без multipart, поэтому сами файлы на сервер не приходят —
// приходят только их имена. В таблицу записываются адреса объектов Minio,
// которых там нет, поэтому на страницах показываются фото и видео по умолчанию
func (h *Handler) CreateLightingEquipmentDraft(ctx *gin.Context) {
	equipmentName := strings.TrimSpace(ctx.PostForm("equipmentName"))
	if equipmentName != "" {
		imageURL := h.Config.ImageURL(generateMediaKey("image", ctx.PostForm("imageFile"), ".jpg"))
		videoURL := h.Config.VideoURL(generateMediaKey("video", ctx.PostForm("videoFile"), ".mp4"))

		err := h.Repository.CreateDraftLightingEquipment(getCurrentLightingUserID(), equipmentName, imageURL, videoURL)
		if err != nil && !errors.Is(err, repository.ErrDraftAlreadyExists) {
			log.Error(err)
			h.errorHandler(ctx, http.StatusInternalServerError, err)
			return
		}
	}
	ctx.Redirect(http.StatusFound, "/lighting-equipment/draft")
}

// PublishLightingEquipmentDraft — POST кнопки "Опубликовать":
// описание и оба поля по теме обязательны, статус меняется через ORM
func (h *Handler) PublishLightingEquipmentDraft(ctx *gin.Context) {
	shortDescription := strings.TrimSpace(ctx.PostForm("shortDescription"))
	powerWatt, powerErr := strconv.ParseFloat(ctx.PostForm("powerWatt"), 64)
	beamAngleDeg, angleErr := strconv.Atoi(ctx.PostForm("beamAngleDeg"))

	// Поля заполнены неверно — возвращаемся на страницу добавления, черновик не меняется
	if shortDescription == "" || len([]rune(shortDescription)) > 500 ||
		powerErr != nil || math.IsNaN(powerWatt) || powerWatt <= 0 || powerWatt > 9999.99 ||
		angleErr != nil || beamAngleDeg <= 0 || beamAngleDeg > 360 {
		ctx.Redirect(http.StatusFound, "/lighting-equipment/draft")
		return
	}

	err := h.Repository.PublishDraftLightingEquipment(getCurrentLightingUserID(), shortDescription, powerWatt, beamAngleDeg)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		ctx.Redirect(http.StatusFound, "/lighting-equipment/draft")
		return
	}
	if err != nil {
		log.Error(err)
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}
	ctx.Redirect(http.StatusFound, "/lighting-equipment")
}

// DeleteLightingEquipment — POST логического удаления с плитки:
// статус меняется на "удалён" сырым SQL, без ORM
func (h *Handler) DeleteLightingEquipment(ctx *gin.Context) {
	equipmentID, err := strconv.Atoi(ctx.PostForm("equipmentId"))
	if err == nil {
		if err = h.Repository.DeleteLightingEquipment(equipmentID); err != nil {
			log.Error(err)
			h.errorHandler(ctx, http.StatusInternalServerError, err)
			return
		}
	}
	ctx.Redirect(http.StatusFound, "/lighting-equipment")
}
