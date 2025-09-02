package websocket

import (
	"fmt"

	"github.com/katonium/kubegame/backend/generated"
)

// GameError represents a custom error with an associated error code.
type GameError struct {
	Code    generated.ErrorEventDataCode
	Message string
	Details map[string]interface{}
}

func (e *GameError) Error() string {
	return e.Message
}

// Custom error constructors for each error code
func NewInvalidRequestError(message string) *GameError {
	return &GameError{
		Code:    generated.INVALIDREQUEST,
		Message: message,
	}
}

func NewGameNotRunningError(message string) *GameError {
	return &GameError{
		Code:    generated.GAMENOTRUNNING,
		Message: message,
	}
}

func NewClusterNotReadyError(message string) *GameError {
	return &GameError{
		Code:    generated.CLUSTERNOTREADY,
		Message: message,
	}
}

func NewInvalidSessionError(message string) *GameError {
	return &GameError{
		Code:    generated.INVALIDSESSION,
		Message: message,
	}
}

func NewNodeNotFoundError(message string) *GameError {
	return &GameError{
		Code:    generated.NODENOTFOUND,
		Message: message,
	}
}

func NewPodNotFoundError(message string) *GameError {
	return &GameError{
		Code:    generated.PODNOTFOUND,
		Message: message,
	}
}

func NewInsufficientResourcesError(message string) *GameError {
	return &GameError{
		Code:    generated.INSUFFICIENTRESOURCES,
		Message: message,
	}
}

func NewCrossSessionAccessError(message string) *GameError {
	return &GameError{
		Code:    generated.CROSSSESSIONACCESS,
		Message: message,
	}
}

// Helper function to create GameError with details
func NewGameErrorWithDetails(code generated.ErrorEventDataCode, message string, details map[string]interface{}) *GameError {
	return &GameError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// Helper to wrap regular errors as InvalidRequestError
func WrapAsInvalidRequest(err error, context string) *GameError {
	message := err.Error()
	if context != "" {
		message = fmt.Sprintf("%s: %s", context, message)
	}
	return NewInvalidRequestError(message)
}
