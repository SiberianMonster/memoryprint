package userhandlers

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
	"log"
	"io/ioutil"
	"net/http"
	"strconv"
)

func Register(rw http.ResponseWriter, r *http.Request) {

	var user models.User
	rw.Header().Set("Content-Type", "application/json")

	resp := make(map[string]string)
	
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, resp, err)
		return
	}

	log.Println(user)

	if user.Username == "" || user.Password == "" || user.Email == "" {
		handlersfunc.HandleWrongCredentialsError(rw, resp)
		return
	}

	defer r.Body.Close()

	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	// не забываем освободить ресурс
	defer cancel()

	if !userstorage.CheckUser(ctx, config.DB, user) {
		handlersfunc.HandleUsernameAlreadyTaken(rw, resp)
		return
	}

	var userID uint

	// Customer
	user.Category = models.CustomerCategory
	userID, err = userstorage.CreateUser(ctx, config.DB, user)
	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
		return
	}

	err = projectstorage.UpdateNewUserProjects(ctx, config.DB, user.Email, userID)
	if err != nil {
		log.Printf("Error happened when updating photobooks for the new user. Err: %s", err)
		handlersfunc.HandleDatabaseServerError(rw, resp)
		return
	}

	// Send verification mail
	from := "elena.valchuk@gmail.com"
	to := []string{user.Email}
	subject := "Email Verification for Bookite"
	mailType := emailutils.MailConfirmation
	mailData := &emailutils.MailData{
		Username: user.Username,
		Code: 	emailutils.GenerateRandomString(8),
	}

	mailReq := emailutils.NewMail(from, to, subject, mailType, mailData)
	err = emailutils.SendMail(mailReq)
	if err != nil {
		log.Printf("unable to send mail", "error", err)
		handlersfunc.HandleMailSendError(rw, resp)
		return
	}

	verificationData := &models.VerificationData{
		Email: user.Email,
		Code : mailData.Code,
		Type : emailutils.MailConfirmation,
		ExpiresAt: time.Now().Add(time.Hour * time.Duration(config.MailVerifCodeExpiration)),
	}

	err = userstorage.StoreVerificationData(ctx, config.DB, verificationData)
	if err != nil {
		log.Printf("unable to store mail verification data", "error", err)
		handlersfunc.HandleDatabaseServerError(rw, resp)
		return
	}

	rw.WriteHeader(http.StatusOK)
	resp["status"] = "new user added successfully"
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)

}

func Login(rw http.ResponseWriter, r *http.Request) {

	var user models.User
	var valid bool

	resp := make(map[string]string)
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, resp, err)
	}
	log.Println(user)

	if user.Email == "" || user.Password == "" {
		handlersfunc.HandleWrongCredentialsError(rw, resp)
		return
	}

	defer r.Body.Close()

	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	// не забываем освободить ресурс
	defer cancel()

	dbUser, err := userstorage.CheckCredentials(ctx, config.DB, user)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
		return
	}
	valid, err = authservice.Authenticate(user, dbUser); 
	if err != nil {
		handlersfunc.HandlePermissionError(rw, resp)
		return
	}

	log.Println(dbUser.Username)

	accessToken, err := authservice.GenerateAccessToken(dbUser)
	if err != nil {
		log.Printf("Error happened when generating jwt token received value. Err: %s", err)
		handlersfunc.HandleJWTError(rw, resp)
		return
	}
	refreshToken, err := authservice.GenerateRefreshToken(dbUser)
	if err != nil {
		log.Printf("Error happened when generating jwt token received value. Err: %s", err)
		handlersfunc.HandleJWTError(rw, resp)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	resp["status"] = "user logged in successfully"
	resp["username"] = dbUser.Username
	resp["accesstoken"] = accessToken
	resp["refreshtoken"] = refreshToken
	resp["userID"] = dbUser.UserID

	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)

}

func ViewUsers(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]string)
	defer r.Body.Close()
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	users, err := userstorage.RetrieveUsers(ctx, config.DB)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
		return
	}

	if len(users) == 0 {
		handlersfunc.HandleNoContent(rw, resp)
		return
	}

	rw.WriteHeader(http.StatusOK)
	json.NewEncoder(rw).Encode(users)
}

func DeleteUser(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]string)
	userNumBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		handlersfunc.HandleWrongBytesInput(rw, resp)
	}
	defer r.Body.Close()
	aByteToInt, _ := strconv.Atoi(string(userNumBytes))
	userID := uint(aByteToInt)

	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()


	_, err =  userstorage.DeleteUser(ctx, config.DB, userID)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
	}

	rw.WriteHeader(http.StatusOK)
	resp["status"] = "user deleted successfully"
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)

}

func UpdateUserCategory(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]string)
	var UserParams models.User
	err := json.NewDecoder(r.Body).Decode(&UserParams)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, resp, err)
	}


	defer r.Body.Close()

	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	_, err = userstorage.UpdateUserCategory(ctx, config.DB, UserParams)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
	}

	
	rw.WriteHeader(http.StatusOK)
	resp["status"] = "user category updated successfully"
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)

}

func UpdateUserStatus(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]string)
	var UserParams models.User

	err := json.NewDecoder(r.Body).Decode(&UserParams)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, resp, err)
	}

	defer r.Body.Close()

	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	_, err = userstorage.UpdateUserStatus(ctx, config.DB, UserParams)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
	}

	
	rw.WriteHeader(http.StatusOK)
	resp["status"] = "user status updated successfully"
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)

}

func verify(actualVerificationData *models.VerificationData, verificationData *models.VerificationData) (bool, error) {

	// check for expiration
	if actualVerificationData.ExpiresAt.Before(time.Now()) {
		log.Println("verification data provided is expired")
		ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
		defer cancel()
		err := ah.repo.DeleteVerificationData(ctx, actualVerificationData.Email, actualVerificationData.Type)
		log.Println("unable to delete verification data from db", "error", err)
		return false, errors.New("Confirmation code has expired. Please try generating a new code")
	}

	if actualVerificationData.Code != verificationData.Code {
		log.Println("verification of mail failed. Invalid verification code provided")
		return false, errors.New("Verification code provided is Invalid. Please look in your mail for the code")
	}

	return true, nil
}

// VerifyMail verifies the provided confirmation code and set the User state to verified
func VerifyMail(rw http.ResponseWriter, r *http.Request) {

	rw.Header().Set("Content-Type", "application/json")

	var verificationData models.VerificationData

	err := json.NewDecoder(r.Body).Decode(&verificationData)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, resp, err)
	}

	defer r.Body.Close()
	verificationData.Type = emailutils.MailConfirmation

	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()

	actualVerificationData, err := userstorage.GetVerificationData(ctx, config.DB, verificationData.Email, verificationData.Type)
	if err != nil {
		log.Printf("unable to fetch verification data", "error", err)
		handlersfunc.HandleDatabaseServerError(rw, resp)
		return
	}

	valid, err := verify(actualVerificationData, &verificationData)
	if !valid {
		handlersfunc.HandleVerificationError(rw, resp)
		return
	}

	// correct code, update user status to verified.
	err = userstorage.UpdateUserVerificationStatus(ctx, config.DB, verificationData.Email)
	if err != nil {
		log.Printf("unable to set user verification status to true")
		handlersfunc.HandleDatabaseServerError(rw, resp)
		return
	}

	// delete the VerificationData from db
	err = userstorage.DeleteVerificationData(ctx, config.DB, verificationData.Type)
	if err != nil {
		log.Printf("unable to delete the verification data", "error", err)
	}

	log.Printf("user mail verification succeeded")

	rw.WriteHeader(http.StatusOK)
	resp["status"] = "user mail verified successfully"
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

// VerifyPasswordReset verifies the code provided for password reset
func VerifyPasswordReset(rw http.ResponseWriter, r *http.Request) {

	rw.Header().Set("Content-Type", "application/json")

	var verificationData models.VerificationData

	err := json.NewDecoder(r.Body).Decode(&verificationData)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, resp, err)
	}

	defer r.Body.Close()
	verificationData.Type = emailutils.PassReset
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()

	actualVerificationData, err := userstorage.GetVerificationData(ctx, config.DB, verificationData.Email, verificationData.Type)
	if err != nil {
		log.Printf("unable to fetch password verification data", "error", err)
		handlersfunc.HandleDatabaseServerError(rw, resp)
		return
	}

	valid, err := verify(actualVerificationData, &verificationData)
	if !valid {
		handlersfunc.HandleVerificationError(rw, resp)
		return
	}

	respData := struct{
		Code string
	}{
		Code: verificationData.Code,
	}

	log.Printf("password reset code verification succeeded")

	rw.WriteHeader(http.StatusOK)
	resp["status"] = "user password reset successfully"
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

// UpdateUsername handles username update request
func (ah *AuthHandler) UpdateUsername(rw http.ResponseWriter, r *http.Request) {

	rw.Header().Set("Content-Type", "application/json")

	var UserParams models.User

	err := json.NewDecoder(r.Body).Decode(&UserParams)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, resp, err)
	}

	defer r.Body.Close()

	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()

	err = userstorage.UpdateUsername(ctx, config.DB, UserParams.Username, UserParams.UserID)
	if err != nil {
		log.Printf("unable to update username", "error", err)
		handlersfunc.HandleDatabaseServerError(rw, resp)
		return
	}

	log.Printf("username updated successfully")

	rw.WriteHeader(http.StatusOK)
	resp["status"] = "username updated successfully"
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

// PasswordReset handles the password reset request
func ResetPassword(rw http.ResponseWriter, r *http.Request) {

	rw.Header().Set("Content-Type", "application/json")

	var user models.User
	var pwdHash string
	passResetReq := &models.PasswordResetReq{}
	err := json.NewDecoder(r.Body).Decode(&passResetReq)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, resp, err)
	}

	defer r.Body.Close()

	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()


	userID := handlersfunc.UserIDContextReader(r)
	user, err = userstorage.GetUserData(ctx, config.DB, userID)
	if err != nil {
		log.Printf("unable to retrieve the user from db", "error", err)
		handlersfunc.HandleDatabaseServerError(rw, resp)
		return
	}

	verificationData, err := userstorage.GetVerificationData(ctx, config.DB, user.Email, emailutils.PassReset)
	if err != nil {
		log.Printf("unable to retrieve the verification data from db", "error", err)
		handlersfunc.HandleDatabaseServerError(rw, resp)
		return
	}

	if verificationData.Code != passResetReq.Code {
		// we should never be here.
		handlersfunc.HandleVerificationError(rw, resp)
		return
	}

	if passResetReq.Password != passResetReq.PasswordRe {
		log.Printf("password and password re-enter did not match")
		handlersfunc.HandleWrongCredentialsError(rw, resp)
		return
	}

	pwdHash, err = userstorage.Hash(fmt.Sprintf("%s:password", user.Password), config.Key)
	if err != nil {
		log.Printf("Error happened when hashing received value. Err: %s", err)
		handlersfunc.HandleDatabaseServerError(rw, resp)
		return
	}
	tokenHash = emailutils.GenerateRandomString(15)
	err = userstorage.UpdatePassword(ctx, config.DB, user, pwdHash, tokenHash)
	if err != nil {
		log.Printf("unable to update user password in db", "error", err)
		handlersfunc.HandleDatabaseServerError(rw, resp)
		return
	}


	// delete the VerificationData from db
	err = userstorage.DeleteVerificationData(ctx, config.DB, verificationData.Email, verificationData.Type)
	if err != nil {
		log.Printf("unable to delete the verification data", "error", err)
	}

	log.Printf("password reset successfully")

	rw.WriteHeader(http.StatusOK)
	resp["status"] = "password reset successfully"
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}