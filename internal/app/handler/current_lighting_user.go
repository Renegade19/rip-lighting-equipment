package handler

// В лабораторной №2 авторизации ещё нет, поэтому текущий проектировщик
// зафиксирован константой. Когда появится авторизация, изменится только
// эта функция, а вызовы в обработчиках останутся прежними
const currentLightingUserID int = 1

func getCurrentLightingUserID() int {
	return currentLightingUserID
}
