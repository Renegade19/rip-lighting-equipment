package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

// Ограничения на загружаемые файлы
const (
	maxEquipmentImageBytes = 10 << 20  // 10 МБ
	maxEquipmentVideoBytes = 100 << 20 // 100 МБ
	// Тело запроса добавления: оба файла и текстовые поля формы
	maxCreateRequestBytes = maxEquipmentImageBytes + maxEquipmentVideoBytes + (1 << 20)
)

// Допустимые типы файлов и расширения, с которыми они сохраняются в Minio
var (
	equipmentImageTypes = map[string]string{
		"image/jpeg": ".jpg",
		"image/png":  ".png",
		"image/webp": ".webp",
		"image/gif":  ".gif",
	}
	equipmentVideoTypes = map[string]string{
		"video/mp4":  ".mp4",
		"video/webm": ".webm",
	}
)

// lightingEquipmentMedia — проверенный файл, готовый к загрузке в Minio
type lightingEquipmentMedia struct {
	header      *multipart.FileHeader
	contentType string
	// Имя файла в Minio: сгенерировано сервером, только латиница, цифры и дефисы
	fileName string
}

// Определяет тип файла по содержимому: заголовку Content-Type
// и расширению из запроса доверять нельзя
func detectMediaType(head []byte) string {
	contentType := http.DetectContentType(head)
	// Стандартная проверка узнаёт не все разновидности mp4 — добавляем проверку
	// по блоку ftyp. У QuickTime (.mov) блок такой же, но браузеры его не играют
	if contentType == "application/octet-stream" && len(head) >= 12 &&
		string(head[4:8]) == "ftyp" && string(head[8:12]) != "qt  " {
		return "video/mp4"
	}
	return contentType
}

// Генерирует имя файла на латинице. От исходного имени не берётся ничего:
// расширение определяется по типу содержимого
func generateMediaFileName(kind, extension string) (string, error) {
	suffix := make([]byte, 4)
	if _, err := rand.Read(suffix); err != nil {
		return "", err
	}
	return fmt.Sprintf("lighting-equipment-%s-%d-%s%s",
		kind, time.Now().UnixMilli(), hex.EncodeToString(suffix), extension), nil
}

// Проверяет файл из формы: размер и тип по содержимому.
// kind — "image" или "video", title — название файла для сообщений об ошибках
func validateLightingEquipmentMedia(header *multipart.FileHeader, kind, title string, allowedTypes map[string]string, maxBytes int64) (*lightingEquipmentMedia, error) {
	if header.Size == 0 {
		return nil, fmt.Errorf("%s: файл пустой", title)
	}
	if header.Size > maxBytes {
		return nil, fmt.Errorf("%s: файл больше %d МБ", title, maxBytes>>20)
	}

	file, err := header.Open()
	if err != nil {
		return nil, fmt.Errorf("%s: не удалось прочитать файл", title)
	}
	defer file.Close()

	// Тип файла определяется по его первым 512 байтам
	head := make([]byte, 512)
	n, err := io.ReadFull(file, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		return nil, fmt.Errorf("%s: не удалось прочитать файл", title)
	}

	contentType := detectMediaType(head[:n])
	extension, allowed := allowedTypes[contentType]
	if !allowed {
		return nil, fmt.Errorf("%s: тип файла %s не поддерживается", title, contentType)
	}

	fileName, err := generateMediaFileName(kind, extension)
	if err != nil {
		return nil, err
	}
	return &lightingEquipmentMedia{header: header, contentType: contentType, fileName: fileName}, nil
}

// Загружает проверенный файл в Minio
func (h *Handler) uploadLightingEquipmentMedia(ctx context.Context, media *lightingEquipmentMedia) error {
	file, err := media.header.Open()
	if err != nil {
		return err
	}
	defer file.Close()

	return h.Repository.UploadLightingEquipmentMedia(ctx, media.fileName, file, media.header.Size, media.contentType)
}
