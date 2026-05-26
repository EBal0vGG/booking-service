package domain

import (
	"errors"
	"fmt"
)

// ErrorCode mirrors API error codes.
type ErrorCode string

const (
	ErrorInvalidRequest      ErrorCode = "INVALID_REQUEST"
	ErrorUnauthorized        ErrorCode = "UNAUTHORIZED"
	ErrorNotFound            ErrorCode = "NOT_FOUND"
	ErrorRoomNotFound        ErrorCode = "ROOM_NOT_FOUND"
	ErrorSlotNotFound        ErrorCode = "SLOT_NOT_FOUND"
	ErrorSlotAlreadyBooked   ErrorCode = "SLOT_ALREADY_BOOKED"
	ErrorSlotReserved        ErrorCode = "SLOT_RESERVED"
	ErrorSlotNotBooked       ErrorCode = "SLOT_NOT_BOOKED"
	ErrorBookingNotFound     ErrorCode = "BOOKING_NOT_FOUND"
	ErrorReservationNotFound ErrorCode = "RESERVATION_NOT_FOUND"
	ErrorWaitlistNotFound    ErrorCode = "WAITLIST_NOT_FOUND"
	ErrorWaitlistJoined      ErrorCode = "WAITLIST_ALREADY_JOINED"
	ErrorForbidden           ErrorCode = "FORBIDDEN"
	ErrorScheduleExists      ErrorCode = "SCHEDULE_EXISTS"
	ErrorInternalError       ErrorCode = "INTERNAL_ERROR"
)

// DomainError is returned by usecases and later mapped to HTTP responses.
type DomainError struct {
	Code    ErrorCode
	Message string
	Err     error
}

func (e *DomainError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return string(e.Code)
}

func (e *DomainError) Unwrap() error {
	if e.Err == nil {
		return nil
	}
	return e.Err
}

func NewDomainError(code ErrorCode, message string) error {
	return &DomainError{Code: code, Message: message}
}

func WrapDomainError(code ErrorCode, message string, err error) error {
	if err == nil {
		return NewDomainError(code, message)
	}
	return &DomainError{Code: code, Message: message, Err: err}
}

// AsDomainError is a helper to avoid boilerplate `errors.As` checks in handlers.
func AsDomainError(err error) (*DomainError, bool) {
	var de *DomainError
	if err == nil {
		return nil, false
	}
	if errors.As(err, &de) {
		return de, true
	}
	return nil, false
}
