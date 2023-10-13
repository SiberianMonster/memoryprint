package tokenizer

import (
	"github.com/SiberianMonster/memoryprint/internal/config"
	"github.com/dgrijalva/jwt-go"
	"log"
	"strings"
	"time"
)

func GenerateToken(userID uint) (string, error) {

	tokenContent := jwt.MapClaims{
		"user_id": userID,
		"expiry":  time.Now().Add(config.TokenExpiration).Unix(),
	}
	jwtToken := jwt.NewWithClaims(jwt.GetSigningMethod("HS256"), tokenContent)
	token, err := jwtToken.SignedString(config.SecretKey)
	if err != nil {
		return token, err
	}

	return token, nil
}

func ValidateToken(jwtToken string) (uint, bool) {

	var userID uint
	cleanJWT := strings.Replace(jwtToken, "Bearer ", "", -1)
	tokenData := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(cleanJWT, tokenData, func(token *jwt.Token) (interface{}, error) {
		return config.SecretKey, nil
	})
	if err != nil {
		log.Printf("Error happened when parsing token. Err: %s", err)
		return userID, false
	}
	if token.Valid {
		userID = uint(tokenData["user_id"].(float64))
		return userID, true
	} else {
		log.Print("invalid token")
		return userID, false
	}
}
