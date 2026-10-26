package handler

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"lighting-equipment/internal/app/ds"
	"lighting-equipment/internal/app/repository"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

var (
	errInvalidEquipmentID = errors.New("ид светильника должен быть положительным целым числом")
	errInvalidJSON        = errors.New("тело запроса должно быть корректным JSON")
)

// Ид светильника из адреса запроса
func parseEquipmentID(ctx *gin.Context) (int, error) {
	equipmentID, err := strconv.Atoi(ctx.Param("equipment_id"))
	if err != nil || equipmentID <= 0 {
		return 0, errInvalidEquipmentID
	}
	return equipmentID, nil
}

// Необязательная граница фильтра мощности: если параметра нет, возвращается nil
func parsePowerWatt(ctx *gin.Context, name string) (*float64, error) {
	raw := strings.TrimSpace(ctx.Query(name))
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return nil, fmt.Errorf("параметр %s должен быть числом", name)
	}
	return &value, nil
}

// GetLightingEquipmentList — GET списка: опубликованные светильники
// с фильтром по диапазону мощности (Вт). У каждого — признак 0/1,
// создан ли он текущим проектировщиком
func (h *Handler) GetLightingEquipmentList(ctx *gin.Context) {
	minPowerWatt, err := parsePowerWatt(ctx, "minPowerWatt")
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}
	maxPowerWatt, err := parsePowerWatt(ctx, "maxPowerWatt")
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}
	// Ползунки фильтра независимы, нижнюю границу можно увести выше верхней —
	// в этом случае границы меняются местами
	if minPowerWatt != nil && maxPowerWatt != nil && *minPowerWatt > *maxPowerWatt {
		minPowerWatt, maxPowerWatt = maxPowerWatt, minPowerWatt
	}

	lightingEquipment, err := h.Repository.GetPublishedLightingEquipment(getCurrentLightingUserID(), minPowerWatt, maxPowerWatt)
	if err != nil {
		h.repositoryErrorHandler(ctx, err)
		return
	}

	tiles := make([]ds.LightingEquipmentTileSerializer, 0, len(lightingEquipment))
	for _, equipment := range lightingEquipment {
		tiles = append(tiles, ds.NewLightingEquipmentTileSerializer(equipment, h.Config.MediaURL))
	}
	ctx.JSON(http.StatusOK, tiles)
}

// GetLightingEquipmentFeed — GET ленты: один опубликованный светильник
// с признаком 0/1, лайкнул ли его текущий проектировщик.
// Один метод, оба параметра необязательны:
//   - без параметров — первый опубликованный (вкладка «Лента» в меню, ид ещё неизвестен);
//   - ?equipmentId=3 — светильник с этим ид;
//   - ?equipmentId=3&next=true — следующий опубликованный после этого ид
func (h *Handler) GetLightingEquipmentFeed(ctx *gin.Context) {
	equipmentID := 0
	next := ctx.Query("next") == "true"

	if raw := strings.TrimSpace(ctx.Query("equipmentId")); raw != "" {
		id, err := strconv.Atoi(raw)
		if err != nil || id <= 0 {
			h.errorHandler(ctx, http.StatusBadRequest, errInvalidEquipmentID)
			return
		}
		equipmentID = id
	} else {
		// Ид не указан: следующий после нуля — первый опубликованный
		next = true
	}

	equipment, err := h.Repository.GetLightingEquipmentForFeed(getCurrentLightingUserID(), equipmentID, next)
	if err != nil {
		h.repositoryErrorHandler(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, ds.NewLightingEquipmentFeedSerializer(*equipment, h.Config.MediaURL))
}

// GetLightingEquipmentDraft — GET черновика текущего проектировщика.
// Ид не указывается: черновик у проектировщика не более одного
func (h *Handler) GetLightingEquipmentDraft(ctx *gin.Context) {
	draft, err := h.Repository.GetDraftLightingEquipment(getCurrentLightingUserID())
	if errors.Is(err, repository.ErrLightingEquipmentNotFound) {
		h.errorHandler(ctx, http.StatusNotFound, errors.New("у проектировщика нет светильника в статусе «черновик»"))
		return
	}
	if err != nil {
		h.repositoryErrorHandler(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, ds.NewLightingEquipmentSerializer(*draft, h.Config.MediaURL))
}

// CreateLightingEquipment — POST добавления: название и два файла (multipart/form-data).
// Файлы сохраняются в Minio под сгенерированными именами на латинице,
// в БД записываются имена этих файлов. Светильник создаётся в статусе "черновик"
func (h *Handler) CreateLightingEquipment(ctx *gin.Context) {
	currentUserID := getCurrentLightingUserID()

	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, maxCreateRequestBytes)
	if err := ctx.Request.ParseMultipartForm(8 << 20); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			h.errorHandler(ctx, http.StatusRequestEntityTooLarge, errors.New("запрос слишком большой"))
			return
		}
		h.errorHandler(ctx, http.StatusBadRequest, errors.New("ожидается форма multipart/form-data с полями equipmentName, equipmentImage, equipmentVideo"))
		return
	}

	equipmentName := strings.TrimSpace(ctx.PostForm("equipmentName"))
	if equipmentName == "" || utf8.RuneCountInString(equipmentName) > 150 {
		h.errorHandler(ctx, http.StatusBadRequest, errors.New("equipmentName: укажите наименование светильника, не длиннее 150 символов"))
		return
	}

	imageHeader, err := ctx.FormFile("equipmentImage")
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, errors.New("equipmentImage: приложите файл изображения"))
		return
	}
	videoHeader, err := ctx.FormFile("equipmentVideo")
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, errors.New("equipmentVideo: приложите файл видео"))
		return
	}

	image, err := validateLightingEquipmentMedia(imageHeader, "image", "equipmentImage", equipmentImageTypes, maxEquipmentImageBytes)
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}
	video, err := validateLightingEquipmentMedia(videoHeader, "video", "equipmentVideo", equipmentVideoTypes, maxEquipmentVideoBytes)
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}

	// Черновик у проектировщика один. Проверяем заранее, чтобы не загружать файлы зря
	_, err = h.Repository.GetDraftLightingEquipment(currentUserID)
	if err == nil {
		h.repositoryErrorHandler(ctx, repository.ErrDraftAlreadyExists)
		return
	}
	if !errors.Is(err, repository.ErrLightingEquipmentNotFound) {
		h.repositoryErrorHandler(ctx, err)
		return
	}

	requestCtx := ctx.Request.Context()
	if err = h.uploadLightingEquipmentMedia(requestCtx, image); err != nil {
		h.repositoryErrorHandler(ctx, err)
		return
	}
	if err = h.uploadLightingEquipmentMedia(requestCtx, video); err != nil {
		h.removeLightingEquipmentMedia(ctx, image)
		h.repositoryErrorHandler(ctx, err)
		return
	}

	draft, err := h.Repository.CreateDraftLightingEquipment(
		currentUserID,
		equipmentName,
		image.fileName,
		video.fileName,
	)
	if err != nil {
		// Запись в БД не создана — загруженные файлы больше не нужны
		h.removeLightingEquipmentMedia(ctx, image, video)
		h.repositoryErrorHandler(ctx, err)
		return
	}

	// Новый светильник — черновик, он доступен методом получения черновика
	ctx.Header("Location", "/api/lighting-equipment/draft")
	ctx.JSON(http.StatusCreated, ds.NewLightingEquipmentSerializer(*draft, h.Config.MediaURL))
}

// Удаляет из Minio файлы, для которых не получилось создать запись в БД
func (h *Handler) removeLightingEquipmentMedia(ctx *gin.Context, files ...*lightingEquipmentMedia) {
	for _, file := range files {
		if err := h.Repository.RemoveLightingEquipmentMedia(ctx.Request.Context(), file.fileName); err != nil {
			log.Warnf("не удалось удалить файл %s из Minio: %v", file.fileName, err)
		}
	}
}

// PublishLightingEquipment — PUT публикации: описание и оба поля по теме обязательны.
// Статус и дату формирования клиент не присылает — их проставляет сервер
func (h *Handler) PublishLightingEquipment(ctx *gin.Context) {
	equipmentID, err := parseEquipmentID(ctx)
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}

	var request ds.PublishLightingEquipmentSerializer
	if err = ctx.ShouldBindJSON(&request); err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, errInvalidJSON)
		return
	}

	shortDescription := ""
	if request.ShortDescription != nil {
		shortDescription = strings.TrimSpace(*request.ShortDescription)
	}
	if shortDescription == "" || utf8.RuneCountInString(shortDescription) > 500 {
		h.errorHandler(ctx, http.StatusBadRequest, errors.New("shortDescription: укажите краткое описание, не длиннее 500 символов"))
		return
	}
	if request.PowerWatt == nil || *request.PowerWatt <= 0 || *request.PowerWatt > 9999.99 {
		h.errorHandler(ctx, http.StatusBadRequest, errors.New("powerWatt: укажите мощность в ваттах, больше 0 и не больше 9999.99"))
		return
	}
	if request.BeamAngleDeg == nil || *request.BeamAngleDeg <= 0 || *request.BeamAngleDeg > 360 {
		h.errorHandler(ctx, http.StatusBadRequest, errors.New("beamAngleDeg: укажите угол рассеивания в градусах, от 1 до 360"))
		return
	}
	// В таблице мощность хранится с двумя знаками после запятой
	powerWatt := math.Round(*request.PowerWatt*100) / 100

	equipment, err := h.Repository.PublishLightingEquipment(
		getCurrentLightingUserID(), equipmentID, shortDescription, powerWatt, *request.BeamAngleDeg,
	)
	if err != nil {
		h.repositoryErrorHandler(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, ds.NewLightingEquipmentSerializer(*equipment, h.Config.MediaURL))
}

// DeleteLightingEquipment — DELETE: логическое удаление своего светильника.
// Запись остаётся в таблице, статус меняется на "удалён"
func (h *Handler) DeleteLightingEquipment(ctx *gin.Context) {
	equipmentID, err := parseEquipmentID(ctx)
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}

	if err = h.Repository.DeleteLightingEquipment(getCurrentLightingUserID(), equipmentID); err != nil {
		h.repositoryErrorHandler(ctx, err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

// SetLightingEquipmentLike — POST лайка от текущего проектировщика:
// поле equipmentLike = 1 ставит лайк, 0 отменяет
func (h *Handler) SetLightingEquipmentLike(ctx *gin.Context) {
	equipmentID, err := parseEquipmentID(ctx)
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}

	var request ds.SetLightingEquipmentLikeSerializer
	if err = ctx.ShouldBindJSON(&request); err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, errInvalidJSON)
		return
	}
	if request.EquipmentLike == nil || (*request.EquipmentLike != 0 && *request.EquipmentLike != 1) {
		h.errorHandler(ctx, http.StatusBadRequest, errors.New("equipmentLike: укажите 1, чтобы поставить лайк, или 0, чтобы отменить"))
		return
	}

	likesTotal, err := h.Repository.SetLightingEquipmentLike(getCurrentLightingUserID(), equipmentID, *request.EquipmentLike == 1)
	if err != nil {
		h.repositoryErrorHandler(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, ds.LightingEquipmentLikesSerializer{
		EquipmentID:          equipmentID,
		LikesTotal:           likesTotal,
		IsLikedByCurrentUser: *request.EquipmentLike,
	})
}
