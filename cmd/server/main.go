package main

import (
	"Production_Ready_HTTP_Server_Code/internal/config"
	"Production_Ready_HTTP_Server_Code/internal/server"
	"Production_Ready_HTTP_Server_Code/pkg/logger"
)

func main() {
	// Загружаем конфигурацию
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	// Создаем логгер
	log := logger.New(cfg.ServiceName, cfg.Environment)
	
	// Создаем и запускаем сервер
	src := server.New(cfg, log)

	if err := src.Start(); err != nil {
		log.Fatal(err, "server failed")
	}
}