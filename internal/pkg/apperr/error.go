package apperr

import (
	"errors"
	"net/http"
	"strings"

	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/constant"
	"gorm.io/gorm"
)

type Error struct {
	Code       constant.ResCode
	Message    string
	HTTPStatus int
	Cause      error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Cause == nil {
		return e.Message
	}
	if e.Message == "" {
		return e.Cause.Error()
	}
	return e.Message + ": " + e.Cause.Error()
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func New(code constant.ResCode) *Error {
	return &Error{
		Code:       code,
		Message:    constant.DefaultErrorMessage(code),
		HTTPStatus: constant.DefaultHTTPStatus(code),
	}
}

func Wrap(code constant.ResCode, cause error) *Error {
	return New(code).WithCause(cause)
}

func (e *Error) WithHTTPStatus(status int) *Error {
	if e == nil {
		return nil
	}
	e.HTTPStatus = status
	return e
}

func (e *Error) WithCause(cause error) *Error {
	if e == nil {
		return nil
	}
	e.Cause = cause
	return e
}

func As(err error) (*Error, bool) {
	if err == nil {
		return nil, false
	}
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}

func FromCode(code constant.ResCode) *Error {
	return New(code)
}

func Validation() *Error {
	return FromCode(constant.CommonBadRequest)
}

func FromError(err error) *Error {
	if err == nil {
		return FromCode(constant.CommonInternal)
	}
	if appErr, ok := As(err); ok {
		return appErr
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return FromCode(constant.CommonNotFound)
	}

	msg := strings.TrimSpace(err.Error())
	if msg == "" {
		return FromCode(constant.CommonInternal)
	}
	return FromCode(constant.CommonInternal).WithCause(err)
}

func FromHTTPStatus(httpStatus int) *Error {
	code := constant.CommonBadRequest
	switch {
	case httpStatus == http.StatusTooManyRequests:
		code = constant.CommonServiceUnavailable
	case httpStatus == http.StatusUnauthorized:
		code = constant.CommonUnauthorized
	case httpStatus == http.StatusForbidden:
		code = constant.CommonForbidden
	case httpStatus == http.StatusMethodNotAllowed:
		code = constant.CommonMethodNotAllowed
	case httpStatus == http.StatusServiceUnavailable:
		code = constant.CommonServiceUnavailable
	case httpStatus == http.StatusNotFound:
		code = constant.CommonNotFound
	case httpStatus == http.StatusConflict:
		code = constant.CommonConflict
	case httpStatus == http.StatusBadRequest || httpStatus == 0:
		code = constant.CommonBadRequest
	case httpStatus >= http.StatusInternalServerError:
		code = constant.CommonInternal
	}

	return FromCode(code)
}
