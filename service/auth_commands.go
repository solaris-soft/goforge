package service

import (
	"net/mail"
	"strings"
)

// EmailRegisterCommand represents a command to
// create a new user with email / password login provider
type EmailRegisterCommand struct {
	Name     string
	Email    string
	Password string
}

func (e *EmailRegisterCommand) clean() {
	e.Name = strings.TrimSpace(strings.ToLower(e.Name))
	e.Email = strings.TrimSpace(strings.ToLower(e.Email))
}

// Validate validates that an EmailRegisterCommand contains
// valid parameters to create a new email / password user
func (e *EmailRegisterCommand) Validate() error {
	e.clean()
	fieldErrors := make(FieldErrors)
	if len(e.Password) < 8 {
		fieldErrors["password"] = "Password too short"
	}
	if len(e.Name) < 1 {
		fieldErrors["name"] = "Invalid name"
	}
	_, err := mail.ParseAddress(e.Email)
	if err != nil {
		fieldErrors["email"] = "Invalid email"
	}
	if len(fieldErrors) > 0 {
		return fieldErrors
	}
	return nil
}

// VerifyEmailCommand represents a command to verify
// an email address for a user account
type VerifyEmailCommand struct {
	Token string
}
