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
	TokenExpiration   = time.Hour * 2160
	MailVerifCodeExpiration   = 30
	PassResetCodeExpiration   = 15
	Key               = "encoding124"
	SendGridApiKey    = "encoding124"
	SubscriptionKey               = "10111992060419912905199187345690"
	UpdateInterval    = time.Minute * 2
	SleepTime         = time.Second *60
	WorkersCount                    = 60
	UserIDKey         contextKey    = "userid"
	UserCategoryKey         contextKey    = "usercategory"
	VerificationDataKey contextKey = "verificationdata"
	MailVerifTemplateID   = "d-5ecbea6e38764af3b703daf03f139b48"
	TempPassTemplateID   = "d-3fc222d11809441abaa8ed459bb44319"
	DesignerOrderTemplateID   = "d-5ecbea6e38764af3b703daf03f139b48"
	ViewerInvitationNewTemplateID   = "d-5ecbea6e38764af3b703daf03f139b48"
	ViewerInvitationExistTemplateID   = "d-5ecbea6e38764af3b703daf03f139b48"
	WelcomeMailTemplateID   = "d-5ecbea6e38764af3b703daf03f139b48"
	AccessTokenPrivateKeyPath = "./auth-private.pem"
	AccessTokenPublicKeyPath = "./auth-public.pem"
	RefreshTokenPrivateKeyPath = "./refresh-private.pem"
	RefreshTokenPublicKeyPath = "./refresh-public.pem"
	
)

var DB *pgxpool.Pool
var DBctx context.Context
var SecretKey = []byte("encoding")
var AdminEmail string
var YandexApiKey string
var TimewebToken string
var BalaToken string
var ImageHost string
var BankDomain string
var BankUsername string
var BankPassword string
var DeliveryDomain string
var DeliveryClientID string
var DeliverySecret string
var EncryptionString string

func GetEnv(key string, fallback *string) *string {
	if value, ok := os.LookupEnv(key); ok {
		return &value
	}
	return fallback
}