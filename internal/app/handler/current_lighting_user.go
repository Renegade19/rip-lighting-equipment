package handler

import "sync"

// В лабораторной №3 авторизации ещё нет, поэтому создатель во всех методах
// один и тот же — проектировщик с этим ид
const fixedLightingUserID int = 1

var (
	currentLightingUserID   int
	currentLightingUserOnce sync.Once
)

// getCurrentLightingUserID — функция-singleton: значение создаётся один раз
// при первом обращении, все методы получают одного и того же проектировщика.
// Когда в лабораторной №4 появится авторизация, изменится только эта функция,
// а вызовы в обработчиках останутся прежними
func getCurrentLightingUserID() int {
	currentLightingUserOnce.Do(func() {
		currentLightingUserID = fixedLightingUserID
	})
	return currentLightingUserID
}
