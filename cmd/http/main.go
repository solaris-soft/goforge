package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/gorilla/csrf"
	"github.com/solaris-soft/goforge/db"
	"github.com/solaris-soft/goforge/handler"
	"github.com/solaris-soft/goforge/service"
	"github.com/solaris-soft/goforge/store"

	"github.com/joho/godotenv"
)

var (
	// Allowed origins for CORS
	allowedOrigins []string = nil
	// Environment this server is running in dev/prod
	env string = "dev"
)

func main() {
	// DB and fail quick initialisations
	initialize()
	defer teardown()
	host := defaultEnv("APP_HOST", ":3000")

	// Ensure teardown on interrupt
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signals
		teardown()
		os.Exit(0)
	}()

	router := chi.NewRouter()

	// Middleware stack
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.RequestID)
	router.Use(middleware.Timeout(60 * time.Second))
	router.Use(cors.Handler(corsOptions()))
	router.Use(csrfMiddleware())

	// Static files
	router.Handle("/public/", http.FileServer(http.Dir("public")))

	// Health check
	router.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(200)
		w.Write([]byte("OK"))
	})

	queries := store.New(db.Pool)

	// Services
	mailerService := service.NewEmailClient(mustEnv("RESEND_API_KEY"), mustEnv("FROM_ADDRESS"))
	authService := service.NewAuthService(queries,
		db.Pool,
		mailerService,
		strings.Split(host, ":")[0], // Remove port
	)

	// Auth Routes
	authController := handler.NewAuthHandler(authService)
	router.Get("/signup", handler.Make(authController.SignUp))
	router.Post("/signup", handler.Make(authController.PostSignUp))

	// Static files
	filesDir := http.Dir("public")
	fileServer(router, "/public", filesDir)

	// Start listening
	slog.Info("Listening", "host", host)
	http.ListenAndServe(host, handler.SessionManager.LoadAndSave(router))
}

func fileServer(r chi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		panic("FileServer does not permit any URL parameters.")
	}

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", http.StatusMovedPermanently).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, http.FileServer(root))
		fs.ServeHTTP(w, r)
	})
}

func initialize() {
	// Load env vars
	if err := godotenv.Load(); err != nil {
		log.Fatalf("couldn't load env vars: %v", err)
	}
	initialiseAllowedOrigins()

	// Setup DB
	if err := db.Init(); err != nil {
		log.Fatalf("couldn't init db: %v", err)
	}

	handler.InitSession()
}

func teardown() {
	slog.Info("Teardown started...")
	db.Teardown()
	slog.Info("Teardown finished.")
}

func mustEnv(key string) string {
	env := os.Getenv(key)
	if env == "" {
		log.Fatalf("%v is required, not set", key)
	}
	return env
}

func defaultEnv(key, fallback string) string {
	env := os.Getenv(key)
	if env == "" {
		return fallback
	}
	return env
}

func corsOptions() cors.Options {
	return cors.Options{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{
			"Accept",
			"Authorization",
			"Content-Type",
			"X-CSRF-Token",
			"HX-Request",
			"HX-Trigger",
			"HX-Trigger-Name",
			"HX-Target",
			"HX-Current-URL",
		},
		ExposedHeaders: []string{
			"HX-Location",
			"HX-Push-Url",
			"HX-Redirect",
			"HX-Refresh",
			"HX-Replace-Url",
			"HX-Reselect",
			"HX-Reswap",
			"HX-Retarget",
			"HX-Trigger",
			"HX-Trigger-After-Settle",
			"HX-Trigger-After-Swap",
		},
		AllowCredentials: true,
		MaxAge:           300,
	}
}

func initialiseAllowedOrigins() {
	raw := mustEnv("ALLOWED_ORIGINS")
	origins := strings.Split(raw, ",")
	allowedOrigins = allowedOrigins[:0]
	for _, origin := range origins {
		if origin = strings.TrimSpace(origin); origin != "" {
			allowedOrigins = append(allowedOrigins, origin)
		}
	}
}

// Configure CSRF tokens
func csrfMiddleware() func(http.Handler) http.Handler {
	secure := strings.Contains(env, "prod")
	cm := csrf.Protect(
		[]byte(mustEnv("CSRF_SECRET_KEY")),
		csrf.Secure(secure),
		csrf.Path("/"),
	)
	return cm
}
