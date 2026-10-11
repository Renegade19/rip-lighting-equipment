package main

import (
	"lighting-equipment/internal/api"

	log "github.com/sirupsen/logrus"
)

func main() {
	log.Info("Запуск приложения LumiCalc")
	api.StartServer()
	log.Info("Приложение LumiCalc остановлено")
}
