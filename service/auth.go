package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/solaris-soft/goforge/store"
)

// User is a service representation of a user
type User struct {
	Name  string
	Email string
}

// Mailer is a client capable of sending emails
type Mailer interface {
	Send(
		ctx context.Context,
		to []string,
		subject string,
		html string,
	) error
}

// TxBeginner is a type that can begin pgx transactions
type TxBeginner interface {
	Begin(context.Context) (pgx.Tx, error)
}

type Auth struct {
	queries *store.Queries
	txdb    TxBeginner
	mailer  Mailer
	appURL  string
}

func NewAuthService(queries *store.Queries, txdb TxBeginner, mailer Mailer, appURL string) Auth {
	return Auth{
		queries: queries,
		txdb:    txdb,
		mailer:  mailer,
		appURL:  appURL,
	}
}

// EmailRegister registers a user to the application as an email password provider type account
func (a Auth) EmailRegister(ctx context.Context, cmd EmailRegisterCommand) (User, error) {
	if err := cmd.Validate(); err != nil {
		return User{}, err
	}

	passwordHash, err := hashPassword(cmd.Password)
	if err != nil {
		return User{}, err
	}

	verificationToken, err := generateToken()
	if err != nil {
		return User{}, err
	}

	var user store.User

	// Complete registration process as transaction to ensure all components complete
	tx, err := a.txdb.Begin(ctx)
	if err != nil {
		return User{}, err
	}
	defer tx.Rollback(ctx)

	// Create the user
	db := a.queries.WithTx(tx)
	createdUser, err := db.CreateUser(ctx, store.CreateUserParams{
		Name: pgtype.Text{
			String: cmd.Name,
			Valid:  true,
		},
		PrimaryEmail: pgtype.Text{
			String: cmd.Email,
			Valid:  true,
		},
	})
	if err != nil {
		return User{}, err
	}

	// Create the user's account with email/password provider
	_, err = db.CreateAccount(ctx, store.CreateAccountParams{
		UserID:   createdUser.ID,
		Provider: "email",
		PasswordHash: pgtype.Text{
			String: passwordHash,
			Valid:  true,
		},
	})
	if err != nil {
		return User{}, err
	}

	// Create an email verification token record
	_, err = db.CreateEmailVerification(ctx, store.CreateEmailVerificationParams{
		UserID:    createdUser.ID,
		TokenHash: hashToken(verificationToken),
		ExpiresAt: pgtype.Timestamp{
			Time:  time.Now().Add(24 * time.Hour),
			Valid: true,
		},
	})
	if err != nil {
		return User{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return User{}, err
	}
	user = createdUser

	// Send email verification request
	err = a.sendRegisterEmail(ctx, user, verificationToken)
	if err != nil {
		return User{}, err
	}

	return User{
			Name:  user.Name.String,
			Email: user.PrimaryEmail.String,
		},
		nil
}

// VerifyEmail marks the user for a valid email verification token as verified.
func (a Auth) VerifyEmail(ctx context.Context, cmd VerifyEmailCommand) error {
	token := strings.TrimSpace(cmd.Token)
	if token == "" {
		return FieldErrors{"token": "Token required"}
	}

	verification, err := a.queries.GetEmailByToken(ctx, hashToken(token))
	if err != nil {
		return err
	}
	user, err := a.queries.GetUserById(ctx, verification.UserID)
	if err != nil {
		return err
	}
	return a.queries.MarkEmailVerified(ctx, user.PrimaryEmail)
}

// Generates the URL for verifying an email.
func (a Auth) generateEmailVerificationURL(token string) string {
	return fmt.Sprintf(
		"%s/verify?token=%s",
		strings.TrimRight(a.appURL, "/"),
		token,
	)
}

// sendRegisterEmail sends an email for a user to verify their email
func (a Auth) sendRegisterEmail(ctx context.Context, user store.User, verificationToken string) error {
	type RegisterEmailData struct {
		Name            string
		VerificationURL string
	}

	tmpl, err := template.ParseFiles("view/emails/register.html")
	if err != nil {
		return err
	}
	var body bytes.Buffer
	err = tmpl.Execute(&body, RegisterEmailData{
		Name:            user.Name.String,
		VerificationURL: a.generateEmailVerificationURL(verificationToken),
	})
	if err != nil {
		return err
	}

	return a.mailer.Send(
		ctx,
		[]string{user.PrimaryEmail.String},
		"Welcome – please confirm email",
		body.String(),
	)
}

// hashPassword converts a raw password string into a hash
func hashPassword(password string) (hash string, err error) {
	return argon2id.CreateHash(password, argon2id.DefaultParams)
}

// generateToken generates a random 32 byte token
func generateToken() (string, error) {
	b := make([]byte, 32)

	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
}

// hashToken converts a raw 32 byte token into a hash for storage
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
