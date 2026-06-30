package handler

import (
	"context"
	"net/http"

	"github.com/solaris-soft/goforge/service"
	"github.com/solaris-soft/goforge/view"
)

type AuthService interface {
	EmailRegister(ctx context.Context, cmd service.EmailRegisterCommand) (service.User, error)
}

type AuthHandler struct {
	AuthService AuthService
}

func NewAuthHandler(svc AuthService) AuthHandler {
	return AuthHandler{
		svc,
	}
}

func (h AuthHandler) SignUp(w http.ResponseWriter, r *http.Request) error {
	return view.SignupPage().Render(r.Context(), w)
}

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
