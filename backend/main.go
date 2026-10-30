package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	"github.com/uptrace/bun"

	"backend/config"

	"backend/db"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	bunDB, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer bunDB.Close()

	if err := db.Migrate(bunDB); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	log.Println("database connected and migrated")

	r := chi.NewRouter()
	r.Get("/health", healthHandler(bunDB))

	log.Printf("listening on %s", cfg.HTTPAddr)
	if err := http.ListenAndServe(cfg.HTTPAddr, r); err != nil {
		log.Fatal(err)
	}
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
			writeJSON(w, 503, map[string]any{"status": "error", "database": err.Error()})
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
			writeJSON(w, 503, map[string]any{"status": "error", "schema": err.Error()})
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
			writeJSON(w, 503, map[string]any{
				"status":         "error",
				"missing_tables": missing,
			})
			return
		}

		writeJSON(w, 200, map[string]any{
			"status":   "ok",
			"database": "ok",
			"tables":   len(requiredTables),
		})
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
