package repository

import "lighting-equipment/internal/app/ds"

// GetPublishedLightingEquipment — список для плитки: только опубликованные,
// фильтр по диапазону мощности (Вт) выполняется на сервере
func (r *Repository) GetPublishedLightingEquipment(minPowerWatt, maxPowerWatt float64) []ds.LightingEquipment {
	result := make([]ds.LightingEquipment, 0, len(r.lightingEquipment))
	for _, equipment := range r.lightingEquipment {
		if equipment.EquipmentStatus != ds.LightingEquipmentStatusPublished {
			continue
		}
		if equipment.PowerWatt >= minPowerWatt && equipment.PowerWatt <= maxPowerWatt {
			result = append(result, equipment)
		}
	}
	return result
}

// GetLightingEquipmentForFeed — светильник для ленты.
// equipmentID == 0 — лента открыта с панели вкладок, показываем первый опубликованный.
// next == true — следующий опубликованный после указанного id, после последнего — снова первый
func (r *Repository) GetLightingEquipmentForFeed(equipmentID int, next bool) (ds.LightingEquipment, bool) {
	var first *ds.LightingEquipment
	for i := range r.lightingEquipment {
		equipment := &r.lightingEquipment[i]
		if equipment.EquipmentStatus != ds.LightingEquipmentStatusPublished {
			continue
		}
		if first == nil {
			first = equipment
		}
		if next && equipment.EquipmentID > equipmentID {
			return *equipment, true
		}
		if !next && equipment.EquipmentID == equipmentID {
			return *equipment, true
		}
	}
	if first != nil && (next || equipmentID == 0) {
		return *first, true
	}
	return ds.LightingEquipment{}, false
}

// GetDraftLightingEquipment — единственный светильник в статусе "черновик"
func (r *Repository) GetDraftLightingEquipment() (ds.LightingEquipment, bool) {
	for _, equipment := range r.lightingEquipment {
		if equipment.EquipmentStatus == ds.LightingEquipmentStatusDraft {
			return equipment, true
		}
	}
	return ds.LightingEquipment{}, false
}
