package config

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
)

// Config — настройки приложения из переменных окружения (.env)
type Config struct {
	ServicePort string

	MinioEndpoint  string
	MinioPort      string
	MinioUseSSL    bool
	MinioBucket    string
	MinioAccessKey string
	MinioSecretKey string

	JWT   JWTConfig
	Redis RedisConfig
}

// JWTConfig — ключ подписи токенов и срок их жизни
type JWTConfig struct {
	SecretKey string
	ExpiresIn time.Duration
}

// RedisConfig — подключение к Redis, где хранится чёрный список токенов
type RedisConfig struct {
	Host     string
	Port     string
	Password string
}

func New() *Config {
	// Файла .env может не быть — тогда берутся значения по умолчанию
	_ = godotenv.Load()

	jwtExpiresIn, err := time.ParseDuration(env("JWT_EXPIRES_IN", "1h"))
	if err != nil || jwtExpiresIn <= 0 {
		log.Warn("JWT_EXPIRES_IN должен быть длительностью вида 1h или 30m, используется 1h")
		jwtExpiresIn = time.Hour
	}

	return &Config{
		ServicePort:    env("PORT", "3000"),
		MinioEndpoint:  env("MINIO_ENDPOINT", "localhost"),
		MinioPort:      env("MINIO_PORT", "9000"),
		MinioUseSSL:    env("MINIO_USE_SSL", "false") == "true",
		MinioBucket:    env("MINIO_BUCKET", "lighting-equipment"),
		MinioAccessKey: env("MINIO_ACCESS_KEY", "minioadmin"),
		MinioSecretKey: env("MINIO_SECRET_KEY", "minioadmin"),
		JWT: JWTConfig{
			SecretKey: env("JWT_SECRET_KEY", "lumicalc-dev-secret"),
			ExpiresIn: jwtExpiresIn,
		},
		Redis: RedisConfig{
			Host:     env("REDIS_HOST", "localhost"),
			Port:     env("REDIS_PORT", "6379"),
			Password: env("REDIS_PASSWORD", ""),
		},
	}
}

// RedisAddress — адрес Redis в виде host:port
func (c *Config) RedisAddress() string {
	return c.Redis.Host + ":" + c.Redis.Port
}

// MinioAddress — адрес S3 API Minio в виде host:port, по нему сервис загружает файлы
func (c *Config) MinioAddress() string {
	return c.MinioEndpoint + ":" + c.MinioPort
}

// MediaURL собирает публичный адрес файла в Minio по его имени из БД.
// В БД хранится только имя файла, адрес для клиента собирается при ответе.
// Если файла нет (nil или пустое имя), адреса тоже нет
func (c *Config) MediaURL(fileName *string) *string {
	if fileName == nil || *fileName == "" {
		return nil
	}
	protocol := "http"
	if c.MinioUseSSL {
		protocol = "https"
	}
	url := fmt.Sprintf("%s://%s:%s/%s/%s", protocol, c.MinioEndpoint, c.MinioPort, c.MinioBucket, *fileName)
	return &url
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
