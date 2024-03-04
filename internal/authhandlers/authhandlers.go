package authhandlers

import (
	"context"
	"encoding/json"
	"github.com/SiberianMonster/memoryprint/internal/config"
	"github.com/SiberianMonster/memoryprint/internal/models"
	"github.com/SiberianMonster/memoryprint/internal/userstorage"
	"github.com/SiberianMonster/memoryprint/internal/handlersfunc"
	"github.com/SiberianMonster/memoryprint/internal/emailutils"
	"github.com/go-playground/validator/v10"
	"fmt"
	"log"
	"net/http"
)


// Greet request greet request
func Greet(rw http.ResponseWriter, r *http.Request) {

	rw.Header().Set("Content-Type", "application/json")
	resp := make(map[string]string)

	rw.WriteHeader(http.StatusOK)
	resp["status"] = "successfully greeted user"
	jsonResp, err := json.Marshal(resp)
	if err != nil {
			log.Printf("Error happened in JSON marshal. Err: %s", err)
			return
	}
	rw.Write(jsonResp)
	
}

// GenerateTempPass generate a new default pass to access account.
func GenerateTempPass(rw http.ResponseWriter, r *http.Request) {

	rw.Header().Set("Content-Type", "application/json")
	resp := make(map[string]int)

	var user *models.RestoreUser
	var dbUser models.User

	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, err)
	}

	defer r.Body.Close()
	// Create a new validator instance
    validate := validator.New()

    // Validate the User struct
    err = validate.Struct(user)
    if err != nil {
        // Validation failed, handle the error
		handlersfunc.HandleValidationError(rw, err)
        return
    }

	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	// не забываем освободить ресурс
	defer cancel()

	if !userstorage.CheckUser(ctx, config.DB, user.Email) {
		handlersfunc.HandleUnregisteredUserError(rw)
		return
	}
	dbUser.ID, err = userstorage.GetUserID(ctx, config.DB, user.Email) 
	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}
	dbUser, err = userstorage.GetUserData(ctx, config.DB, dbUser.ID)
	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}
	tempPass := emailutils.GenerateRandomString(8)
	log.Println(tempPass)

	// Send verification mail
	from := "support@memoryprint.ru"
	to := []string{user.Email}
	subject := "Temporary Login Details for MemoryPrint"
	mailType := emailutils.MailPassTemp
	mailData := &emailutils.MailData{
		Username: dbUser.Name,
		Code: 	tempPass,
	}

	ms := &emailutils.SGMailService{config.YandexApiKey, config.MailVerifCodeExpiration, config.PassResetCodeExpiration, config.WelcomeMailTemplateID, config.MailVerifTemplateID, config.TempPassTemplateID, config.DesignerOrderTemplateID, config.ViewerInvitationNewTemplateID, config.ViewerInvitationExistTemplateID}
	print("created ms")
	mailReq := emailutils.NewMail(from, to, subject, mailType, mailData)
	print("created mail")
	err = emailutils.SendMail(mailReq, ms)
	if err != nil {
		log.Printf("unable to send mail", "error", err)
		handlersfunc.HandleMailSendError(rw)
		return
	}
	
	dbUser.Password, err = userstorage.Hash(fmt.Sprintf("%s:password", tempPass), config.Key)
	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}
	
	_, err = userstorage.UpdatePassword(ctx, config.DB, dbUser)
	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}

	rw.WriteHeader(http.StatusOK)
	resp["response"] = 1
	jsonResp, err := json.Marshal(resp)
	if err != nil {
			log.Printf("Error happened in JSON marshal. Err: %s", err)
			return
	}
	rw.Write(jsonResp)
}




