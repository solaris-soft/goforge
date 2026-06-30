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

// AuthStore is the required data layer for the AuthService
type AuthStore interface {
	CreateAccount(ctx context.Context, arg store.CreateAccountParams) (store.Account, error)
	CreateUser(ctx context.Context, arg store.CreateUserParams) (store.User, error)
	GetUserAccounts(ctx context.Context, userID pgtype.UUID) ([]store.Account, error)
	GetUserById(ctx context.Context, id pgtype.UUID) (store.User, error)
	GetUserByEmail(ctx context.Context, email pgtype.Text) (store.User, error)
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
	inTx   func(context.Context, func(AuthStore) error) error
	mailer Mailer
	appURL string
}

func NewAuthService(queries *store.Queries, txdb TxBeginner, mailer Mailer, appURL string) Auth {
	return Auth{
		inTx: func(ctx context.Context, fn func(AuthStore) error) error {
			tx, err := txdb.Begin(ctx)
			if err != nil {
				return err
			}
			defer tx.Rollback(ctx)

			if err := fn(queries.WithTx(tx)); err != nil {
				return err
			}
			return tx.Commit(ctx)
		},
		mailer: mailer,
		appURL: appURL,
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
	var user store.User

	// Create user and user's account in transaction
	err = a.inTx(ctx, func(db AuthStore) error {
		// Create the user
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
			return err
		}

		// Create the user account
		_, err = db.CreateAccount(ctx, store.CreateAccountParams{
			UserID:   createdUser.ID,
			Provider: "email",
			PasswordHash: pgtype.Text{
				String: passwordHash,
				Valid:  true,
			},
		})
		if err != nil {
			return err
		}

		user = createdUser
		return nil
	})
	if err != nil {
		return User{}, err
	}

	// Send email verification request
	err = a.sendRegisterEmail(ctx, user)
	if err != nil {
		return User{}, err
	}

	return User{
			Name:  user.Name.String,
			Email: user.PrimaryEmail.String,
		},
		nil
}

// Generates the URL for verifying an email
func (a Auth) generateEmailVerificationURL(user store.User) (token string, err error) {
	rawToken, err := generateToken()
	if err != nil {
		return "", err
	}
	hash := hashToken(rawToken)
	return fmt.Sprintf(
		"%s/verify?token=%s",
		a.appURL,
		hash,
	), nil
}

// sendRegisterEmail sends an email for a user to verify their email
func (a Auth) sendRegisterEmail(ctx context.Context, user store.User) error {
	type RegisterEmailData struct {
		Name            string
		VerificationURL string
	}

	tmpl, err := template.ParseFiles("views/emails/register.html")
	if err != nil {
		return err
	}
	var body bytes.Buffer
	url, err := a.generateEmailVerificationURL(user)
	if err != nil {
		return err
	}
	err = tmpl.Execute(&body, RegisterEmailData{
		Name:            user.Name.String,
		VerificationURL: url,
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
