package authhandlers

import (
	"context"
	"encoding/json"
	"github.com/SiberianMonster/memoryprint/internal/config"
	"github.com/SiberianMonster/memoryprint/internal/models"
	"github.com/SiberianMonster/memoryprint/internal/tokenizer"
	"github.com/SiberianMonster/memoryprint/internal/userstorage"
	"github.com/SiberianMonster/memoryprint/internal/projectstorage"
	"github.com/SiberianMonster/memoryprint/internal/handlersfunc"
	"github.com/SiberianMonster/memoryprint/internal/authservice"
	"github.com/SiberianMonster/memoryprint/internal/emailutils"
	"log"
	"io/ioutil"
	"net/http"
	"strconv"
)

// RefreshToken handles refresh token request
func RefreshToken(rw http.ResponseWriter, r *http.Request) {

	rw.Header().Set("Content-Type", "application/json")
	resp := make(map[string]string)

	var user models.User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, resp, err)
		return
	}

	accessToken, err := authservice.GenerateAccessToken(&user)
	if err != nil {
		log.Printf("Error happened when generating jwt access token. Err: %s", err)
		handlersfunc.HandleJWTError(rw, resp)
		return
	}

	rw.WriteHeader(http.StatusOK)
	resp["status"] = "successfully generated new access token"
	resp["accesstoken"] = accessToken
	jsonResp, err := json.Marshal(resp)
	if err != nil {
			log.Printf("Error happened in JSON marshal. Err: %s", err)
			return
	}
	rw.Write(jsonResp)

}

// Greet request greet request
func Greet(rw http.ResponseWriter, r *http.Request) {

	rw.Header().Set("Content-Type", "application/json")
	resp := make(map[string]string)

	userNumBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		handlersfunc.HandleWrongBytesInput(rw, resp)
	}
	defer r.Body.Close()
	aByteToInt, _ := strconv.Atoi(string(userNumBytes))
	userID := uint(aByteToInt)

	rw.WriteHeader(http.StatusOK)
	resp["status"] = "successfully greeted user"
	jsonResp, err := json.Marshal(resp)
	if err != nil {
			log.Printf("Error happened in JSON marshal. Err: %s", err)
			return
	}
	rw.Write(jsonResp)
	
}

// GeneratePassResetCode generate a new secret code to reset password.
func GeneratePassResetCode(rw http.ResponseWriter, r *http.Request) {

	rw.Header().Set("Content-Type", "application/json")
	resp := make(map[string]string)
	var user models.User

	userNumBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		handlersfunc.HandleWrongBytesInput(rw, resp)
	}
	defer r.Body.Close()
	aByteToInt, _ := strconv.Atoi(string(userNumBytes))
	userID := uint(aByteToInt)

	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	// не забываем освободить ресурс
	defer cancel()

	user, err = userstorage.GetUserData(ctx, config.DB, userID) 
	if err != nil {
		handlersfunc.HandleWrongCredentialsError(rw, resp)
		return
	}


	// Send verification mail
	from := "elena.valchuk@gmail.com"
	to := []string{user.Email}
	subject := "Password Reset for MemoryPrint"
	mailType := emailutils.PassReset 
	mailData := &emailutils.MailData{
		Username: user.Username,
		Code: 	emailutils.GenerateRandomString(8),
	}

	mailReq := emailutils.NewMail(from, to, subject, mailType, mailData)
	err = emailutils.SendMail(mailReq)
	if err != nil {
		handlersfunc.HandleMailSendError(rw, resp)
		return
	}

	// store the password reset code to db
	verificationData := &models.VerificationData{
		Email: user.Email,
		Code:  mailData.Code,
		Type:  emailutils.PassReset,
		ExpiresAt: time.Now().Add(time.Minute * time.Duration(config.PassResetCodeExpiration)),
	}

	err = userstorage.StoreVerificationData(ctx, config.DB, verificationData)
	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
		return
	}

	rw.WriteHeader(http.StatusOK)
	resp["status"] = "successfully mailed password reset code"
	jsonResp, err := json.Marshal(resp)
	if err != nil {
			log.Printf("Error happened in JSON marshal. Err: %s", err)
			return
	}
	rw.Write(jsonResp)
}




