package api

import (
	"context"

	"lighting-equipment/internal/app/config"
	"lighting-equipment/internal/app/dsn"
	"lighting-equipment/internal/app/handler"
	"lighting-equipment/internal/app/redis"
	"lighting-equipment/internal/app/repository"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// StartServer собирает веб-сервис: конфигурация, подключение к БД, Minio и Redis,
// обработчики и маршруты. Страниц сервис не отдаёт — только JSON и Swagger
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

	redisClient, err := redis.New(context.Background(), cfg.RedisAddress(), cfg.Redis.Password)
	if err != nil {
		log.Fatalf("не удалось подключиться к Redis: %v", err)
	}
	defer redisClient.Close()

	h := handler.New(repo, redisClient, cfg)

	r := gin.Default()
	h.RegisterRoutes(r)

	if err := r.Run(":" + cfg.ServicePort); err != nil {
		log.Fatalf("сервер не запустился: %v", err)
	}
}
