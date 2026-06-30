package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/solaris-soft/goforge/service"
)

type mockAuthService struct {
	emailRegisterFn func(ctx context.Context, cmd service.EmailRegisterCommand) (service.User, error)
}

func (m mockAuthService) EmailRegister(ctx context.Context, cmd service.EmailRegisterCommand) (service.User, error) {
	if m.emailRegisterFn == nil {
		return service.User{}, nil
	}
	return m.emailRegisterFn(ctx, cmd)
}

// Test the sign up page
func TestAuthHandlerSignUp(t *testing.T) {
	h := NewAuthHandler(mockAuthService{})

	t.Run("It returns the SignUp Page", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/signup", nil)
		w := httptest.NewRecorder()

		if err := h.SignUp(w, r); err != nil {
			t.Fatalf("SignUp returned error: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		if !strings.Contains(w.Body.String(), "Signup") {
			t.Fatalf("expected body to contain %q, got %q", "Sign up", w.Body.String())
		}
	})

	t.Run("A user can sign up with a valid email and password", func(t *testing.T) {
		form := url.Values{
			"name":     {"Josh"},
			"email":    {"josh@gmail.com"},
			"password": {"password123"},
		}

		r := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader(form.Encode()))
		w := httptest.NewRecorder()

		if err := h.PostSignUp(w, r); err != nil {
			t.Fatalf("PostSignUp returned error: %v", err)
		}

		if w.Code != http.StatusSeeOther {
			t.Fatalf("expected status %d, got %d",
				http.StatusSeeOther, w.Code)
		}

		if got := w.Header().Get("Location"); got != "/signin" {
			t.Fatalf("expected redirect to /signin, got %q", got)
		}
	})
}
