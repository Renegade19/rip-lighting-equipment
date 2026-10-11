package ds

// LightingUser — проектировщик освещения: подбирает светильники и считает освещённость
type LightingUser struct {
	UserID   int    `gorm:"column:user_id;primaryKey;size:32"`
	Login    string `gorm:"column:login;type:varchar(50);not null;uniqueIndex:uq_lighting_user_login"`
	FullName string `gorm:"column:full_name;type:varchar(100);not null"`
}

func (LightingUser) TableName() string {
	return "lighting_users"
}
