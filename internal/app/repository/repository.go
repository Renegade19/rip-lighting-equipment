package repository

import "lighting-equipment/internal/app/ds"

// Repository — в лабораторной №1 данные хранятся прямо в коде (без БД):
// единая коллекция светильников с вложенными лайками.
// Фото и видео лежат в Minio, здесь только их ключи
type Repository struct {
	lightingEquipment []ds.LightingEquipment
}

func New() *Repository {
	return &Repository{
		lightingEquipment: []ds.LightingEquipment{
			{
				EquipmentID:      1,
				EquipmentName:    "SMART+ MATTER Filament Multicolor Classic Edison 40 4.8W E27",
				ShortDescription: "Умная филаментная лампа с поддержкой Matter и настройкой цвета, тёплый свет для жилых помещений.",
				EquipmentStatus:  ds.LightingEquipmentStatusPublished,
				ImageKey:         "vintage-1906-classic-a.jpg",
				VideoKey:         "smart-matter-classic-a.mp4",
				PowerWatt:        4.8,
				BeamAngleDeg:     300,
				LikedUserIDs:     []int{3, 7, 8, 12, 15, 19, 21, 24, 28, 31, 33, 36, 39},
			},
			{
				EquipmentID:      2,
				EquipmentName:    "Vintage 1906 Edison 28 Filament 4W 820 Gold E27",
				ShortDescription: "Филаментная лампа Edison с золотистой колбой, для акцентного декоративного освещения.",
				EquipmentStatus:  ds.LightingEquipmentStatusPublished,
				ImageKey:         "vintage-1906-edison-28-gold-e27.jpg",
				PowerWatt:        4,
				BeamAngleDeg:     300,
				LikedUserIDs:     []int{2, 7, 11, 15, 22, 29, 34},
			},
			{
				EquipmentID:      3,
				EquipmentName:    "Vintage 1906 Big Oval 10 Filament Magnetic 4W 818 Smoke E27",
				ShortDescription: "Большая овальная филаментная лампа с дымчатой колбой, магнитный стиль нити накала.",
				EquipmentStatus:  ds.LightingEquipmentStatusPublished,
				ImageKey:         "vintage-1906-big-oval-10-smoke-e27.jpg",
				PowerWatt:        4,
				BeamAngleDeg:     320,
				LikedUserIDs:     []int{5, 12, 18, 27},
			},
			{
				EquipmentID:      4,
				EquipmentName:    "Vintage 1906 Classic B 40 Filament DIM 4.8W 827 Clear E14",
				ShortDescription: "Диммируемая филаментная свеча с прозрачной колбой, цоколь E14, для декоративных светильников.",
				EquipmentStatus:  ds.LightingEquipmentStatusPublished,
				ImageKey:         "vintage-1906-classic-b40-clear-e14.jpg",
				PowerWatt:        4.8,
				BeamAngleDeg:     320,
				LikedUserIDs:     []int{1, 7, 9, 14, 15, 20, 23, 30, 35, 38},
			},
			{
				EquipmentID:      5,
				EquipmentName:    "Vintage 1906 Big Elipse 6 Filament Magnetic 4W 818 Smoke E27",
				ShortDescription: "Компактная эллиптическая филаментная лампа с дымчатой колбой, магнитный стиль нити накала.",
				EquipmentStatus:  ds.LightingEquipmentStatusDeleted,
				ImageKey:         "vintage-1906-big-elipse-6-smoke-e27.jpg",
				PowerWatt:        4,
				BeamAngleDeg:     320,
				LikedUserIDs:     []int{},
			},
			{
				EquipmentID:      6,
				EquipmentName:    "Vintage 1906 Edison 22 Filament 2.5W 824 Gold E27",
				ShortDescription: "Филаментная лампа Edison меньшего размера с золотистой колбой, тёплый декоративный свет.",
				EquipmentStatus:  ds.LightingEquipmentStatusDraft,
				ImageKey:         "vintage-1906-edison-22-gold-e27.jpg",
				PowerWatt:        2.5,
				BeamAngleDeg:     300,
				LikedUserIDs:     []int{},
			},
		},
	}
}
