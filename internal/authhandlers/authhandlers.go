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
			log.Printf("Error happened in JSON marshal in greet. Err: %s", err)
			return
	}
	rw.Write(jsonResp)
	
}

// GenerateTempPass generate a new default pass to access account.
func GenerateTempPass(rw http.ResponseWriter, r *http.Request) {

	rw.Header().Set("Content-Type", "application/json")
	resp := make(map[string]int)

	var user *models.RestoreUser
	var dbUser models.UserInfo

	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, err)
	}

	defer r.Body.Close()
	// Create a new validator instance
    validate := validator.New()
	log.Println("Generating temp pass")
	log.Println(user)

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
	log.Println("Checked registration")
	log.Println(user)
	dbUser.ID, err = userstorage.GetUserID(ctx, config.DB, user.Email) 
	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}
	log.Println("Matched email")
	log.Println(dbUser)
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
	subject := "=?koi8-r?B?98/T09TBzs/XzMXOycUgxM/T1NXQwSDLIMHLy8HVztTVIE1lbW9yeVByaW50==?="
	mailType := emailutils.MailPassTemp
	mailData := &emailutils.MailData{
		Username: dbUser.Name,
		Code: 	tempPass,
	}

	ms := &emailutils.SGMailService{config.YandexApiKey}
	print("created temp pass email")
	mailReq := emailutils.NewMail(from, to, subject, mailType, mailData)
	err = emailutils.SendMail(mailReq, ms)
	if err != nil {
		log.Printf("unable to send temp pass mail", "error", err)
		handlersfunc.HandleMailSendError(rw)
		return
	}
	
	err = userstorage.UpdateUser(ctx, config.DB, tempPass, dbUser.ID)
	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}

	rw.WriteHeader(http.StatusOK)
	resp["response"] = 1
	jsonResp, err := json.Marshal(resp)
	if err != nil {
			log.Printf("Error happened in JSON marshal for temp pass. Err: %s", err)
			return
	}
	rw.Write(jsonResp)
}




