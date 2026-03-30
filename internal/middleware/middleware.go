package middleware

import (
	"Production_Ready_HTTP_Server_Code/pkg/logger"
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var(
	// Метрики Prometheus
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_request_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_request_duration_seconds",
			Help: "Duration of HTTP requests",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
)

type responseWriter struct {
	http.ResponseWriter 
	statusCode int
}

type rateLimiter struct {
	requests []time.Time
}

// Chain применяет несколько middleware последовательно
func Chain(middlewares ...mux.MiddlewareFunc) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		for i := len(middlewares)-1; i >= 0; i-- {
			next = middlewares[i](next)
		}
		return next
	}
}

// Recovery middleware восстанавливает после паники
func Recovery(log *logger.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					// Логируем панику
					log.Error().
						Interface("panic", err).
						Str("path", r.URL.Path).
						Str("method", r.Method).
						Msg("recovered from panic")
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// Logger middleware логирует все запросы
func Logger(log *logger.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Создаем обертку для ResponseWriter, чтобы захватить статус код
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Добавляем request ID в контекст
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = generateRequestID()
			}
			ctx := context.WithValue(r.Context(), "request_id", requestID)
			r = r.WithContext(ctx)

			// Обрабатываем запрос
			next.ServeHTTP(wrapped, r)

			// Логируем после обработки
			duration := time.Since(start)
			log.Info().
				Str("request_id", requestID).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Int("status", wrapped.statusCode).
				Dur("duration", duration).
				Str("ip", r.RemoteAddr).
				Str("user_agent", r.UserAgent()).
				Msg("request completed")
		})
	}
}

// Metrics middleware собирает метрики Prometheus
func Metrics() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			wrapped := &responseWriter{ResponseWriter: w}

			next.ServeHTTP(wrapped, r)

			duration := time.Since(start)
			path := getRoutePath(r)

			httpRequestsTotal.WithLabelValues(
				r.Method, path, http.StatusText(wrapped.statusCode),
			).Inc()

			httpRequestDuration.WithLabelValues(
				r.Method, path,
			).Observe(duration.Seconds())
		})
	}
}

// CORS middleware настраивает CORS
func CORS(allowedOrigins []string) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Проверяем разрешен ли origin
			allowed := false 
			for _, o := range allowedOrigins {
				if o == "*" || o == origin {
					allowed = true 
					break
				}
			}

			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allowed-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			// Handle preflight
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return 
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RateLimit middleware ограничивает количество запросов
func RateLimit(requests int, duration time.Duration) mux.MiddlewareFunc {
	// Простая имплементация на карте, в продакшене используйте Redis
	visitors := make(map[string]*rateLimiter)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := strings.Split(r.RemoteAddr, ":")[0]

			limiter, exists := visitors[ip]
			if !exists {
				limiter = &rateLimiter {
					requests: make([]time.Time, 0, requests),
				}
				visitors[ip] = limiter
			}

			if !limiter.allow(requests, duration) {
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return 
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Auth middleware проверяет JWT токен
func Auth(jwtSecret string) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := r.Header.Get("Authorization")
			if token == "" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return 
			}

			// Удаляем "Bearer " из токена
			token = strings.TrimPrefix(token, "Bearer")

			// Валидация токена (упрощенно)
			if token == "" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return 
			}

			// В реальном проекте здесь проверка JWT
            // claims, err := validateJWT(token, jwtSecret)

			next.ServeHTTP(w, r)
		})
	}
}

// Вспомогательные типы и функции
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code 
	rw.ResponseWriter.WriteHeader(code)
}

func (rl *rateLimiter) allow(maxRequests int, duration time.Duration) bool {
	now := time.Now()
	cutoff := now.Add(-duration)

	// Удаляем старые запросы
	
}