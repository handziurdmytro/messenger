module github.com/handziurdmytro/messenger/user-service

go 1.27.0

require (
	github.com/caarlos0/env/v11 v11.4.1
	github.com/handziurdmytro/messenger/pkg/errkit v0.0.0-00010101000000-000000000000
	github.com/handziurdmytro/messenger/pkg/logger v0.0.0-00010101000000-000000000000
	github.com/handziurdmytro/messenger/pkg/pgkit v0.0.0-00010101000000-000000000000
	github.com/handziurdmytro/messenger/pkg/runner v0.0.0-00010101000000-000000000000
	github.com/jackc/pgx/v5 v5.11.0
	github.com/joho/godotenv v1.5.1
	github.com/segmentio/kafka-go v0.4.51
	github.com/stretchr/testify v1.12.1
	github.com/swaggo/swag v1.16.6
	golang.org/x/sync v0.22.0
)

require (
	github.com/KyleBanks/depth v1.2.1 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.20.0 // indirect
	github.com/go-openapi/spec v0.20.6 // indirect
	github.com/go-openapi/swag v0.19.15 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/klauspost/compress v1.18.6 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/mailru/easyjson v0.7.6 // indirect
	github.com/pierrec/lz4/v4 v4.1.16 // indirect
	github.com/rogpeppe/go-internal v1.16.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/mod v0.37.0 // indirect
	golang.org/x/text v0.40.0 // indirect
	golang.org/x/tools v0.47.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

replace (
	github.com/handziurdmytro/messenger/pkg/errkit => ../../pkg/errkit
	github.com/handziurdmytro/messenger/pkg/logger => ../../pkg/logger
	github.com/handziurdmytro/messenger/pkg/pgkit => ../../pkg/pgkit
	github.com/handziurdmytro/messenger/pkg/runner => ../../pkg/runner
)
