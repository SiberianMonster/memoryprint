package userhandlers

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/gorilla/mux"
	
	"github.com/SiberianMonster/memoryprint/internal/config"
	"github.com/SiberianMonster/memoryprint/internal/emailutils"
	"github.com/SiberianMonster/memoryprint/internal/models"
	"github.com/SiberianMonster/memoryprint/internal/userstorage"
	"github.com/SiberianMonster/memoryprint/internal/projectstorage"
	"github.com/SiberianMonster/memoryprint/internal/handlersfunc"
	"github.com/SiberianMonster/memoryprint/internal/authservice"
	"github.com/go-playground/validator/v10"
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

	var user models.SignUpUser
	var signedUser models.User
	var tBody TokenRespBody
	rw.Header().Set("Content-Type", "application/json")

	resp := make(map[string]TokenRespBody)
	log.Printf("Register user")
	
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, err)
		return
	}
	// Create a new validator instance
    validate := validator.New()

    // Validate the User struct
    err = validate.Struct(user)
    if err != nil {
        // Validation failed, handle the error
		handlersfunc.HandleValidationError(rw, err)
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

	if userstorage.CheckUser(ctx, config.DB, user.Email) {
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

	signedUser.Email = user.Email
	signedUser.Password = user.Password
	accessToken, err := authservice.GenerateAccessToken(&signedUser)
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

	var user models.LoginUser
	var loggedUser models.User
	var tBody TokenRespBody

	resp := make(map[string]TokenRespBody)
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, err)
	}
	log.Printf("Login user")
	log.Println(user)
	// Create a new validator instance
    validate := validator.New()

    // Validate the User struct
    err = validate.Struct(user)
    if err != nil {
        // Validation failed, handle the error
		handlersfunc.HandleValidationError(rw, err)
        return
    }
	log.Println("validated user")
	if user.Email == "" && user.Password == "" {
		handlersfunc.HandleWrongCredentialsError(rw)
		return
	}

	defer r.Body.Close()

	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	// не забываем освободить ресурс
	defer cancel()

	if !userstorage.CheckUser(ctx, config.DB, user.Email) {
		handlersfunc.HandleUnregisteredUserError(rw)
		return
	}
	log.Println("checked user")
	loggedUser.Email = user.Email
	loggedUser.Password = user.Password
	log.Println(loggedUser)
	dbUser, err := userstorage.CheckCredentials(ctx, config.DB, loggedUser)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}
	_, err = authservice.Authenticate(loggedUser, &dbUser); 
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

func MakeUserAdmin(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]string)

	rw.Header().Set("Content-Type", "application/json")
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	userID := uint(aByteToInt)

	log.Printf("Making user admin again")
	
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	// не забываем освободить ресурс
	defer cancel()

	userstorage.MakeUserAdmin(ctx, config.DB, userID) 
	resp["response"] = "1"
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)

}