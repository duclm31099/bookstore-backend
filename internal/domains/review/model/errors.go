package model

import (
	"errors"
	"fmt"
)

// Error codes
const (
	ErrCodeReviewNotFound  = "REV001"
	ErrCodeAlreadyReviewed = "REV002"
	ErrCodeNotEligible     = "REV003"
	ErrCodeCannotEdit      = "REV004"
	ErrCodeCannotDelete    = "REV005"
	ErrCodeInvalidRating   = "REV006"
	ErrCodeContentTooShort = "REV007"
	ErrCodeContentTooLong  = "REV008"
	ErrCodeTooManyImages   = "REV009"
	ErrCodeUnauthorized    = "REV010"
)

// Errors
var (
	ErrReviewNotFound  = errors.New("review not found")
	ErrAlreadyReviewed = errors.New("already reviewed this book")
	ErrNotEligible     = errors.New("not eligible to review")
	ErrCannotEdit      = errors.New("cannot edit review after 7 days")
	ErrCannotDelete    = errors.New("cannot delete review after 30 days")
	ErrUnauthorized    = errors.New("unauthorized to perform this action")
)

// ReviewError custom error type
type ReviewError struct {
	Code    string
	Message string
	Err     error
}

func (e *ReviewError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Error constructors
func NewReviewNotFoundError() *ReviewError {
	return &ReviewError{
		Code:    ErrCodeReviewNotFound,
		Message: "Review not found",
		Err:     ErrReviewNotFound,
	}
}

func NewAlreadyReviewedError() *ReviewError {
	return &ReviewError{
		Code:    ErrCodeAlreadyReviewed,
		Message: "You have already reviewed this book",
		Err:     ErrAlreadyReviewed,
	}
}

func NewNotEligibleError(reason string) *ReviewError {
	return &ReviewError{
		Code:    ErrCodeNotEligible,
		Message: fmt.Sprintf("Not eligible to review: %s", reason),
		Err:     ErrNotEligible,
	}
}

func NewCannotEditError() *ReviewError {
	return &ReviewError{
		Code:    ErrCodeCannotEdit,
		Message: "Cannot edit review after 7 days of creation",
		Err:     ErrCannotEdit,
	}
}

func NewCannotDeleteError() *ReviewError {
	return &ReviewError{
		Code:    ErrCodeCannotDelete,
		Message: "Cannot delete review after 30 days of creation",
		Err:     ErrCannotDelete,
	}
}
func NewUnauthorizedError(message string) *ReviewError {
	return &ReviewError{
		Code:    ErrCodeCannotEdit,
		Message: message,
		Err:     ErrCannotEdit,
	}
}
