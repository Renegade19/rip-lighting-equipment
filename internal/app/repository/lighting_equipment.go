package repository

import (
	"errors"
	"time"

	"lighting-equipment/internal/app/ds"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Базовый запрос чтения. Вычисляемые поля считает сама БД, в коде срезы не перебираются:
// количество лайков и признак "лайкнул текущий проектировщик" — подзапросами
// к таблице связи м-м, признак "создал текущий проектировщик" — выражением CASE
func (r *Repository) lightingEquipmentWithLikes(currentUserID int) *gorm.DB {
	likesTotal := r.db.Model(&ds.LightingEquipmentLike{}).
		Select("COUNT(*)").
		Where("lighting_equipment_likes.equipment_id = lighting_equipment.equipment_id")

	// Пара (проектировщик, светильник) уникальна, поэтому здесь получается 0 либо 1
	likedByCurrentUser := r.db.Model(&ds.LightingEquipmentLike{}).
		Select("COUNT(*)").
		Where("lighting_equipment_likes.equipment_id = lighting_equipment.equipment_id").
		Where("lighting_equipment_likes.user_id = ?", currentUserID)

	return r.db.Model(&ds.LightingEquipment{}).Select(
		"lighting_equipment.*, "+
			"(?) AS likes_total, "+
			"(?) AS is_liked_by_current_user, "+
			"CASE WHEN lighting_equipment.creator_id = ? THEN 1 ELSE 0 END AS is_created_by_current_user",
		likesTotal, likedByCurrentUser, currentUserID,
	)
}

// 1. GET списка: только опубликованные, фильтр по диапазону мощности выполняется в SQL.
// Граница nil означает, что с этой стороны диапазон не ограничен
func (r *Repository) GetPublishedLightingEquipment(currentUserID int, minPowerWatt, maxPowerWatt *float64) ([]ds.LightingEquipmentWithLikes, error) {
	query := r.lightingEquipmentWithLikes(currentUserID).
		Where("equipment_status = ?", ds.LightingEquipmentStatusPublished)
	if minPowerWatt != nil {
		query = query.Where("power_watt >= ?", *minPowerWatt)
	}
	if maxPowerWatt != nil {
		query = query.Where("power_watt <= ?", *maxPowerWatt)
	}

	lightingEquipment := []ds.LightingEquipmentWithLikes{}
	err := query.Order("equipment_id ASC").Find(&lightingEquipment).Error
	return lightingEquipment, err
}

// 2. GET ленты: БД возвращает сразу одну строку, только из опубликованных.
// next == false — светильник с указанным id.
// next == true  — следующий после id; сортировка сначала ставит строки с большим id,
// поэтому после последнего светильника лента возвращается к первому.
// Лента без id — это next от нуля, то есть первый опубликованный
func (r *Repository) GetLightingEquipmentForFeed(currentUserID, equipmentID int, next bool) (*ds.LightingEquipmentWithLikes, error) {
	query := r.lightingEquipmentWithLikes(currentUserID).
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
	err := query.Take(&equipment).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrLightingEquipmentNotFound
	}
	if err != nil {
		return nil, err
	}

	// Создатель нужен для вложенной сериализации
	if err = r.db.Take(&equipment.Creator, equipment.CreatorID).Error; err != nil {
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
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrLightingEquipmentNotFound
	}
	if err != nil {
		return nil, err
	}
	return &draft, nil
}

// 4. POST добавления: создаёт светильник в статусе "черновик".
// Системные поля (статус, создатель, дата создания) проставляет сервер.
// Второй черновик не даёт создать частичный уникальный индекс uq_draft_per_creator
func (r *Repository) CreateDraftLightingEquipment(creatorID int, equipmentName, imageName, videoName string) (*ds.LightingEquipment, error) {
	draft := ds.LightingEquipment{
		EquipmentName:   equipmentName,
		EquipmentStatus: ds.LightingEquipmentStatusDraft,
		ImageName:       imageName,
		VideoName:       videoName,
		CreatedAt:       time.Now(),
		CreatorID:       creatorID,
	}

	err := r.db.Omit(clause.Associations).Create(&draft).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return nil, ErrDraftAlreadyExists
	}
	if err != nil {
		return nil, err
	}
	return &draft, nil
}

// Находит неудалённый светильник и блокирует строку до конца транзакции.
// Менять светильник может только его создатель
func lockOwnLightingEquipment(tx *gorm.DB, creatorID, equipmentID int) (*ds.LightingEquipment, error) {
	var equipment ds.LightingEquipment
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("equipment_id = ? AND equipment_status <> ?", equipmentID, ds.LightingEquipmentStatusDeleted).
		Take(&equipment).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrLightingEquipmentNotFound
	}
	if err != nil {
		return nil, err
	}
	if equipment.CreatorID != creatorID {
		return nil, ErrForeignLightingEquipment
	}
	return &equipment, nil
}

// 5. PUT публикации: переход черновик -> опубликован, других переходов метод не делает.
// Одновременно заполняются описание и оба поля по теме, дату формирования ставит сервер
func (r *Repository) PublishLightingEquipment(creatorID, equipmentID int, shortDescription string, powerWatt float64, beamAngleDeg int) (*ds.LightingEquipment, error) {
	var published *ds.LightingEquipment
	err := r.db.Transaction(func(tx *gorm.DB) error {
		equipment, err := lockOwnLightingEquipment(tx, creatorID, equipmentID)
		if err != nil {
			return err
		}
		if equipment.EquipmentStatus != ds.LightingEquipmentStatusDraft {
			return ErrLightingEquipmentNotDraft
		}

		formedAt := time.Now()
		equipment.ShortDescription = &shortDescription
		equipment.PowerWatt = &powerWatt
		equipment.BeamAngleDeg = &beamAngleDeg
		equipment.EquipmentStatus = ds.LightingEquipmentStatusPublished
		equipment.FormedAt = &formedAt

		published = equipment
		return tx.Model(equipment).
			Select("ShortDescription", "PowerWatt", "BeamAngleDeg", "EquipmentStatus", "FormedAt").
			Updates(equipment).Error
	})
	if err != nil {
		return nil, err
	}
	return published, nil
}

// 6. DELETE: только логическое удаление — запись остаётся в таблице,
// меняется статус. Удалить можно черновик или опубликованный светильник
func (r *Repository) DeleteLightingEquipment(creatorID, equipmentID int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		equipment, err := lockOwnLightingEquipment(tx, creatorID, equipmentID)
		if err != nil {
			return err
		}
		return tx.Model(equipment).
			Update("EquipmentStatus", ds.LightingEquipmentStatusDeleted).Error
	})
}

// 7. POST лайка: liked == true ставит лайк, false — отменяет.
// Повторный запрос ничего не меняет: вторая такая же строка не появится
// из-за уникального индекса, а отмена несуществующего лайка удаляет ноль строк.
// Возвращает количество лайков светильника после изменения
func (r *Repository) SetLightingEquipmentLike(userID, equipmentID int, liked bool) (int64, error) {
	var likesTotal int64
	err := r.db.Transaction(func(tx *gorm.DB) error {
		// Лайкать можно только опубликованные светильники
		var publishedTotal int64
		err := tx.Model(&ds.LightingEquipment{}).
			Where("equipment_id = ? AND equipment_status = ?", equipmentID, ds.LightingEquipmentStatusPublished).
			Count(&publishedTotal).Error
		if err != nil {
			return err
		}
		if publishedTotal == 0 {
			return ErrLightingEquipmentNotFound
		}

		if liked {
			like := ds.LightingEquipmentLike{LikedByUserID: userID, LikedEquipmentID: equipmentID}
			err = tx.Omit(clause.Associations).
				Clauses(clause.OnConflict{DoNothing: true}).
				Create(&like).Error
		} else {
			err = tx.
				Where("user_id = ? AND equipment_id = ?", userID, equipmentID).
				Delete(&ds.LightingEquipmentLike{}).Error
		}
		if err != nil {
			return err
		}

		return tx.Model(&ds.LightingEquipmentLike{}).
			Where("equipment_id = ?", equipmentID).
			Count(&likesTotal).Error
	})
	return likesTotal, err
}
