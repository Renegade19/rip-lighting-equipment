package repository

import (
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
)

// Загружает фото или видео светильника в бакет Minio под указанным именем
func (r *Repository) UploadLightingEquipmentMedia(ctx context.Context, fileName string, file io.Reader, fileSize int64, contentType string) error {
	_, err := r.minio.PutObject(ctx, r.minioBucket, fileName, file, fileSize, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("не удалось сохранить файл %s в Minio: %w", fileName, err)
	}
	return nil
}

// Удаляет файл из бакета. Нужен, когда файлы уже загружены,
// а запись о светильнике в БД создать не удалось
func (r *Repository) RemoveLightingEquipmentMedia(ctx context.Context, fileName string) error {
	return r.minio.RemoveObject(ctx, r.minioBucket, fileName, minio.RemoveObjectOptions{})
}
