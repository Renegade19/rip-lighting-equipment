package repository

import (
	"errors"
	"log"
	"os"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Ошибки бизнес-логики. Обработчики переводят их в коды HTTP
var (
	ErrLightingEquipmentNotFound = errors.New("светильник не найден")
	ErrForeignLightingEquipment  = errors.New("светильник создан другим проектировщиком")
	ErrLightingEquipmentNotDraft = errors.New("опубликовать можно только светильник в статусе «черновик»")
	ErrDraftAlreadyExists        = errors.New("у проектировщика уже есть светильник в статусе «черновик»")
	ErrLoginAlreadyTaken         = errors.New("проектировщик с таким логином уже зарегистрирован")
	ErrLightingUserNotFound      = errors.New("проектировщик не найден")
)

// Settings — параметры подключения к хранилищам
type Settings struct {
	PostgresDSN string

	MinioAddress   string // host:port
	MinioAccessKey string
	MinioSecretKey string
	MinioBucket    string
	MinioUseSSL    bool
}

// Repository — доступ к данным: PostgreSQL через gORM, файлы — в Minio
type Repository struct {
	db          *gorm.DB
	minio       *minio.Client
	minioBucket string
}

func New(settings Settings) (*Repository, error) {
	db, err := gorm.Open(postgres.Open(settings.PostgresDSN), &gorm.Config{
		// Нарушение уникальности приходит как gorm.ErrDuplicatedKey
		TranslateError: true,
		// "Запись не найдена" — штатная ситуация (нет черновика, удалённый светильник),
		// в журнал как ошибка не пишется
		Logger: logger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true,
		}),
	})
	if err != nil {
		return nil, err
	}

	minioClient, err := minio.New(settings.MinioAddress, &minio.Options{
		Creds:  credentials.NewStaticV4(settings.MinioAccessKey, settings.MinioSecretKey, ""),
		Secure: settings.MinioUseSSL,
	})
	if err != nil {
		return nil, err
	}

	return &Repository{
		db:          db,
		minio:       minioClient,
		minioBucket: settings.MinioBucket,
	}, nil
}
