module github.com/handziurdmytro/messenger/api-gateway

go 1.27.0

require (
	github.com/caarlos0/env/v11 v11.4.1
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/handziurdmytro/messenger/pkg/logger v0.0.0-00010101000000-000000000000
	github.com/handziurdmytro/messenger/pkg/runner v0.0.0-00010101000000-000000000000
	github.com/joho/godotenv v1.5.1
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/kr/pretty v0.3.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
)

replace (
	github.com/handziurdmytro/messenger/pkg/logger => ../../pkg/logger
	github.com/handziurdmytro/messenger/pkg/runner => ../../pkg/runner
)
