module github.com/handziurdmytro/messenger/message-service

go 1.26.2

require (
	github.com/caarlos0/env/v11 v11.4.1
	github.com/gocql/gocql v1.7.0
	github.com/google/uuid v1.6.0
	github.com/handziurdmytro/messenger/pkg/logger v0.0.0-00010101000000-000000000000
	github.com/handziurdmytro/messenger/pkg/runner v0.0.0-00010101000000-000000000000
	github.com/joho/godotenv v1.5.1
	github.com/segmentio/kafka-go v0.4.51
	golang.org/x/sync v0.22.0
)

require (
	github.com/golang/snappy v0.0.3 // indirect
	github.com/hailocab/go-hostpool v0.0.0-20160125115350-e80d13ce29ed // indirect
	github.com/klauspost/compress v1.15.9 // indirect
	github.com/kr/pretty v0.3.0 // indirect
	github.com/pierrec/lz4/v4 v4.1.15 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	golang.org/x/net v0.55.0 // indirect
	golang.org/x/text v0.38.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
)

replace (
	github.com/handziurdmytro/messenger/pkg/logger => ../../pkg/logger
	github.com/handziurdmytro/messenger/pkg/runner => ../../pkg/runner
)
