package ds

// Статусы светильника
const (
	LightingEquipmentStatusDraft     = "черновик"
	LightingEquipmentStatusPublished = "опубликован"
	LightingEquipmentStatusDeleted   = "удалён"
)

// LightingEquipment — услуга: модель светильника.
// Два поля по предметной области: мощность (Вт) и угол рассеивания (°)
type LightingEquipment struct {
	EquipmentID      int
	EquipmentName    string
	ShortDescription string
	EquipmentStatus  string
	ImageKey         string // ключ изображения в Minio
	VideoKey         string // ключ видео в Minio
	PowerWatt        float64
	BeamAngleDeg     int
	LikedUserIDs     []int // id пользователей, поставивших лайк
}
