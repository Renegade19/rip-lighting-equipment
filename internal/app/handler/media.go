package handler

import (
	"fmt"
	"net/http"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Фото и видео по умолчанию хранятся вместе с иконками на SSR-сервере.
// Подставляются, если в БД пустое поле url либо файл по ссылке недоступен
const (
	DefaultImageURL = "/images/default-equipment.svg"
	DefaultVideoURL = "/videos/default-equipment.mp4"
)

const mediaCheckTTL = time.Minute

type mediaCheck struct {
	ok        bool
	checkedAt time.Time
}

var (
	mediaClient = &http.Client{Timeout: time.Second}
	mediaChecks sync.Map // url -> mediaCheck
)

// Проверяет запросом HEAD, что файл по ссылке существует.
// Результат запоминается на минуту, чтобы не опрашивать Minio на каждый запрос страницы
func isMediaReachable(url string) bool {
	if cached, found := mediaChecks.Load(url); found {
		if check := cached.(mediaCheck); time.Since(check.checkedAt) < mediaCheckTTL {
			return check.ok
		}
	}

	ok := false
	if response, err := mediaClient.Head(url); err == nil {
		response.Body.Close()
		ok = response.StatusCode == http.StatusOK
	}
	mediaChecks.Store(url, mediaCheck{ok: ok, checkedAt: time.Now()})
	return ok
}

// Возвращает рабочий адрес изображения либо изображение по умолчанию
func resolveImageURL(imageURL *string) string {
	if imageURL == nil || *imageURL == "" || !isMediaReachable(*imageURL) {
		return DefaultImageURL
	}
	return *imageURL
}

// Возвращает рабочий адрес видео либо видео по умолчанию
func resolveVideoURL(videoURL *string) string {
	if videoURL == nil || *videoURL == "" || !isMediaReachable(*videoURL) {
		return DefaultVideoURL
	}
	return *videoURL
}

var mediaExtension = regexp.MustCompile(`^\.[a-z0-9]{1,5}$`)

// Генерирует ключ объекта Minio на латинице для выбранного файла.
// От имени файла берётся только расширение
func generateMediaKey(kind, fileName, defaultExtension string) string {
	extension := strings.ToLower(path.Ext(fileName))
	if !mediaExtension.MatchString(extension) {
		extension = defaultExtension
	}
	return fmt.Sprintf("lighting-equipment-%s-%d%s", kind, time.Now().UnixMilli(), extension)
}
