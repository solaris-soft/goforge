package service

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/solaris-soft/goforge/store"
)

type fakeMailer struct {
	to      []string
	subject string
	html    string
}

func (f *fakeMailer) Send(ctx context.Context, to []string, subject, html string) error {
	f.to = to
	f.subject = subject
	f.html = html
	return nil
}

func TestEmailRegister(t *testing.T) {
	ctx := context.Background()
	db := testDB(t)
	chdirProjectRoot(t)

	mailer := &fakeMailer{}
	auth := NewAuthService(store.New(db), db, mailer, "http://localhost:3000")

	got, err := auth.EmailRegister(ctx, EmailRegisterCommand{
		Name:     "Jane",
		Email:    "jane@example.com",
		Password: "secret123",
	})
	if err != nil {
		t.Fatalf("EmailRegister returned error: %v", err)
	}
	if got != (User{Name: "jane", Email: "jane@example.com"}) {
		t.Fatalf("EmailRegister = %#v", got)
	}

	q := store.New(db)
	user, err := q.GetUserByEmail(ctx, pgtype.Text{String: "jane@example.com", Valid: true})
	if err != nil {
		t.Fatalf("GetUserByEmail: %v", err)
	}
	if user.Name.String != "jane" {
		t.Fatalf("user name = %q", user.Name.String)
	}

	accounts, err := q.GetUserAccounts(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetUserAccounts: %v", err)
	}
	if len(accounts) != 1 || accounts[0].Provider != "email" {
		t.Fatalf("accounts = %#v", accounts)
	}
	passwordHash := accounts[0].PasswordHash.String
	if !strings.HasPrefix(passwordHash, "$argon2id$") || strings.Contains(passwordHash, "secret123") {
		t.Fatalf("bad password hash: %q", passwordHash)
	}

	token := tokenFromEmail(t, mailer.html)
	if _, err := q.GetEmailByToken(ctx, hashToken(token)); err != nil {
		t.Fatalf("GetEmailByToken: %v", err)
	}
	if err := auth.VerifyEmail(ctx, VerifyEmailCommand{Token: token}); err != nil {
		t.Fatalf("VerifyEmail: %v", err)
	}
	user, err = q.GetUserByEmail(ctx, pgtype.Text{String: "jane@example.com", Valid: true})
	if err != nil {
		t.Fatalf("GetUserByEmail after verify: %v", err)
	}
	if !user.EmailVerified {
		t.Fatal("email was not verified")
	}
}

func tokenFromEmail(t *testing.T, html string) string {
	t.Helper()
	const marker = "/verify?token="
	start := strings.Index(html, marker)
	if start < 0 {
		t.Fatalf("verification link not found in email: %s", html)
	}
	token := html[start+len(marker):]
	if end := strings.IndexByte(token, '"'); end >= 0 {
		token = token[:end]
	}
	if token == "" {
		t.Fatal("empty verification token")
	}
	return token
}

func testDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	ctx := context.Background()
	db, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)

	_, err = db.Exec(ctx, `TRUNCATE email_verification_tokens, accounts, users RESTART IDENTITY CASCADE`)
	if err != nil {
		t.Fatalf("reset test db: %v", err)
	}
	return db
}

func chdirProjectRoot(t *testing.T) {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Dir(filepath.Dir(file))
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
}
