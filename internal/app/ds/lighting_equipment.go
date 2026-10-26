package ds

import "time"

// Статусы светильника
const (
	LightingEquipmentStatusDraft     = "черновик"
	LightingEquipmentStatusPublished = "опубликован"
	LightingEquipmentStatusDeleted   = "удалён"
)

// LightingEquipment — услуга: модель светильника.
// Два поля по предметной области: мощность (Вт) и угол рассеивания (°)
type LightingEquipment struct {
	EquipmentID      int     `gorm:"column:equipment_id;primaryKey;size:32"`
	EquipmentName    string  `gorm:"column:equipment_name;type:varchar(150);not null"`
	ShortDescription *string `gorm:"column:short_description;type:varchar(500)"`
	EquipmentStatus  string  `gorm:"column:equipment_status;type:lighting_equipment_status;not null;default:'черновик'"`
	// Имена файлов в бакете Minio (сами файлы хранятся в Minio).
	// Обязательные: фото и видео задаются при создании, по кнопке «Далее»
	ImageName string `gorm:"column:image_name;type:varchar(255);not null"`
	VideoName string `gorm:"column:video_name;type:varchar(255);not null"`

	// Поля по предметной области
	PowerWatt    *float64 `gorm:"column:power_watt;type:numeric(6,2)"`
	BeamAngleDeg *int     `gorm:"column:beam_angle_deg;type:integer"`

	CreatedAt time.Time `gorm:"column:created_at;type:timestamp;not null;default:CURRENT_TIMESTAMP"`
	// Дата формирования — проставляется при публикации
	FormedAt *time.Time `gorm:"column:formed_at;type:timestamp"`

	// Создатель. У проектировщика не более одного черновика — частичный уникальный индекс.
	// Каскадное удаление запрещено: RESTRICT
	CreatorID int          `gorm:"column:creator_id;size:32;not null;uniqueIndex:uq_draft_per_creator,where:equipment_status = 'черновик'"`
	Creator   LightingUser `gorm:"foreignKey:CreatorID;references:UserID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT"`
}

func (LightingEquipment) TableName() string {
	return "lighting_equipment"
}

// LightingEquipmentWithLikes — светильник вместе с вычисляемыми полями.
// Их считает БД в том же запросе: количество лайков — подзапросом
// к таблице связи м-м, признаки 0/1 — относительно текущего проектировщика
type LightingEquipmentWithLikes struct {
	LightingEquipment
	LikesTotal int64 `gorm:"column:likes_total"`
	// 1 — светильник создан текущим проектировщиком
	IsCreatedByCurrentUser int `gorm:"column:is_created_by_current_user"`
	// 1 — текущий проектировщик поставил этому светильнику лайк
	IsLikedByCurrentUser int `gorm:"column:is_liked_by_current_user"`
}
