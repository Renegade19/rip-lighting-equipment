package main

import (
	"lighting-equipment/internal/app/ds"
	"lighting-equipment/internal/app/dsn"

	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Миграции выполняются отдельной командой (go run ./cmd/migrate),
// при старте приложения схема БД не меняется
func main() {
	_ = godotenv.Load()

	db, err := gorm.Open(postgres.Open(dsn.FromEnv()), &gorm.Config{})
	if err != nil {
		log.Fatalf("не удалось подключиться к БД: %v", err)
	}

	// Тип-перечисление статусов светильника: черновик / опубликован / удалён
	err = db.Exec(`
		DO $$ BEGIN
			CREATE TYPE lighting_equipment_status AS ENUM ('черновик', 'опубликован', 'удалён');
		EXCEPTION WHEN duplicate_object THEN NULL;
		END $$`).Error
	if err != nil {
		log.Fatalf("не удалось создать тип lighting_equipment_status: %v", err)
	}

	err = db.AutoMigrate(
		&ds.LightingUser{},
		&ds.LightingEquipment{},
		&ds.LightingEquipmentLike{},
	)
	if err != nil {
		log.Fatalf("ошибка миграций: %v", err)
	}

	log.Info("Миграции выполнены успешно: lighting_users, lighting_equipment, lighting_equipment_likes")
}
