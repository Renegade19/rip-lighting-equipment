package ds

// LightingEquipmentLike — связь м-м "проектировщик — светильник":
// собственный первичный ключ и два внешних. Каскадное удаление запрещено (RESTRICT)
type LightingEquipmentLike struct {
	LikeID           int `gorm:"column:like_id;primaryKey;size:32"`
	LikedByUserID    int `gorm:"column:user_id;size:32;not null;uniqueIndex:uq_like_user_equipment"`
	LikedEquipmentID int `gorm:"column:equipment_id;size:32;not null;uniqueIndex:uq_like_user_equipment"`

	LikedByUser    LightingUser      `gorm:"foreignKey:LikedByUserID;references:UserID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT"`
	LikedEquipment LightingEquipment `gorm:"foreignKey:LikedEquipmentID;references:EquipmentID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT"`
}

func (LightingEquipmentLike) TableName() string {
	return "lighting_equipment_likes"
}
