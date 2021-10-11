module github.com/robotomize/cribe

go 1.17

require (
	github.com/aws/aws-sdk-go v1.40.59
	github.com/client9/misspell v0.3.4
	github.com/go-redis/redis/v8 v8.11.4
	github.com/go-telegram-bot-api/telegram-bot-api v1.0.1-0.20201020035208-b6df6c273aa8
	github.com/golang-migrate/migrate/v4 v4.15.0
	github.com/golang/mock v1.6.0
	github.com/jackc/pgx/v4 v4.13.0
	github.com/kkdai/youtube/v2 v2.7.4
	github.com/lib/pq v1.10.2
	github.com/nicksnyder/go-i18n/v2 v2.1.2
	github.com/sethvargo/go-envconfig v0.3.5
	github.com/streadway/amqp v1.0.0
	go.uber.org/zap v1.19.1
	golang.org/x/text v0.3.7
	golang.org/x/tools v0.1.7
	gopkg.in/yaml.v2 v2.4.0
)

require (
	github.com/bitly/go-simplejson v0.5.0 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/hashicorp/errwrap v1.0.0 // indirect
	github.com/hashicorp/go-multierror v1.1.0 // indirect
	github.com/jackc/chunkreader/v2 v2.0.1 // indirect
	github.com/jackc/pgconn v1.10.0 // indirect
	github.com/jackc/pgio v1.0.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgproto3/v2 v2.1.1 // indirect
	github.com/jackc/pgservicefile v0.0.0-20200714003250-2b9c44734f2b // indirect
	github.com/jackc/pgtype v1.8.1 // indirect
	github.com/jackc/puddle v1.1.3 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/technoweenie/multipartstreamer v1.0.1 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.7.0 // indirect
	golang.org/x/crypto v0.0.0-20210711020723-a769d52b0f97 // indirect
	golang.org/x/mod v0.5.1 // indirect
	golang.org/x/net v0.0.0-20211008194852-3b03d305991f // indirect
	golang.org/x/sys v0.0.0-20211007075335-d3039528d8ac // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
)

replace (
	github.com/go-telegram-bot-api/telegram-bot-api => github.com/robotomize/telegram-bot-api v1.0.1-0.20211011160432-7f279bff1862
	github.com/kkdai/youtube/v2 v2.7.4 => github.com/robotomize/youtube/v2 v2.7.5-0.20211004084108-fc2d2467347a
)
