package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/solaris-soft/goforge/store"
)

type mockQuerier struct {
	createAccount func(context.Context, store.CreateAccountParams) (store.Account, error)
	createUser    func(context.Context, store.CreateUserParams) (store.User, error)
}

func (m *mockQuerier) CreateAccount(ctx context.Context, arg store.CreateAccountParams) (store.Account, error) {
	if m.createAccount == nil {
		return store.Account{}, errors.New("unexpected CreateAccount call")
	}
	return m.createAccount(ctx, arg)
}

func (m *mockQuerier) CreateUser(ctx context.Context, arg store.CreateUserParams) (store.User, error) {
	if m.createUser == nil {
		return store.User{}, errors.New("unexpected CreateUser call")
	}
	return m.createUser(ctx, arg)
}

func (m *mockQuerier) GetUserAccounts(ctx context.Context, userID pgtype.UUID) ([]store.Account, error) {
	return nil, errors.New("unexpected GetUserAccounts call")
}

func (m *mockQuerier) GetUserById(ctx context.Context, id pgtype.UUID) (store.User, error) {
	return store.User{}, errors.New("unexpected GetUserById call")
}

func (m *mockQuerier) GetUserByUsername(ctx context.Context, username string) (store.User, error) {
	return store.User{}, errors.New("unexpected GetUserByUsername call")
}

func TestEmailRegister(t *testing.T) {
	expected := User{Name: "jane", Email: "jane@example.com"}
	called := false

	mockStore := &mockQuerier{
		createUser: func(ctx context.Context, arg store.CreateUserParams) (store.User, error) {
			if arg.Username != expected.Name {
				t.Fatalf("username = %q, want %q", arg.Name.String, expected.Name)
			}
			if !arg.PrimaryEmail.Valid || arg.PrimaryEmail.String != expected.Email {
				t.Fatalf("primary email = %#v, want %q", arg.PrimaryEmail, expected.Email)
			}
			return store.User{Name: expected.Name, PrimaryEmail: arg.PrimaryEmail}, nil
		},
		createAccount: func(ctx context.Context, arg store.CreateAccountParams) (store.Account, error) {
			called = true
			if arg.Provider != "email" {
				t.Fatalf("provider = %q, want email", arg.Provider)
			}
			if !strings.HasPrefix(arg.PasswordHash.String, "$argon2id$") {
				t.Fatalf("password hash = %q, want argon2id hash", arg.PasswordHash.String)
			}
			if strings.Contains(arg.PasswordHash.String, "secret123") {
				t.Fatal("password hash contains plaintext password")
			}
			return store.Account{}, nil
		},
	}
	auth := Auth{
		inTx: func(ctx context.Context, fn func(AuthStore) error) error {
			return fn(mockStore)
		},
	}

	got, err := auth.EmailRegister(context.Background(), EmailRegisterCommand{
		Name:     expected.Name,
		Email:    expected.Email,
		Password: "secret123",
	})
	if err != nil {
		t.Fatalf("EmailRegister returned error: %v", err)
	}
	if !called {
		t.Fatal("CreateUser was not called")
	}
	if got != expected {
		t.Fatalf("EmailRegister = %#v, want %#v", got, expected)
	}
}
