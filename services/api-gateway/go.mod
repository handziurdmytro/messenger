module github.com/handziurdmytro/messenger/api-gateway

go 1.26.2

require (
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/joho/godotenv v1.5.1
)

require (
	github.com/handziurdmytro/messenger/pkg/env v0.0.0-00010101000000-000000000000
	gopkg.in/yaml.v3 v3.0.1
)

replace github.com/handziurdmytro/messenger/pkg/env => ../../pkg/env
