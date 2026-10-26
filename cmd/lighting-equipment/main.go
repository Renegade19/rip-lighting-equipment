package main

import (
	_ "lighting-equipment/docs"
	"lighting-equipment/internal/api"

	log "github.com/sirupsen/logrus"
)

//	@title			LumiCalc API
//	@version		1.0
//	@description	Веб-сервис расчёта освещённости помещения светодиодными лампами.
//	@description	Услуги — модели светильников, пользователи — проектировщики освещения.
//	@description	Гостю доступно только чтение. Методы создателя требуют JWT:
//	@description	выполните вход, нажмите Authorize и введите «Bearer {accessToken}».

//	@contact.name	Парахневич А.

//	@host		localhost:3000
//	@BasePath	/api
//	@schemes	http

//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				JWT из ответа POST /lighting-users/login в виде «Bearer {accessToken}»

func main() {
	log.Info("Запуск приложения LumiCalc")
	api.StartServer()
	log.Info("Приложение LumiCalc остановлено")
}
