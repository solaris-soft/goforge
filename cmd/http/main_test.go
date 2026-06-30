package main

import "testing"

func TestInitialiseAllowedOriginsTrimsCSV(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", "https://app.example.com, http://localhost:5173 ,")
	allowedOrigins = nil

	initialiseAllowedOrigins()

	want := []string{"https://app.example.com", "http://localhost:5173"}
	if len(allowedOrigins) != len(want) {
		t.Fatalf("expected %d origins, got %d: %#v", len(want), len(allowedOrigins), allowedOrigins)
	}
	for i := range want {
		if allowedOrigins[i] != want[i] {
			t.Fatalf("origin %d: expected %q, got %q", i, want[i], allowedOrigins[i])
		}
	}
}

func TestCorsOptionsExposeHtmxHeaders(t *testing.T) {
	allowedOrigins = []string{"https://app.example.com"}
	opts := corsOptions()

	if !opts.AllowCredentials {
		t.Fatal("expected credentials for session-backed htmx requests")
	}
	if !contains(opts.AllowedHeaders, "HX-Request") {
		t.Fatal("expected HX-Request in allowed headers")
	}
	if !contains(opts.ExposedHeaders, "HX-Redirect") {
		t.Fatal("expected HX-Redirect in exposed headers")
	}
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
