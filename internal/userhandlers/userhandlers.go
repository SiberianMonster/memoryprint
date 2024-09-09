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
	"github.com/SiberianMonster/memoryprint/internal/transactions"
	"github.com/go-playground/validator/v10"
	"log"
	"net/http"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"

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
	var subLink string 
	subLink, err = userstorage.GetAESEncrypted(user.Email)
	log.Println(user.Email)
	if err != nil {
		log.Println(err)
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}

	// Send welcome email
	from := "support@memoryprint.ru"
	to := []string{user.Email}
	subject := "Welcome to MemoryPrint"
	mailType := emailutils.MailWelcome
	mailData := &emailutils.MailData{
		Username: user.Name,
		SubscriptionLink: subLink,
	}

	ms := &emailutils.SGMailService{config.YandexApiKey}
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

func GetUserInfo(rw http.ResponseWriter, r *http.Request) {

	
	rw.Header().Set("Content-Type", "application/json")

	resp := make(map[string]models.UserInfo)
	log.Printf("Get user data")
	
	userID := handlersfunc.UserIDContextReader(r)

	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	// не забываем освободить ресурс
	defer cancel()

	dbUser, err := userstorage.GetUserData(ctx, config.DB, userID)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}

	rw.WriteHeader(http.StatusOK)
	resp["response"] = dbUser
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)

}

func UpdateUsername(rw http.ResponseWriter, r *http.Request) {

	var updatedUser models.UpdatedUsername
	rw.Header().Set("Content-Type", "application/json")

	resp := make(map[string]string)
	log.Printf("Update user data")
	
	err := json.NewDecoder(r.Body).Decode(&updatedUser)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, err)
		return
	}
	userID := handlersfunc.UserIDContextReader(r)

	log.Println(updatedUser)

	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	// не забываем освободить ресурс
	defer cancel()
	err = userstorage.UpdateUsername(ctx, config.DB, updatedUser.Name, userID)
	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}

	rw.WriteHeader(http.StatusOK)
	resp["response"] = "1"
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)

}

func UpdateUserInfo(rw http.ResponseWriter, r *http.Request) {

	var updatedUser models.UpdatedUser
	var dbUser models.User
	rw.Header().Set("Content-Type", "application/json")

	resp := make(map[string]string)
	log.Printf("Update user data")
	
	err := json.NewDecoder(r.Body).Decode(&updatedUser)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, err)
		return
	}
	// Create a new validator instance
    validate := validator.New()

    // Validate the User struct
    err = validate.Struct(updatedUser)
    if err != nil {
        // Validation failed, handle the error
		handlersfunc.HandleValidationError(rw, err)
        return
    }
	if updatedUser.Password == updatedUser.NewPassword {
		handlersfunc.HandleSamePassError(rw)
		return
	}

	log.Println(updatedUser)

	userID := handlersfunc.UserIDContextReader(r)
	defer r.Body.Close()

	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	// не забываем освободить ресурс
	defer cancel()

	dbUser, err = userstorage.CheckCredentialsByID(ctx, config.DB, userID)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}
	var u models.User
	u.Password = updatedUser.Password
	_, err = authservice.Authenticate(u, &dbUser); 
	if err != nil {
		handlersfunc.HandleWrongCredentialsError(rw)
		return
	}

	err = userstorage.UpdateUser(ctx, config.DB, updatedUser.NewPassword, userID)
	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}

	rw.WriteHeader(http.StatusOK)
	resp["response"] = "1"
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

func CancelSubscription(rw http.ResponseWriter, r *http.Request) {

		resp := make(map[string]string)
		code := mux.Vars(r)["code"]
	
		ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
		defer cancel()
		err := userstorage.CancelSubscription(ctx, config.DB, code)
			
		if err != nil {
			handlersfunc.HandleDatabaseServerError(rw)
			return
		}
	
		rw.WriteHeader(http.StatusOK)
		resp["response"] = "1"
		jsonResp, err := json.Marshal(resp)
		if err != nil {
			return
		}
		rw.Write(jsonResp)

}

func RenewSubscription(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]string)
	code := mux.Vars(r)["code"]

	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	err := userstorage.RenewSubscription(ctx, config.DB, code)
		
	if err != nil {
		handlersfunc.HandleFailedRenewSubscription(rw)
		return
	}

	rw.WriteHeader(http.StatusOK)
	resp["response"] = "1"
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

// CreateCertificate creates a new gift certificate entry.
func CreateCertificate(rw http.ResponseWriter, r *http.Request) {

	rw.Header().Set("Content-Type", "application/json")
	resp := make(map[string]models.TransactionLink)
	var transaction models.TransactionLink
	var cID uint

	var certificate *models.GiftCertificate

	err := json.NewDecoder(r.Body).Decode(&certificate)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, err)
	}

	defer r.Body.Close()
	// Create a new validator instance
    validate := validator.New()
	log.Println(certificate)

    // Validate the User struct
    err = validate.Struct(certificate)
    if err != nil {
        // Validation failed, handle the error
		handlersfunc.HandleValidationError(rw, err)
        return
    }

	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	// не забываем освободить ресурс
	defer cancel()

	cID, err = userstorage.CreateCertificate(ctx, config.DB, certificate)
	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}
	transaction.PaymentLink, err = transactions.CreateTransaction(cID, certificate.Deposit, "CERTIFICATE")
	// Impossible to create payment link
	if err != nil {
		handlersfunc.HandleFailedPaymentURL(rw)
		return
	}


	rw.WriteHeader(http.StatusOK)
	resp["response"] = transaction
	jsonResp, err := json.Marshal(resp)
	if err != nil {
			log.Printf("Error happened in JSON marshal. Err: %s", err)
			return
	}
	rw.Write(jsonResp)
}




// CreatePromocode creates a new promocode.
func CreatePromocode(rw http.ResponseWriter, r *http.Request) {

	rw.Header().Set("Content-Type", "application/json")
	resp := make(map[string]string)

	var promooffer *models.NewPromooffer

	err := json.NewDecoder(r.Body).Decode(&promooffer)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, err)
	}

	defer r.Body.Close()
	// Create a new validator instance
    validate := validator.New()
	log.Println(promooffer)

    // Validate the User struct
    err = validate.Struct(promooffer)
    if err != nil {
        // Validation failed, handle the error
		handlersfunc.HandleValidationError(rw, err)
        return
    }

	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	// не забываем освободить ресурс
	defer cancel()

	
	err = userstorage.CreatePromooffer(ctx, config.DB, promooffer)
	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}

	rw.WriteHeader(http.StatusOK)
	resp["response"] = "1"
	jsonResp, err := json.Marshal(resp)
	if err != nil {
			log.Printf("Error happened in JSON marshal. Err: %s", err)
			return
	}
	rw.Write(jsonResp)
}

// CheckPromocode checks the validity of the code.
func CheckPromocode(rw http.ResponseWriter, r *http.Request) {

	rw.Header().Set("Content-Type", "application/json")
	resp := make(map[string]models.ResponsePromocode)

	var promooffer *models.CheckPromooffer
	var checkP models.CheckPromocode

	err := json.NewDecoder(r.Body).Decode(&promooffer)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, err)
	}

	defer r.Body.Close()
	// Create a new validator instance
    validate := validator.New()
	log.Println(promooffer)

    // Validate the User struct
    err = validate.Struct(promooffer)
    if err != nil {
        // Validation failed, handle the error
		handlersfunc.HandleValidationError(rw, err)
        return
    }

	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	// не забываем освободить ресурс
	defer cancel()

	userID := handlersfunc.UserIDContextReader(r)
	
	checkP, err = userstorage.CheckPromocode(ctx, config.DB, promooffer.Code, userID)
	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}
	if checkP.Status == "INVALID" {
		handlersfunc.HandleMissingPromocode(rw)
		return
	}

	if checkP.Status == "FORBIDDEN" {
		handlersfunc.HandleMissingPromocode(rw)
		return
	}

	if checkP.Status == "EXPIRED" {
		handlersfunc.HandleExpiredError(rw)
		return
	}

	if checkP.Status == "ALREADY USED" {
		handlersfunc.HandleAlreadyUsedError(rw)
		return
	}

	rw.WriteHeader(http.StatusOK)
	resp["response"] = checkP.Promocode
	jsonResp, err := json.Marshal(resp)
	if err != nil {
			log.Printf("Error happened in JSON marshal. Err: %s", err)
			return
	}
	rw.Write(jsonResp)
}

func LoadPromocodes(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]models.Promooffers)
	var rPromooffers []models.Promooffer
	var rPromooffer models.Promooffers
	defer r.Body.Close()

	
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	rPromooffers, err := userstorage.LoadPromocodes(ctx, config.DB)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}

	rPromooffer.Promocodes = rPromooffers
	rw.WriteHeader(http.StatusOK)
	resp["response"] = rPromooffer
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)

}

// UsePromocode applies promocode to the published projects.
func UsePromocode(rw http.ResponseWriter, r *http.Request) {

	rw.Header().Set("Content-Type", "application/json")
	resp := make(map[string]models.ResponsePromocodeUse)
	var responseP models.ResponsePromocodeUse
	var requestP models.RequestPromooffer
	var checkP models.CheckPromocode

	err := json.NewDecoder(r.Body).Decode(&requestP)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, err)
	}

	defer r.Body.Close()
	// Create a new validator instance
    validate := validator.New()
	log.Println(requestP)

    // Validate the User struct
    err = validate.Struct(requestP)
    if err != nil {
        // Validation failed, handle the error
		handlersfunc.HandleValidationError(rw, err)
        return
    }

	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	// не забываем освободить ресурс
	defer cancel()

	userID := handlersfunc.UserIDContextReader(r)
	
	checkP, err = userstorage.CheckPromocode(ctx, config.DB, requestP.Code, userID)
	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}
	
	responseP, err = userstorage.UsePromocode(ctx, config.DB, requestP)
	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}
	if checkP.Status == "INVALID" {
		handlersfunc.HandleMissingPromocode(rw)
		return
	}

	if checkP.Status == "FORBIDDEN" {
		handlersfunc.HandlePermissionError(rw)
		return
	}

	if checkP.Status == "EXPIRED" {
		handlersfunc.HandleExpiredError(rw)
		return
	}

	if checkP.Status == "ALREADY USED" {
		handlersfunc.HandleAlreadyUsedError(rw)
		return
	}
	if checkP.Status == "VALID" && responseP.BasePrice == responseP.DiscountedPrice {
		handlersfunc.HandleWrongPromocodeCategoryError(rw)
		return
	}
	
	rw.WriteHeader(http.StatusOK)
	resp["response"] = responseP
	jsonResp, err := json.Marshal(resp)
	if err != nil {
			log.Printf("Error happened in JSON marshal. Err: %s", err)
			return
	}
	rw.Write(jsonResp)
}

// UseCertificate applies gift certificate to the published projects.
func UseCertificate(rw http.ResponseWriter, r *http.Request) {

	rw.Header().Set("Content-Type", "application/json")
	code := mux.Vars(r)["code"]
	resp := make(map[string]models.ResponseCertificate)
	var responseC models.ResponseCertificate

	
	defer r.Body.Close()
	
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	// не забываем освободить ресурс
	defer cancel()

	userID := handlersfunc.UserIDContextReader(r)
	
	deposit, status, err := userstorage.UseCertificate(ctx, config.DB, code, userID)
	if status == "INVALID" {
		handlersfunc.HandleWrongGiftCodeError(rw)
		return
	}

	if status == "DEPLETED" {
		handlersfunc.HandleAlreadyUsedGiftcertificateError(rw)
		return
	}
	responseC.Deposit = deposit
	rw.WriteHeader(http.StatusOK)
	resp["response"] = responseC
	jsonResp, err := json.Marshal(resp)
	if err != nil {
			log.Printf("Error happened in JSON marshal. Err: %s", err)
			return
	}
	rw.Write(jsonResp)
}


func SentGiftCertificateMail(ctx context.Context, storeDB *pgxpool.Pool) {

	ticker := time.NewTicker(config.UpdateInterval)
	var err error
	var certificateList []models.GiftCertificate

	jobCh := make(chan models.GiftCertificate)
	for i := 0; i < config.WorkersCount; i++ {
		go func() {
			for job := range jobCh {
	
				err = userstorage.MailCertificate(ctx, storeDB, job)
				if err != nil {
					log.Printf("Error happened when updating pending gift certificates. Err: %s", err)
					continue
				}
			}
		}()
	}

	for range ticker.C {

		certificateList, err = userstorage.LoadUnSentCertificate(ctx, storeDB)
		if err != nil {
			log.Printf("Error happened when retrieving pending gift certificates. Err: %s", err)
			continue
		}

		for _, certificate := range certificateList {
			jobCh <- certificate

		}

	}
}
