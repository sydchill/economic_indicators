module backend

go 1.25.11

// go.mod additions
require (
	github.com/go-chi/chi/v5 v5.5.0
	github.com/go-chi/httprate v0.9.0
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/golang-migrate/migrate/v4 v4.20.1
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.9.2
	github.com/uptrace/bun v1.3.0
	github.com/uptrace/bun/dialect/pgdialect v1.3.0
	github.com/uptrace/bun/driver/pgdriver v1.3.0
	golang.org/x/crypto v0.53.0 // argon2
	golang.org/x/oauth2 v0.36.0
	google.golang.org/api v0.287.0 // idtoken verification
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/joho/godotenv v1.5.1
	github.com/lib/pq v1.10.9 // indirect
	github.com/puzpuzpuz/xsync/v3 v3.5.1 // indirect
	github.com/tmthrgd/go-hex v0.0.0-20190904060850-447a3041c3bc // indirect
	github.com/vmihailenco/msgpack/v5 v5.4.1 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	go.opentelemetry.io/otel v1.44.0 // indirect
	go.opentelemetry.io/otel/trace v1.44.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
	mellium.im/sasl v0.3.2 // indirect
)
