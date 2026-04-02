package server

import (
	"Production_Ready_HTTP_Server_Code/internal/config"
	"Production_Ready_HTTP_Server_Code/internal/handlers"
	"Production_Ready_HTTP_Server_Code/internal/middleware"
	"Production_Ready_HTTP_Server_Code/pkg/logger"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
)

// Server представляет HTTP сервер
type Server struct {
	config *config.Config
	logger *logger.Logger
	router *mux.Router
	http *http.Server
}

// New создает новый экземпляр сервера
func New(cfg *config.Config, log *logger.Logger) *Server {
	s := &Server{
		config: cfg,
		logger: log,
		router: mux.NewRouter(),
	}

	s.setupMiddleware()
	s.setupRoutes()

	return s
}

// setupMiddleware настраивает глобальные middleware
func (s *Server) setupMiddleware() {
	// Применяем middleware в правильном порядке
	s.router.Use(
		middleware.Recovery(s.logger),
		middleware.Logger(s.logger),
		middleware.Metrics(),
		middleware.CORS(s.config.CORSAllowedOrigins),
		middleware.RateLimit(s.config.RateLimitRequests, s.config.RateLimitDuration),
	)
}

// setupRoutes настраивает все маршруты
func (s *Server) setupRoutes() {
	// Создаем обработчики
	handlers := handlers.New(s.config, s.logger)

	// Регистрируем маршруты приложения
	handlers.RegisterRoutes(s.router)

	// Метрики Prometheus
	s.router.Handle("/metrics", promhttp.Handler().Methods("GET"))

	// Статические файлы (если есть)
	s.router.PathPrefix("/static/").Handler(
		http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))),
	)

	// 404 обработчик
	s.router.NotFoundHandler = http.HandlerFunc(s.notFoundHandler)
}

// notFoundHandler обрабатывает 404 ошибки
func (s *Server) notFoundHandler(w http.ResponseWriter, r *http.Request) {
	s.logger.Warn().
		Str("path", r.URL.Path).
		Str("method", r.Method).
		Msg("route not found")

	http.Error(w, "Not Found", http.StatusNotFound)
}

// Start запускает сервер
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%s", s.config.ServerPort)

	s.http = &http.Server{
		Addr: addr,
		Handler: s.router,
		ReadTimeout: s.config.ServerReadTimeout,
		WriteTimeout: s.config.ServerWriteTimeout,
		IdleTimeout: s.config.ServerIdleTimeout,
		ErrorLog: nil, // Используем наш логгер
	}

	s.logger.Info().
		Str("addr", addr).
		Str("env", s.config.Environment).
		Msg("starting server")

	// Запускаем сервер в горутине
	go func() {
		if err := s.http.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Fatal(err, "failed to start server")
		}
	}()

	// Ждем сигнал завершения
	return s.waitForShutdown()
}

// waitForShutdown ожидает сигнал завершения и gracefully shutdown
func (s *Server) waitForShutdown() error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<- quit

	s.logger.Info().Msg("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.http.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	s.logger.Info().Msg("server stopped gracefully")
	return nil 
}