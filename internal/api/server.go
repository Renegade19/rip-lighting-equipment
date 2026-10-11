package api

import (
	"lighting-equipment/internal/app/config"
	"lighting-equipment/internal/app/dsn"
	"lighting-equipment/internal/app/handler"
	"lighting-equipment/internal/app/repository"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// StartServer собирает приложение: конфигурация, подключение к БД,
// обработчики, шаблоны и статика
func StartServer() {
	cfg := config.New()

	repo, err := repository.New(dsn.FromEnv())
	if err != nil {
		log.Fatalf("не удалось подключиться к БД: %v", err)
	}
	h := handler.New(repo, cfg)

	r := gin.Default()

	// Шаблоны страниц и общая панель вкладок
	r.LoadHTMLGlob("templates/*")
	// Стили, иконки, фото и видео по умолчанию лежат на SSR-сервере,
	// фото и видео светильников — в Minio
	r.Static("/css", "./resources/css")
	r.Static("/images", "./resources/images")
	r.Static("/videos", "./resources/videos")

	h.RegisterRoutes(r)

	if err := r.Run(":" + cfg.ServicePort); err != nil {
		log.Fatalf("сервер не запустился: %v", err)
	}
}
