package handlers

import (
	"Production_Ready_HTTP_Server_Code/internal/config"
	"Production_Ready_HTTP_Server_Code/pkg/logger"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// Handler содержит зависимости для обработчиков
type Handler struct {
	config *config.Config 
	logger *logger.Logger
}

// New создает новый экземпляр Handler
func New(cfg *config.Config, log *logger.Logger) *Handler {
	return &Handler{
		config: cfg,
		logger: log,
	}
}

// RegisterRoutes регистрирует все маршруты
func (h *Handler) RegisterRoutes(router *mux.Router) {
	// Health check endpoints
	router.HandleFunc("/health", h.HealthCheck).Methods("GET")
	router.HandleFunc("/ready", h.ReadinessCheck).Methods("GET")

	// API v1
	api := router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/users", h.GetUsers).Methods("GET")
	api.HandleFunc("/users/{id}", h.GetUser).Methods("GET")
	api.HandleFunc("/users", h.CreateUser).Methods("POST")
	api.HandleFunc("/users/{id}", h.UpdateUser).Methods("PUT")
	api.HandleFunc("/users/{id}", h.DeleteUser).Methods("DELETE")
}

// HealthCheck возвращает статус сервиса
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	h.respondWithJSON(w, http.StatusOK, map[string]string {
		"status": "healthy",
		"time": time.Now().UTC().String(),
	})
}

// ReadinessCheck проверяет готовность сервиса принимать трафик
func (h *Handler) ReadinessCheck(w http.ResponseWriter, r *http.Request) {
	// Здесь проверяем соединения с БД, Redis и т.д.
	h.respondWithJSON(w, http.StatusOK, map[string]string {
		"status": "ready",
	})
}

// GetUsers возвращает список пользователей
func (h *Handler) GetUsers(w http.ResponseWriter, r *http.Request) {
	// Получаем request ID из контекста
	requestID, _ := r.Context().Value("request_id").(string)

	// Логируем запрос
	h.logger.Debug().
		Str("request_id", requestID).
		Msg("getting users list")

	users := []map[string]interface{}{
		{"id": 1, "name": "John Doe", "email": "john@example.com"},
		{"id": 2, "name": "Jane Smith", "email": "jane@example.com"}, 
	}

	h.respondWithJSON(w, http.StatusOK, users)
}

// GetUser возвращает одного пользователя
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	user := map[string]interface{}{
		"id": id,
		"name": "John Doe",
		"email": "john@example.com",
	}

	h.respondWithJSON(w, http.StatusOK, user)
}

// CreateUser создает нового пользователя
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var user struct {
		Name string `json:"name"`
		Email string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return 
	}

	// Валидация
	if user.Name == "" || user.Email == "" {
		h.respondWithError(w, http.StatusBadRequest, "Name and email are required")
		return 
	}

	// Здесь создание пользователя в БД
	h.respondWithJSON(w, http.StatusCreated, user)
}

// UpdateUser обновляет пользователя
func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var user map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return 
	}

	user["id"] = id 

	h.respondWithJSON(w, http.StatusOK, user)
}

// DeleteUser удаляет пользователя
func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	h.logger.Info().Str("user_id", id).Msg("user deleted")

	h.respondWithJSON(w, http.StatusNoContent, nil)
}

// Вспомогательные функции для ответов
func (h *Handler) respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to marshal response")
		w.WriteHeader(http.StatusInternalServerError)
		return 
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func (h *Handler) respondWithError(w http.ResponseWriter, code int, message string) {
	h.respondWithJSON(w, code, map[string]string{"error": message})
}