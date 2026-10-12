package api

import (
	"lighting-equipment/internal/app/config"
	"lighting-equipment/internal/app/dsn"
	"lighting-equipment/internal/app/handler"
	"lighting-equipment/internal/app/repository"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// StartServer собирает веб-сервис: конфигурация, подключение к БД и Minio,
// обработчики и маршруты. Страниц сервис не отдаёт — только JSON
func StartServer() {
	cfg := config.New()

	repo, err := repository.New(repository.Settings{
		PostgresDSN:    dsn.FromEnv(),
		MinioAddress:   cfg.MinioAddress(),
		MinioAccessKey: cfg.MinioAccessKey,
		MinioSecretKey: cfg.MinioSecretKey,
		MinioBucket:    cfg.MinioBucket,
		MinioUseSSL:    cfg.MinioUseSSL,
	})
	if err != nil {
		log.Fatalf("не удалось подключиться к хранилищам: %v", err)
	}
	h := handler.New(repo, cfg)

	r := gin.Default()
	h.RegisterRoutes(r)

	if err := r.Run(":" + cfg.ServicePort); err != nil {
		log.Fatalf("сервер не запустился: %v", err)
	}
}
