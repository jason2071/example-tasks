package utils

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AppError struct {
	ErrorCode    string `json:"error_code"`
	ErrorMessage error  `json:"error_message"`
}

func (e *AppError) Error() string {
	return fmt.Sprintf("[%s] %s", e.ErrorCode, e.ErrorMessage)
}

// นิยาม Error ชุดมาตรฐานของระบบ
var (
	ErrNotFound = &AppError{
		ErrorCode:    "E001",
		ErrorMessage: errors.New("resource not found"),
	}
	ErrInvalidRequest = &AppError{
		ErrorCode:    "E002",
		ErrorMessage: errors.New("invalid request data"),
	}
	ErrDuplicateEntry = &AppError{
		ErrorCode:    "E003",
		ErrorMessage: errors.New("task with the same title already exists"),
	}
	ErrInternalServer = &AppError{
		ErrorCode:    "E500",
		ErrorMessage: errors.New("internal server error"),
	}
	Success = &AppError{
		ErrorCode:    "SUCCESS",
		ErrorMessage: nil,
	}
)

// func switch error code
func GetAppErrorByCode(code string) *AppError {
	switch code {
	case "E001":
		return ErrNotFound
	case "E002":
		return ErrInvalidRequest
	case "E003":
		return ErrDuplicateEntry
	case "E500":
		return ErrInternalServer
	case "SUCCESS":
		return Success
	default:
		return &AppError{
			ErrorCode:    "UNKNOWN",
			ErrorMessage: errors.New("unknown error"),
		}
	}
}

// HandleError writes a JSON error response for *AppError and reports
// whether it handled the error (true = response written).
func HandleError(c *gin.Context, err error) bool {
	if appErr, ok := err.(*AppError); ok {
		var statusCode int
		switch appErr.ErrorCode {
		case "E001":
			statusCode = http.StatusNotFound
		case "E002", "E003":
			statusCode = http.StatusBadRequest
		case "E500":
			statusCode = http.StatusInternalServerError
		default:
			statusCode = http.StatusInternalServerError
		}

		c.JSON(statusCode, gin.H{
			"error": appErr.ErrorMessage.Error(),
			"code":  appErr.ErrorCode,
			"path":  c.Request.URL.Path,
		})
		return true
	}
	return false
}
