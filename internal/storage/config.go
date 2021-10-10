package storage

type S3Config struct {
	Region   string `env:"S3_REGION,default=eu-west-3"`
	AccessID string `env:"S3_ACCESS_ID"`
	Secret   string `env:"S3_SECRET_KEY"`
}
