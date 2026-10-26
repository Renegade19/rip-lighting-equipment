package repository

import (
	"database/sql"
	"errors"
	"time"

	"lighting-equipment/internal/app/ds"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ErrDraftAlreadyExists — у проектировщика уже есть светильник в статусе "черновик"
var ErrDraftAlreadyExists = errors.New("черновик светильника уже существует")

// Базовый запрос: количество лайков считает сама БД подзапросом
// к таблице связи м-м, в коде срезы не перебираются
func (r *Repository) lightingEquipmentWithLikes() *gorm.DB {
	likesTotal := r.db.Model(&ds.LightingEquipmentLike{}).
		Select("COUNT(*)").
		Where("lighting_equipment_likes.equipment_id = lighting_equipment.equipment_id")

	return r.db.Model(&ds.LightingEquipment{}).
		Select("lighting_equipment.*, (?) AS likes_total", likesTotal)
}

// 1. GET списка для плитки: только опубликованные,
// фильтр по диапазону мощности выполняется в SQL
func (r *Repository) GetPublishedLightingEquipment(minPowerWatt, maxPowerWatt float64) ([]ds.LightingEquipmentWithLikes, error) {
	var lightingEquipment []ds.LightingEquipmentWithLikes
	err := r.lightingEquipmentWithLikes().
		Where("equipment_status = ?", ds.LightingEquipmentStatusPublished).
		Where("power_watt BETWEEN ? AND ?", minPowerWatt, maxPowerWatt).
		Order("equipment_id ASC").
		Find(&lightingEquipment).Error
	return lightingEquipment, err
}

// 2. GET ленты: БД возвращает сразу одну строку, фильтрации в коде нет.
// next == false — опубликованный светильник с указанным id.
// next == true  — следующий опубликованный после id; сортировка сначала ставит
// строки с большим id, поэтому после последнего светильника лента возвращается к первому.
// Лента без id — это next от нуля, то есть первый опубликованный
func (r *Repository) GetLightingEquipmentForFeed(equipmentID int, next bool) (*ds.LightingEquipmentWithLikes, error) {
	query := r.lightingEquipmentWithLikes().
		Where("equipment_status = ?", ds.LightingEquipmentStatusPublished)

	if next {
		query = query.Order(clause.OrderBy{Expression: clause.Expr{
			SQL:  "(lighting_equipment.equipment_id > ?) DESC, lighting_equipment.equipment_id ASC",
			Vars: []any{equipmentID},
		}})
	} else {
		query = query.Where("equipment_id = ?", equipmentID)
	}

	var equipment ds.LightingEquipmentWithLikes
	if err := query.Take(&equipment).Error; err != nil {
		return nil, err
	}
	return &equipment, nil
}

// 3. GET черновика: у проектировщика не более одного такого светильника
func (r *Repository) GetDraftLightingEquipment(creatorID int) (*ds.LightingEquipment, error) {
	var draft ds.LightingEquipment
	err := r.db.
		Where("creator_id = ? AND equipment_status = ?", creatorID, ds.LightingEquipmentStatusDraft).
		Take(&draft).Error
	if err != nil {
		return nil, err
	}
	return &draft, nil
}

// 4. POST добавления (кнопка «Далее»): создаёт черновик через ORM.
// Сами файлы в этой лабораторной не загружаются — в таблицу пишутся
// адреса несуществующих объектов Minio
func (r *Repository) CreateDraftLightingEquipment(creatorID int, equipmentName, imageURL, videoURL string) error {
	_, err := r.GetDraftLightingEquipment(creatorID)
	if err == nil {
		return ErrDraftAlreadyExists
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	draft := ds.LightingEquipment{
		EquipmentName:   equipmentName,
		EquipmentStatus: ds.LightingEquipmentStatusDraft,
		ImageURL:        &imageURL,
		VideoURL:        &videoURL,
		CreatedAt:       time.Now(),
		CreatorID:       creatorID,
	}
	return r.db.Create(&draft).Error
}

// 5. POST публикации: смена статуса черновик -> опубликован через ORM,
// одновременно заполняются описание, оба поля по теме и дата формирования
func (r *Repository) PublishDraftLightingEquipment(creatorID int, shortDescription string, powerWatt float64, beamAngleDeg int) error {
	draft, err := r.GetDraftLightingEquipment(creatorID)
	if err != nil {
		return err
	}

	formedAt := time.Now()
	draft.ShortDescription = &shortDescription
	draft.PowerWatt = &powerWatt
	draft.BeamAngleDeg = &beamAngleDeg
	draft.EquipmentStatus = ds.LightingEquipmentStatusPublished
	draft.FormedAt = &formedAt
	return r.db.Save(draft).Error
}

// 6. POST логического удаления: SQL-запрос UPDATE через курсор, без ORM.
// Запись остаётся в таблице, меняется только статус.
// RETURNING возвращает id изменённой строки, курсор указывает на эту строку
func (r *Repository) DeleteLightingEquipment(equipmentID int) error {
	query := `UPDATE lighting_equipment
	             SET equipment_status = ?
	           WHERE equipment_id = ?
	             AND equipment_status <> ?
	       RETURNING equipment_id`

	// Создание курсора (строковый указатель)
	row := r.db.Raw(query,
		ds.LightingEquipmentStatusDeleted, equipmentID, ds.LightingEquipmentStatusDeleted,
	).Row()

	// Сканирование строки: id удалённого светильника
	var deletedEquipmentID int
	err := row.Scan(&deletedEquipmentID)
	if errors.Is(err, sql.ErrNoRows) {
		// Светильника нет или он уже удалён — обновлять нечего
		return nil
	}
	return err
}
