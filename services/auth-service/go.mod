module github.com/handziurdmytro/messenger/auth-service

go 1.27.0

require (
	github.com/caarlos0/env/v11 v11.4.1
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/google/uuid v1.6.0
	github.com/handziurdmytro/messenger/pkg/logger v0.0.0-00010101000000-000000000000
	github.com/handziurdmytro/messenger/pkg/runner v0.0.0-00010101000000-000000000000
	github.com/jackc/pgx/v5 v5.10.0
	github.com/joho/godotenv v1.5.1
	github.com/segmentio/kafka-go v0.4.51
	golang.org/x/crypto v0.53.0
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/klauspost/compress v1.15.9 // indirect
	github.com/pierrec/lz4/v4 v4.1.15 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/text v0.38.0 // indirect
)

replace (
	github.com/handziurdmytro/messenger/pkg/logger => ../../pkg/logger
	github.com/handziurdmytro/messenger/pkg/runner => ../../pkg/runner
)
