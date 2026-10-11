package repository

import (
	"context"
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

// 6. POST логического удаления: сырой SQL через курсор, без ORM.
// Запись остаётся в таблице, меняется только статус.
// Курсор объявляется с FOR UPDATE, строка читается через FETCH,
// обновление идёт по позиции курсора (WHERE CURRENT OF) —
// всё внутри одной транзакции, как того требует PostgreSQL
func (r *Repository) DeleteLightingEquipment(equipmentID int) error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}

	ctx := context.Background()
	tx, err := sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx,
		`DECLARE lighting_equipment_cursor CURSOR FOR
		   SELECT equipment_id
		     FROM lighting_equipment
		    WHERE equipment_id = $1
		      AND equipment_status <> $2
		      FOR UPDATE`,
		equipmentID, ds.LightingEquipmentStatusDeleted,
	)
	if err != nil {
		return err
	}

	// Читаем одну строку из курсора
	var fetchedEquipmentID int
	err = tx.QueryRowContext(ctx, `FETCH NEXT FROM lighting_equipment_cursor`).Scan(&fetchedEquipmentID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		// Светильника нет или он уже удалён — обновлять нечего
	case err != nil:
		return err
	default:
		// Обновляем именно ту строку, на которой стоит курсор
		_, err = tx.ExecContext(ctx,
			`UPDATE lighting_equipment
			    SET equipment_status = $1
			  WHERE CURRENT OF lighting_equipment_cursor`,
			ds.LightingEquipmentStatusDeleted,
		)
		if err != nil {
			return err
		}
	}

	if _, err = tx.ExecContext(ctx, `CLOSE lighting_equipment_cursor`); err != nil {
		return err
	}
	return tx.Commit()
}
