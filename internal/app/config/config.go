package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
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
}

func New() *Config {
	// Файла .env может не быть — тогда берутся значения по умолчанию
	_ = godotenv.Load()

	return &Config{
		ServicePort:    env("PORT", "3000"),
		MinioEndpoint:  env("MINIO_ENDPOINT", "localhost"),
		MinioPort:      env("MINIO_PORT", "9000"),
		MinioUseSSL:    env("MINIO_USE_SSL", "false") == "true",
		MinioBucket:    env("MINIO_BUCKET", "lighting-equipment"),
		MinioAccessKey: env("MINIO_ACCESS_KEY", "minioadmin"),
		MinioSecretKey: env("MINIO_SECRET_KEY", "minioadmin"),
	}
}

// MinioAddress — адрес S3 API Minio в виде host:port, по нему сервис загружает файлы
func (c *Config) MinioAddress() string {
	return c.MinioEndpoint + ":" + c.MinioPort
}

// MediaURL собирает публичный адрес файла в Minio по его имени из БД.
// В БД хранится только имя файла, адрес для клиента собирается при ответе
func (c *Config) MediaURL(fileName string) string {
	protocol := "http"
	if c.MinioUseSSL {
		protocol = "https"
	}
	return fmt.Sprintf("%s://%s:%s/%s/%s", protocol, c.MinioEndpoint, c.MinioPort, c.MinioBucket, fileName)
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
