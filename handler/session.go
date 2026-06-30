package handler

import (
	"encoding/gob"
	"time"

	"github.com/solaris-soft/goforge/db"
	"github.com/solaris-soft/goforge/store"

	"github.com/alexedwards/scs/pgxstore"
	"github.com/alexedwards/scs/v2"
)

var SessionManager *scs.SessionManager

func InitSession() {
	// Register structs we want to set on the session
	gob.Register(store.User{})

	// Initialize a new session manager and configure the session lifetime.
	SessionManager = scs.New()
	SessionManager.Store = pgxstore.New(db.Pool)
	SessionManager.Lifetime = 24 * time.Hour
}
