package errors

import "net/http"

type AppError struct {
	Code       string
	Message    string
	StatusCode int
}

func (e *AppError) Error() string { return e.Message }

var (
	ErrInternal          = &AppError{"INTERNAL_ERROR", "Erro interno", http.StatusInternalServerError}
	ErrValidation        = &AppError{"INVALID_REQUEST", "Dados inválidos", http.StatusBadRequest}
	ErrUnauthorized      = &AppError{"UNAUTHORIZED", "Token ausente, expirado ou inválido", http.StatusUnauthorized}
	ErrForbidden         = &AppError{"FORBIDDEN", "Acesso negado", http.StatusForbidden}
	ErrNotFound          = &AppError{"NOT_FOUND", "Recurso não encontrado", http.StatusNotFound}
	ErrCharacterNotIdle  = &AppError{"CHARACTER_NOT_IDLE", "Personagem não está idle", http.StatusForbidden}
	ErrCharacterNotFound = &AppError{"NOT_FOUND", "Personagem não encontrado", http.StatusNotFound}
	ErrHuntNotFound      = &AppError{"HUNT_NOT_FOUND", "Hunt não encontrada", http.StatusNotFound}
	ErrHuntAlreadyActive = &AppError{"HUNT_ALREADY_ACTIVE", "Já existe uma hunt em andamento", http.StatusConflict}
	ErrInvalidDuration   = &AppError{"INVALID_DURATION", "Duração deve ser entre 1 e 360 minutos", http.StatusUnprocessableEntity}
	ErrLevelTooLow       = &AppError{"LEVEL_TOO_LOW", "Level insuficiente para esta hunt", http.StatusUnprocessableEntity}
	ErrNoActiveHunt      = &AppError{"NO_ACTIVE_HUNT", "Nenhuma hunt em andamento", http.StatusNotFound}
	ErrSessionNotFound   = &AppError{"SESSION_NOT_FOUND", "Sessão não encontrada", http.StatusNotFound}
)
