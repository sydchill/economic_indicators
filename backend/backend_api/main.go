package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	"github.com/uptrace/bun"

	"github.com/sydchill/economic_indicators/backend/api/auth"
	"github.com/sydchill/economic_indicators/shared/config"
	"github.com/sydchill/economic_indicators/shared/db"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.LoadServer()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	bunDB, err := db.New(ctx, cfg.DatabaseURL)
	cancel()
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer bunDB.Close()

	// Migrations deferred to cmd/migrate in the next component.

	authSvc := auth.NewService(bunDB, cfg.GoogleClientID)
	authH := auth.NewHandler(authSvc)
	// marketH := market.NewHandler(bunDB)
	r := chi.NewRouter()
	r.Get("/health", healthHandler(bunDB))
	r.Mount("/api/auth", authH.Routes())

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("listening on %s", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	// Graceful shutdown
	shutdown, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-shutdown.Done()

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelShutdown()
	_ = srv.Shutdown(shutdownCtx)
	log.Println("server stopped")
}

func healthHandler(db *bun.DB) http.HandlerFunc {
	type tableRef struct {
		Schema string `bun:"table_schema"`
		Name   string `bun:"table_name"`
	}

	requiredTables := []string{
		"auth.users",
		"auth.credentials",
		"auth.oauth_accounts",
		"auth.sessions",
		"auth.password_reset_tokens",
		"users.profiles",
	}

	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := db.PingContext(ctx); err != nil {
			auth.WriteJSON(w, 503, map[string]any{"status": "error", "database": err.Error()})
			return
		}

		var refs []tableRef
		err := db.NewSelect().
			TableExpr("information_schema.tables").
			ColumnExpr("table_schema, table_name").
			Where("table_schema IN ('auth', 'users')").
			Where("table_type = 'BASE TABLE'").
			Scan(ctx, &refs)
		if err != nil {
			auth.WriteJSON(w, 503, map[string]any{"status": "error", "schema": err.Error()})
			return
		}

		present := make(map[string]bool, len(refs))
		for _, t := range refs {
			present[t.Schema+"."+t.Name] = true
		}
		missing := make([]string, 0)
		for _, name := range requiredTables {
			if !present[name] {
				missing = append(missing, name)
			}
		}

		if len(missing) > 0 {
			auth.WriteJSON(w, 503, map[string]any{
				"status":         "error",
				"missing_tables": missing,
			})
			return
		}

		auth.WriteJSON(w, 200, map[string]any{
			"status":   "ok",
			"database": "ok",
			"tables":   len(requiredTables),
		})
	}
}
