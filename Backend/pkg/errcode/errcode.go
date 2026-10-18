package errcode

type ErrorCode string

const (
	ErrValidationFailed   ErrorCode = "VALIDATION_FAILED"
	ErrUnauthorized       ErrorCode = "UNAUTHORIZED"
	ErrForbidden          ErrorCode = "FORBIDDEN"
	ErrNotFound           ErrorCode = "NOT_FOUND"
	ErrConflict           ErrorCode = "CONFLICT"
	ErrInternal           ErrorCode = "INTERNAL_ERROR"
	ErrBadRequest         ErrorCode = "BAD_REQUEST"
	ErrTimelineViolation  ErrorCode = "TIMELINE_VIOLATION"
	ErrCycleDetected      ErrorCode = "CYCLE_DETECTED"
	ErrTimerAlreadyActive ErrorCode = "TIMER_ALREADY_ACTIVE"
)

type AppError struct {
	Code       ErrorCode   `json:"code"`
	Message    string      `json:"message"`
	HTTPStatus int         `json:"-"`
	Details    interface{} `json:"details,omitempty"`
}

func (e *AppError) Error() string {
	return e.Message
}

func New(code ErrorCode, message string, httpStatus int, details interface{}) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
		Details:    details,
	}
}

func NewBadRequest(message string, details interface{}) *AppError {
	return New(ErrBadRequest, message, 400, details)
}

func NewUnauthorized(message string) *AppError {
	return New(ErrUnauthorized, message, 401, nil)
}

func NewForbidden(message string) *AppError {
	return New(ErrForbidden, message, 403, nil)
}

func NewNotFound(message string) *AppError {
	return New(ErrNotFound, message, 404, nil)
}

func NewConflict(message string, details interface{}) *AppError {
	return New(ErrConflict, message, 409, details)
}

func NewValidation(message string, details interface{}) *AppError {
	return New(ErrValidationFailed, message, 422, details)
}

func NewInternal(message string) *AppError {
	return New(ErrInternal, message, 500, nil)
}

func NewTimerAlreadyActive(message string) *AppError {
	return New(ErrTimerAlreadyActive, message, 409, nil)
}
