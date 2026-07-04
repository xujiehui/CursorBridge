package apperrors

import (
	"errors"
	"fmt"
	"net/http"
)

type Code string

const (
	ErrInvalidSystemSetting     Code = "ErrInvalidSystemSetting"
	ErrCursorAccountUnavailable Code = "ErrCursorAccountUnavailable"
	ErrByokChannelRateLimited   Code = "ErrByokChannelRateLimited"
	ErrByokChannelNotAvailable  Code = "ErrByokChannelNotAvailable"
	ErrInvalidBidiAppendPayload Code = "ErrInvalidBidiAppendPayload"
	ErrInvalidRequest           Code = "ErrInvalidRequest"
	ErrNotFound                 Code = "ErrNotFound"
	ErrProxyNotRunning          Code = "ErrProxyNotRunning"
	ErrUpstream                 Code = "ErrUpstream"
)

type AppError struct {
	Code       Code   `json:"code"`
	Message    string `json:"message"`
	HTTPStatus int    `json:"status"`
}

func (e *AppError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func New(code Code, message string, status int) *AppError {
	return &AppError{Code: code, Message: message, HTTPStatus: status}
}

func Wrap(code Code, message string, status int, err error) *AppError {
	if err == nil {
		return New(code, message, status)
	}
	return New(code, message+": "+err.Error(), status)
}

func BadRequest(message string) *AppError {
	return New(ErrInvalidRequest, message, http.StatusBadRequest)
}

func NotFound(message string) *AppError {
	return New(ErrNotFound, message, http.StatusNotFound)
}

func Status(err error) int {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.HTTPStatus
	}
	return http.StatusInternalServerError
}

func Public(err error) *AppError {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return New(ErrInvalidSystemSetting, "internal server error", http.StatusInternalServerError)
}
