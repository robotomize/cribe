module github.com/robotomize/cribe

go 1.17

require (
	github.com/aws/aws-sdk-go v1.40.54
	github.com/client9/misspell v0.3.4
	github.com/go-redis/redis/v8 v8.11.3
	github.com/go-telegram-bot-api/telegram-bot-api v4.6.4+incompatible
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/kkdai/youtube/v2 v2.7.4
	github.com/nicksnyder/go-i18n/v2 v2.1.2
	github.com/sethvargo/go-envconfig v0.3.5
	github.com/streadway/amqp v1.0.0
	go.uber.org/zap v1.17.0
	golang.org/x/text v0.3.6
	golang.org/x/tools v0.1.6
	gopkg.in/yaml.v2 v2.4.0
)

require (
	github.com/bitly/go-simplejson v0.5.0 // indirect
	github.com/cespare/xxhash/v2 v2.1.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/technoweenie/multipartstreamer v1.0.1 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	golang.org/x/mod v0.4.2 // indirect
	golang.org/x/net v0.0.0-20210805182204-aaa1db679c0d // indirect
	golang.org/x/sys v0.0.0-20210809222454-d867a43fc93e // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
)

replace github.com/kkdai/youtube/v2 v2.7.4 => github.com/robotomize/youtube/v2 v2.7.5-0.20211004084108-fc2d2467347a
