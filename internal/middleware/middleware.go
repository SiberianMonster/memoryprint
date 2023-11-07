package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/SiberianMonster/memoryprint/internal/authservice"
	"github.com/SiberianMonster/memoryprint/internal/config"
	"github.com/SiberianMonster/memoryprint/internal/userstorage"
	"github.com/SiberianMonster/memoryprint/internal/models"
	"github.com/SiberianMonster/memoryprint/internal/handlersfunc"
	"log"
	"strings"
	"net/http"
)

func extractToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	authHeaderContent := strings.Split(authHeader, " ")
	if len(authHeaderContent) != 2 {
		return "", errors.New("Token not provided or malformed")
	}
	return authHeaderContent[1], nil
}

// MiddlewareValidateAccessToken validates whether the request contains a bearer token
// it also decodes and authenticates the given token
func MiddlewareValidateAccessToken(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "application/json")
		resp := make(map[string]string)

		log.Println("validating access token")

		token, err := extractToken(r)
		if err != nil {
			log.Printf("Error happened when extracting jwt access token. Err: %s", err)
			handlersfunc.HandleJWTError(w, resp)
			return
			}

		userEmail, err := authservice.ValidateAccessToken(token)
		if err != nil {
			log.Printf("Error happened when validating jwt access token. Err: %s", err)
			handlersfunc.HandleJWTError(w, resp)
			return
		}
		userID, err := userstorage.GetUserID(context.Background(), config.DB, userEmail)
		if err != nil {
			log.Printf("Error happened when getting user ID by email. Err: %s", err)
			handlersfunc.HandleDatabaseServerError(w, resp)
			return
		}
		
		ctx := context.WithValue(r.Context(), config.UserIDKey, userID)
		r = r.WithContext(ctx)

		h.ServeHTTP(w, r)
	})
}

// MiddlewareValidateRefreshToken validates whether the request contains a bearer token
// it also decodes and authenticates the given token
func MiddlewareValidateRefreshToken(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "application/json")
		resp := make(map[string]string)

		log.Println("validating refresh token")
		token, err := extractToken(r)
		if err != nil {
			log.Printf("Error happened when extracting jwt access token. Err: %s", err)
			handlersfunc.HandleJWTError(w, resp)
			return
		}

		userEmail, customKey, err := authservice.ValidateRefreshToken(token)
		if err != nil {
			log.Printf("Error happened when validating jwt refresh token. Err: %s", err)
			handlersfunc.HandleJWTError(w, resp)
			return
		}

		userID, err := userstorage.GetUserID(context.Background(), config.DB, userEmail)
		if err != nil {
			log.Printf("Error happened when getting user ID by email. Err: %s", err)
			handlersfunc.HandleDatabaseServerError(w, resp)
			return
		}

		user, err := userstorage.GetUserData(context.Background(), config.DB, userID)
		if err != nil {
			handlersfunc.HandleWrongCredentialsError(w, resp)
			return
		}

		actualCustomKey := authservice.GenerateCustomKey(user.Email, user.TokenHash)
		if customKey != actualCustomKey {
			log.Printf("wrong token: authetincation failed")
			handlersfunc.HandleJWTError(w, resp)
			return
		}

		ctx := context.WithValue(r.Context(), config.UserIDKey, userID)
		r = r.WithContext(ctx)

		h.ServeHTTP(w, r)
	})
}



func AdminHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		userID := handlersfunc.UserIDContextReader(r)
		userCategory, err := userstorage.CheckUserCategory(r.Context(), config.DB, userID)
		resp := make(map[string]string)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			log.Print("Failed to check user category")
			resp["status"] = "user unauthorized"
			jsonResp, err := json.Marshal(resp)
			if err != nil {
				log.Printf("Error happened in JSON marshal. Err: %s", err)
				return
			}
			w.Write(jsonResp)
			return
		}

        // If user is admin, allows access.
        if userCategory == models.AdminCategory {
            h.ServeHTTP(w, r)
        } else {
            // Otherwise, 403.
            w.WriteHeader(http.StatusUnauthorized)
			log.Print("Not Admin")
			resp["status"] = "user unauthorized"
			jsonResp, err := json.Marshal(resp)
			if err != nil {
				log.Printf("Error happened in JSON marshal. Err: %s", err)
				return
			}
			w.Write(jsonResp)
			return
        }

        return
    })
}

func PAHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		userID := handlersfunc.UserIDContextReader(r)
		userCategory, err := userstorage.CheckUserCategory(r.Context(), config.DB, userID)
		resp := make(map[string]string)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			log.Print("Failed to check user category")
			resp["status"] = "user unauthorized"
			jsonResp, err := json.Marshal(resp)
			if err != nil {
				log.Printf("Error happened in JSON marshal. Err: %s", err)
				return
			}
			w.Write(jsonResp)
			return
		}

        // If user is print agent, allows access.
        if userCategory == models.PrintAgentUserCategory {
            h.ServeHTTP(w, r)
        } else {
            // Otherwise, 403.
            w.WriteHeader(http.StatusUnauthorized)
			log.Print("Not Print Agent")
			resp["status"] = "user unauthorized"
			jsonResp, err := json.Marshal(resp)
			if err != nil {
				log.Printf("Error happened in JSON marshal. Err: %s", err)
				return
			}
			w.Write(jsonResp)
			return
        }

        return
    })
}