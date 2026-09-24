package services

import "errors"

var (
	ErrInvalidInput          = errors.New("invalid input")
	ErrNoText                = errors.New("PDF contains no extractable text")
	ErrProviderNotConfigured = errors.New("provider is not configured")
)

const InsufficientInformation = "I don't have enough information to answer that."
