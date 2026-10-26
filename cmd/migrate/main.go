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

	// В БД, созданной до перехода на имена файлов, столбцы назывались image_url и video_url
	// и хранили полные адреса. Переименовываем их и оставляем в значениях только имя файла
	if err = renameMediaColumns(db); err != nil {
		log.Fatalf("не удалось переименовать столбцы файлов: %v", err)
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

func renameMediaColumns(db *gorm.DB) error {
	migrator := db.Migrator()
	columns := map[string]string{"image_url": "image_name", "video_url": "video_name"}
	for oldName, newName := range columns {
		if !migrator.HasColumn(&ds.LightingEquipment{}, oldName) {
			continue
		}
		if err := migrator.RenameColumn(&ds.LightingEquipment{}, oldName, newName); err != nil {
			return err
		}
		// http://localhost:9000/lighting-equipment/file.jpg -> file.jpg
		err := db.Exec("UPDATE lighting_equipment SET " + newName + " = regexp_replace(" + newName + ", '^.*/', '') WHERE " + newName + " LIKE '%/%'").Error
		if err != nil {
			return err
		}
	}
	return nil
}
