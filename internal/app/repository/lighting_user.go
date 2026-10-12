package repository

import (
	"errors"

	"lighting-equipment/internal/app/ds"

	"gorm.io/gorm"
)

// POST регистрации: создаёт проектировщика. Логин уникален — индекс uq_lighting_user_login
func (r *Repository) CreateLightingUser(login, passwordHash, fullName string) (*ds.LightingUser, error) {
	user := ds.LightingUser{
		Login:        login,
		FullName:     fullName,
		PasswordHash: passwordHash,
	}

	err := r.db.Create(&user).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return nil, ErrLoginAlreadyTaken
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}
