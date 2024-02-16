package userhandlers

import (
	"context"
	"encoding/json"
	
	"github.com/SiberianMonster/memoryprint/internal/config"
	"github.com/SiberianMonster/memoryprint/internal/emailutils"
	"github.com/SiberianMonster/memoryprint/internal/models"
	"github.com/SiberianMonster/memoryprint/internal/userstorage"
	"github.com/SiberianMonster/memoryprint/internal/projectstorage"
	"github.com/SiberianMonster/memoryprint/internal/handlersfunc"
	"github.com/SiberianMonster/memoryprint/internal/authservice"
	"log"
	"net/http"
)

type TokenRespBody struct {
	Token string `json:"token"`
}

type UserRespBody struct {
	User AdminBool `json:"user"`
}

type AdminBool struct {
	IsAdmin bool `json:"is_admin"`
}

func Register(rw http.ResponseWriter, r *http.Request) {

	if r.Method == "OPTIONS" {
		rw.WriteHeader(http.StatusOK)
		return
	}

	var user models.User
	var tBody TokenRespBody
	rw.Header().Set("Content-Type", "application/json")

	resp := make(map[string]TokenRespBody)
	log.Printf("Register user")
	
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, err)
		return
	}

	log.Println(user)

	if user.Name == "" || user.Password == "" || user.Email == "" {
		handlersfunc.HandleWrongCredentialsError(rw)
		return
	}

	defer r.Body.Close()

	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	// не забываем освободить ресурс
	defer cancel()

	if userstorage.CheckUser(ctx, config.DB, user) {
		handlersfunc.HandleUsernameAlreadyTaken(rw)
		return
	}

	var userID uint

	// Send welcome email
	from := "support@memoryprint.ru"
	to := []string{user.Email}
	subject := "Welcome to MemoryPrint"
	mailType := emailutils.MailWelcome
	mailData := &emailutils.MailData{
		Username: user.Name,
	}

	ms := &emailutils.SGMailService{config.YandexApiKey, config.MailVerifCodeExpiration, config.PassResetCodeExpiration, config.WelcomeMailTemplateID, config.MailVerifTemplateID, config.TempPassTemplateID, config.DesignerOrderTemplateID, config.ViewerInvitationNewTemplateID, config.ViewerInvitationExistTemplateID}
	mailReq := emailutils.NewMail(from, to, subject, mailType, mailData)
	err = emailutils.SendMail(mailReq, ms)
	if err != nil {
		log.Printf("unable to send mail", "error", err)
		handlersfunc.HandleMailSendError(rw)
		return
	}

	// Customer
	user.Category = models.CustomerCategory
	userID, err = userstorage.CreateUser(ctx, config.DB, user)
	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}

	err = projectstorage.UpdateNewUserProjects(ctx, config.DB, user.Email, userID)
	if err != nil {
		log.Printf("Error happened when updating photobooks for the new user. Err: %s", err)
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}

	accessToken, err := authservice.GenerateAccessToken(&user)
	if err != nil {
		log.Printf("Error happened when generating jwt token received value. Err: %s", err)
		handlersfunc.HandleJWTError(rw)
		return
	}

	rw.WriteHeader(http.StatusOK)
	tBody.Token = accessToken
	resp["response"] = tBody
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)

}

func Login(rw http.ResponseWriter, r *http.Request) {

	if origin := r.Header.Get("Origin"); origin != "" {
        rw.Header().Set("Access-Control-Allow-Origin", origin)
        rw.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
        rw.Header().Set("Access-Control-Allow-Headers",
            "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
    }


	if r.Method == "OPTIONS" {
		return
	}

	var user *models.User
	var tBody TokenRespBody

	resp := make(map[string]TokenRespBody)
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, err)
	}
	log.Printf("Login user")
	log.Println(user)

	if user.Email == "" && user.Name == "" {
		handlersfunc.HandleWrongCredentialsError(rw)
		return
	}

	defer r.Body.Close()

	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	// не забываем освободить ресурс
	defer cancel()

	if !userstorage.CheckUser(ctx, config.DB, *user) {
		handlersfunc.HandleUnregisteredUserError(rw)
		return
	}

	dbUser, err := userstorage.CheckCredentials(ctx, config.DB, *user)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}
	_, err = authservice.Authenticate(user, &dbUser); 
	if err != nil {
		handlersfunc.HandleWrongCredentialsError(rw)
		return
	}

	log.Println(dbUser.Name)

	accessToken, err := authservice.GenerateAccessToken(&dbUser)
	if err != nil {
		log.Printf("Error happened when generating jwt token received value. Err: %s", err)
		handlersfunc.HandleJWTError(rw)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	
	tBody.Token = accessToken
	resp["response"] = tBody

	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)

}

func CheckUserCategory(rw http.ResponseWriter, r *http.Request) {

	if r.Method == "OPTIONS" {
		rw.WriteHeader(http.StatusOK)
		return
	}

	resp := make(map[string]UserRespBody)
	var uBody UserRespBody
	var isAdmin AdminBool
	userID := handlersfunc.UserIDContextReader(r)

	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	userCategory, err := userstorage.CheckUserCategory(ctx, config.DB, userID)
		
	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}

    if userCategory == models.AdminCategory {
        isAdmin.IsAdmin = true
	} else {
		isAdmin.IsAdmin = false
	}

	uBody.User = isAdmin
	rw.WriteHeader(http.StatusOK)
	resp["response"] = uBody
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		return
	}
	rw.Write(jsonResp)

}