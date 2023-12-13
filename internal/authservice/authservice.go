// Storage package contains functions for storing photos and projects in a pgx database.
//
// Available at https://github.com/SiberianMonster/memoryprint/tree/development/internal/userstorage
package authservice

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/ioutil"
	"fmt"
	"log"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/SiberianMonster/memoryprint/internal/config"
	"github.com/SiberianMonster/memoryprint/internal/models"
)

var err error

func Hash(value, key string) (string, error) {
	mac := hmac.New(sha256.New, []byte(key))
	_, err := mac.Write([]byte(value))
	return fmt.Sprintf("%x", mac.Sum(nil)), err
}

// RefreshTokenCustomClaims specifies the claims for refresh token
type RefreshTokenCustomClaims struct {
	UserEmail    string
	CustomKey string
	KeyType   string
	jwt.StandardClaims
}

// AccessTokenCustomClaims specifies the claims for access token
type AccessTokenCustomClaims struct {
	UserEmail  string
	KeyType string
	jwt.StandardClaims
}


func Authenticate(u *models.User, dbUser *models.User) (bool, error) {

	pwdHash, err := Hash(fmt.Sprintf("%s:password", u.Password), config.Key)
	if err != nil {
		log.Printf("Error happened when hashing received value. Err: %s", err)
		return false, err
	}

	if pwdHash != dbUser.Password {
		log.Printf("Wrong password")
		err = errors.New("wrong password")
		return false, err
	}
	return true, nil
}

// GenerateCustomKey creates a new key for our jwt payload
// the key is a hashed combination of the userID and user tokenhash
func GenerateCustomKey(userEmail string, tokenHash string) string {

	log.Println("generating custom key")
	log.Println(userEmail)
	log.Println(tokenHash)
	// data := userID + tokenHash
	h := hmac.New(sha256.New, []byte(tokenHash))
	h.Write([]byte(userEmail))
	sha := hex.EncodeToString(h.Sum(nil))
	return sha
}


func GenerateRefreshToken(dbUser *models.User) (string, error) {

	cusKey := GenerateCustomKey(dbUser.Email, dbUser.TokenHash)
	tokenType := "refresh"

	claims := RefreshTokenCustomClaims{
		dbUser.Email,
		cusKey,
		tokenType,
		jwt.StandardClaims{
			Issuer: "memoryprint.auth.service",
		},
	}

	signBytes, err := ioutil.ReadFile(config.RefreshTokenPrivateKeyPath)
	if err != nil {
		log.Printf("unable to read private key", "error", err)
		return "", errors.New("could not generate refresh token. please try again later")
	}

	signKey, err := jwt.ParseRSAPrivateKeyFromPEM(signBytes)
	if err != nil {
		log.Printf("unable to parse private key", "error", err)
		return "", errors.New("could not generate refresh token. please try again later")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	return token.SignedString(signKey)
}

// GenerateAccessToken generates a new access token for the given user
func GenerateAccessToken(dbUser *models.User) (string, error) {

	tokenType := "access"

	claims := AccessTokenCustomClaims{
		dbUser.Email,
		tokenType,
		jwt.StandardClaims{
			ExpiresAt: time.Now().Add(config.TokenExpiration).Unix(),
			Issuer:    "memoryprint.auth.service",
		},
	}

	signBytes, err := ioutil.ReadFile(config.AccessTokenPrivateKeyPath)
	if err != nil {
		log.Printf("unable to read private key", "error", err)
		return "", errors.New("could not generate access token. please try again later")
	}

	signKey, err := jwt.ParseRSAPrivateKeyFromPEM(signBytes)
	if err != nil {
		log.Printf("unable to parse private key", "error", err)
		return "", errors.New("could not generate access token. please try again later")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	return token.SignedString(signKey)
}

// ValidateAccessToken parses and validates the given access token
// returns the userId present in the token payload
func ValidateAccessToken(tokenString string) (string, error) {

	token, err := jwt.ParseWithClaims(tokenString, &AccessTokenCustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			log.Printf("Unexpected signing method in auth token")
			return nil, errors.New("Unexpected signing method in auth token")
		}
		verifyBytes, err := ioutil.ReadFile(config.AccessTokenPublicKeyPath)
		if err != nil {
			log.Printf("unable to read public key", "error", err)
			return nil, err
		}

		verifyKey, err := jwt.ParseRSAPublicKeyFromPEM(verifyBytes)
		if err != nil {
			log.Printf("unable to parse public key", "error", err)
			return nil, err
		}

		return verifyKey, nil
	})

	if err != nil {
		log.Printf("unable to parse claims", "error", err)
		return "", err
	}

	claims, ok := token.Claims.(*AccessTokenCustomClaims)
	if !ok || !token.Valid || claims.UserEmail == "" || claims.KeyType != "access" {
		return "", errors.New("invalid token: authentication failed")
	}
	return claims.UserEmail, nil
}

// ValidateRefreshToken parses and validates the given refresh token
// returns the userId and customkey present in the token payload
func ValidateRefreshToken(tokenString string) (string, string, error) {

	token, err := jwt.ParseWithClaims(tokenString, &RefreshTokenCustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			log.Printf("Unexpected signing method in auth token")
			return nil, errors.New("Unexpected signing method in auth token")
		}
		verifyBytes, err := ioutil.ReadFile(config.RefreshTokenPublicKeyPath)
		if err != nil {
			log.Printf("unable to read public key", "error", err)
			return nil, err
		}

		verifyKey, err := jwt.ParseRSAPublicKeyFromPEM(verifyBytes)
		if err != nil {
			log.Printf("unable to parse public key", "error", err)
			return nil, err
		}

		return verifyKey, nil
	})

	if err != nil {
		log.Printf("unable to parse claims", "error", err)
		return "", "", err
	}

	claims, ok := token.Claims.(*RefreshTokenCustomClaims)
	log.Printf("ok", ok)
	if !ok || !token.Valid || claims.UserEmail == "" || claims.KeyType != "refresh" {
		log.Printf("could not extract claims from token")
		return "", "", errors.New("invalid token: authentication failed")
	}
	return claims.UserEmail, claims.CustomKey, nil
}