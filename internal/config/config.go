package config

import (
	"context"
	"os"
	"time"
	"github.com/jackc/pgx/v5/pgxpool"
)

type contextKey string

const (
	ContextDBTimeout  = time.Second *5
	ContextSrvTimeout = time.Second *10
	TokenExpiration   = time.Minute *30
	MailVerifCodeExpiration   = 30
	PassResetCodeExpiration   = 15
	Key               = "encoding124"
	SendGridApiKey    = "encoding124"
	UpdateInterval    = time.Second * 5
	SleepTime         = time.Second *60
	WorkersCount                    = 60
	UserIDKey         contextKey    = "userid"
	UserCategoryKey         contextKey    = "usercategory"
	VerificationDataKey contextKey = "verificationdata"
	MailVerifTemplateID   = ""
	PassResetTemplateID   = ""
	AccessTokenPrivateKeyPath = "./access-private.pem"
	AccessTokenPublicKeyPath = "./access-public.pem"
	RefreshTokenPrivateKeyPath = "./refresh-private.pem"
	RefreshTokenPublicKeyPath = "./refresh-public.pem"
	
)

var DB *pgxpool.Pool
var DBctx context.Context
var SecretKey = []byte("encoding")

func GetEnv(key string, fallback *string) *string {
	if value, ok := os.LookupEnv(key); ok {
		return &value
	}
	return fallback
}