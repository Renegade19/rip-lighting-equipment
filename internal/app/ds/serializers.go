package ds

import "time"

// Сериализаторы — структуры, которые описывают JSON веб-сервиса.
// Модели наружу напрямую не отдаются: сериализатор ответа решает, какие поля
// увидит клиент, сериализатор запроса — какие поля клиент может прислать.
// Системных полей (ид, статус, создатель, даты) в сериализаторах запросов нет,
// поэтому изменить их с клиента нельзя

// MediaURLFunc собирает публичный адрес файла в Minio по имени файла из БД
type MediaURLFunc func(fileName string) string

// Формат дат в ответах: местное время сервера, как оно хранится в столбцах timestamp
const timestampLayout = "2006-01-02T15:04:05"

func formatTimestamp(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.Format(timestampLayout)
	return &formatted
}

// ---------- Ответы: домен "светильники" ----------

// LightingEquipmentTileSerializer — краткий сериализатор: карточка в списке (плитка)
type LightingEquipmentTileSerializer struct {
	EquipmentID   int      `json:"equipmentId"`
	EquipmentName string   `json:"equipmentName"`
	ImageURL      string   `json:"imageUrl"`
	PowerWatt     *float64 `json:"powerWatt"`
	LikesTotal    int64    `json:"likesTotal"`
	// 1 — светильник создан текущим проектировщиком, 0 — другим
	IsCreatedByCurrentUser int `json:"isCreatedByCurrentUser"`
}

func NewLightingEquipmentTileSerializer(equipment LightingEquipmentWithLikes, mediaURL MediaURLFunc) LightingEquipmentTileSerializer {
	return LightingEquipmentTileSerializer{
		EquipmentID:            equipment.EquipmentID,
		EquipmentName:          equipment.EquipmentName,
		ImageURL:               mediaURL(equipment.ImageName),
		PowerWatt:              equipment.PowerWatt,
		LikesTotal:             equipment.LikesTotal,
		IsCreatedByCurrentUser: equipment.IsCreatedByCurrentUser,
	}
}

// LightingEquipmentFeedSerializer — подробный сериализатор: светильник в ленте.
// Вложенная сериализация: создатель отдаётся объектом, а не числом creatorId
type LightingEquipmentFeedSerializer struct {
	EquipmentID      int                    `json:"equipmentId"`
	EquipmentName    string                 `json:"equipmentName"`
	ShortDescription *string                `json:"shortDescription"`
	ImageURL         string                 `json:"imageUrl"`
	VideoURL         string                 `json:"videoUrl"`
	PowerWatt        *float64               `json:"powerWatt"`
	BeamAngleDeg     *int                   `json:"beamAngleDeg"`
	FormedAt         *string                `json:"formedAt"`
	Creator          LightingUserSerializer `json:"creator"`
	LikesTotal       int64                  `json:"likesTotal"`
	// 1 — текущий проектировщик лайкнул светильник, 0 — нет
	IsLikedByCurrentUser int `json:"isLikedByCurrentUser"`
}

func NewLightingEquipmentFeedSerializer(equipment LightingEquipmentWithLikes, mediaURL MediaURLFunc) LightingEquipmentFeedSerializer {
	return LightingEquipmentFeedSerializer{
		EquipmentID:          equipment.EquipmentID,
		EquipmentName:        equipment.EquipmentName,
		ShortDescription:     equipment.ShortDescription,
		ImageURL:             mediaURL(equipment.ImageName),
		VideoURL:             mediaURL(equipment.VideoName),
		PowerWatt:            equipment.PowerWatt,
		BeamAngleDeg:         equipment.BeamAngleDeg,
		FormedAt:             formatTimestamp(equipment.FormedAt),
		Creator:              NewLightingUserSerializer(equipment.Creator),
		LikesTotal:           equipment.LikesTotal,
		IsLikedByCurrentUser: equipment.IsLikedByCurrentUser,
	}
}

// LightingEquipmentSerializer — светильник создателя целиком:
// ответ на получение черновика, добавление и публикацию
type LightingEquipmentSerializer struct {
	EquipmentID      int      `json:"equipmentId"`
	EquipmentName    string   `json:"equipmentName"`
	ShortDescription *string  `json:"shortDescription"`
	EquipmentStatus  string   `json:"equipmentStatus"`
	ImageURL         string   `json:"imageUrl"`
	VideoURL         string   `json:"videoUrl"`
	PowerWatt        *float64 `json:"powerWatt"`
	BeamAngleDeg     *int     `json:"beamAngleDeg"`
	CreatedAt        *string  `json:"createdAt"`
	FormedAt         *string  `json:"formedAt"`
	CreatorID        int      `json:"creatorId"`
}

func NewLightingEquipmentSerializer(equipment LightingEquipment, mediaURL MediaURLFunc) LightingEquipmentSerializer {
	return LightingEquipmentSerializer{
		EquipmentID:      equipment.EquipmentID,
		EquipmentName:    equipment.EquipmentName,
		ShortDescription: equipment.ShortDescription,
		EquipmentStatus:  equipment.EquipmentStatus,
		ImageURL:         mediaURL(equipment.ImageName),
		VideoURL:         mediaURL(equipment.VideoName),
		PowerWatt:        equipment.PowerWatt,
		BeamAngleDeg:     equipment.BeamAngleDeg,
		CreatedAt:        formatTimestamp(&equipment.CreatedAt),
		FormedAt:         formatTimestamp(equipment.FormedAt),
		CreatorID:        equipment.CreatorID,
	}
}

// LightingEquipmentLikesSerializer — ответ на установку и отмену лайка
type LightingEquipmentLikesSerializer struct {
	EquipmentID          int   `json:"equipmentId"`
	LikesTotal           int64 `json:"likesTotal"`
	IsLikedByCurrentUser int   `json:"isLikedByCurrentUser"`
}

// ---------- Запросы: домен "светильники" ----------

// PublishLightingEquipmentSerializer — тело PUT публикации.
// Указатели отличают "поле не прислали" от нулевого значения
type PublishLightingEquipmentSerializer struct {
	ShortDescription *string  `json:"shortDescription"`
	PowerWatt        *float64 `json:"powerWatt"`
	BeamAngleDeg     *int     `json:"beamAngleDeg"`
}

// SetLightingEquipmentLikeSerializer — тело POST лайка: 1 ставит лайк, 0 отменяет
type SetLightingEquipmentLikeSerializer struct {
	EquipmentLike *int `json:"equipmentLike"`
}

// ---------- Домен "проектировщики" ----------

// LightingUserSerializer — проектировщик в ответах. Хэша пароля здесь нет
type LightingUserSerializer struct {
	UserID   int    `json:"userId"`
	Login    string `json:"login"`
	FullName string `json:"fullName"`
}

func NewLightingUserSerializer(user LightingUser) LightingUserSerializer {
	return LightingUserSerializer{
		UserID:   user.UserID,
		Login:    user.Login,
		FullName: user.FullName,
	}
}

// RegisterLightingUserSerializer — тело POST регистрации
type RegisterLightingUserSerializer struct {
	Login    string `json:"login" example:"n.fotonov"`
	Password string `json:"password" example:"lumicalc-demo"`
	FullName string `json:"fullName" example:"Фотонов Н. К."`
}

// LoginLightingUserSerializer — тело POST аутентификации
type LoginLightingUserSerializer struct {
	Login    string `json:"login" example:"a.parahnevich"`
	Password string `json:"password" example:"lumicalc-demo"`
}

// LoginLightingUserResponseSerializer — ответ аутентификации: JWT и сам проектировщик.
// Токен передаётся в остальных запросах заголовком Authorization: Bearer {accessToken}
type LoginLightingUserResponseSerializer struct {
	AccessToken string                 `json:"accessToken"`
	TokenType   string                 `json:"tokenType" example:"Bearer"`
	ExpiresIn   int64                  `json:"expiresIn" example:"3600"` // секунд до истечения токена
	User        LightingUserSerializer `json:"user"`
}

// ---------- Общие ответы ----------

// StatusSerializer — ответ без данных: ошибка либо подтверждение действия
type StatusSerializer struct {
	Status      string `json:"status" example:"error"`
	Description string `json:"description"`
}
