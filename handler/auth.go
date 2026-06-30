package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/solaris-soft/goforge/service"
	"github.com/solaris-soft/goforge/view"
)

type AuthService interface {
	EmailRegister(ctx context.Context, cmd service.EmailRegisterCommand) (service.User, error)
	VerifyEmail(ctx context.Context, cmd service.VerifyEmailCommand) error
}

type AuthHandler struct {
	AuthService AuthService
}

func NewAuthHandler(svc AuthService) AuthHandler {
	return AuthHandler{
		svc,
	}
}

// Renders the sign up page
func (h AuthHandler) SignUp(w http.ResponseWriter, r *http.Request) error {
	return view.SignupPage().Render(r.Context(), w)
}

// Processes a sign up request
func (h AuthHandler) PostSignUp(w http.ResponseWriter, r *http.Request) error {
	err := r.ParseForm()
	if err != nil {
		return err
	}

	email := r.Form.Get("email")
	password := r.Form.Get("password")
	name := r.Form.Get("name")

	_, err = h.AuthService.
		EmailRegister(
			r.Context(),
			service.EmailRegisterCommand{
				Name:     name,
				Email:    email,
				Password: password,
			},
		)
	if err != nil {
		return err
	}
	http.Redirect(w, r, "/signin", http.StatusSeeOther)
	return nil
}

// Processes an email verification request
func (h AuthHandler) Verify(w http.ResponseWriter, r *http.Request) error {
	// Get the token
	token := r.URL.Query().Get("token")
	if token == "" {
		return errors.New("token not provided")
	}

	// Verify token
	err := h.AuthService.VerifyEmail(r.Context(), service.VerifyEmailCommand{Token: token})
	if err != nil {
		return err
	}

	// Return success
	return view.VerifyEmailPage(
		view.VerifyEmailErrors{},
	).
		Render(r.Context(), w)
}
