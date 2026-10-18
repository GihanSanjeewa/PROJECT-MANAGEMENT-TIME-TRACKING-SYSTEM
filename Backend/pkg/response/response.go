package response

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"project-management-system/pkg/errcode"
)

type Meta struct {
	RequestID string    `json:"request_id,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type ErrorDetails struct {
	Code    errcode.ErrorCode `json:"code"`
	Message string            `json:"message"`
	Details interface{}       `json:"details,omitempty"`
}

type Envelope struct {
	Success bool          `json:"success"`
	Data    interface{}   `json:"data"`
	Error   *ErrorDetails `json:"error"`
	Meta    Meta          `json:"meta"`
}

func JSON(w http.ResponseWriter, statusCode int, data interface{}, requestID string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)

	env := Envelope{
		Success: statusCode >= 200 && statusCode < 300,
		Data:    data,
		Error:   nil,
		Meta: Meta{
			RequestID: requestID,
			Timestamp: time.Now().UTC(),
		},
	}
	_ = json.NewEncoder(w).Encode(env)
}

func Error(w http.ResponseWriter, err error, requestID string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	var appErr *errcode.AppError
	if errors.As(err, &appErr) {
		w.WriteHeader(appErr.HTTPStatus)
		env := Envelope{
			Success: false,
			Data:    nil,
			Error: &ErrorDetails{
				Code:    appErr.Code,
				Message: appErr.Message,
				Details: appErr.Details,
			},
			Meta: Meta{
				RequestID: requestID,
				Timestamp: time.Now().UTC(),
			},
		}
		_ = json.NewEncoder(w).Encode(env)
		return
	}

	// Internal Server Error (Never leak raw SQL or internals)
	w.WriteHeader(http.StatusInternalServerError)
	env := Envelope{
		Success: false,
		Data:    nil,
		Error: &ErrorDetails{
			Code:    errcode.ErrInternal,
			Message: "An internal server error occurred",
			Details: nil,
		},
		Meta: Meta{
			RequestID: requestID,
			Timestamp: time.Now().UTC(),
		},
	}
	_ = json.NewEncoder(w).Encode(env)
}
