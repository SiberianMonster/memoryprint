// Storage package contains functions for storing photos and projects in a pgx database.
//
// Available at https://github.com/SiberianMonster/memoryprint/tree/development/internal/userstorage
package userstorage

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"github.com/SiberianMonster/memoryprint/internal/config"
	"github.com/SiberianMonster/memoryprint/internal/emailutils"
	"github.com/SiberianMonster/memoryprint/internal/models"
	"github.com/SiberianMonster/memoryprint/internal/projectstorage"
	"log"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"
	"time"
	"strings"
	"errors"
	"math/rand"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5"
)

var err error

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
// GenerateRandomString generate a string of random characters of given length
func GenerateRandomString(n int) string {
	sb := strings.Builder{}
	sb.Grow(n)
	for i := 0; i < n; i++ {
		idx := rand.Int63() % int64(len(letterBytes))
		sb.WriteByte(letterBytes[idx])
	}
	return sb.String()
}


func Hash(value, key string) (string, error) {
	mac := hmac.New(sha256.New, []byte(key))
	_, err := mac.Write([]byte(value))
	return fmt.Sprintf("%x", mac.Sum(nil)), err
}

// GetAESDecrypted decrypts given text in AES 256 CBC
func GetAESDecrypted(encrypted string) (string, error) {
	iv := "2410196226071937"

	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)

	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher([]byte(config.SubscriptionKey))

	if err != nil {
		return "", err
	}

	if len(ciphertext)%aes.BlockSize != 0 {
		return "", fmt.Errorf("block size cant be zero")
	}

	mode := cipher.NewCBCDecrypter(block, []byte(iv))
	mode.CryptBlocks(ciphertext, ciphertext)
	ciphertext = PKCS5UnPadding(ciphertext)

	return string(ciphertext), nil
}

// PKCS5UnPadding  pads a certain blob of data with necessary data to be used in AES block cipher
func PKCS5UnPadding(src []byte) []byte {
	length := len(src)
	unpadding := int(src[length-1])

	return src[:(length - unpadding)]
}

// GetAESEncrypted encrypts given text in AES 256 CBC
func GetAESEncrypted(plaintext string) (string, error) {
	iv := "2410196226071937"

	var plainTextBlock []byte
	length := len(plaintext)

	if length%16 != 0 {
		extendBlock := 16 - (length % 16)
		plainTextBlock = make([]byte, length+extendBlock)
		copy(plainTextBlock[length:], bytes.Repeat([]byte{uint8(extendBlock)}, extendBlock))
	} else {
		plainTextBlock = make([]byte, length)
	}

	copy(plainTextBlock, plaintext)
	block, err := aes.NewCipher([]byte(config.SubscriptionKey))

	if err != nil {
		return "", err
	}

	ciphertext := make([]byte, len(plainTextBlock))
	mode := cipher.NewCBCEncrypter(block, []byte(iv))
	mode.CryptBlocks(ciphertext, plainTextBlock)

	str := base64.StdEncoding.EncodeToString(ciphertext)

	return str, nil
}

func CalculateBasePriceByID(ctx context.Context, storeDB *pgxpool.Pool, pID uint) (float64, error) {

	var totalBaseprice float64
	var basePrice float64
	var extraPriceperpage float64
	var size, variant, cover string
	var countPages int

	err := storeDB.QueryRow(ctx, "SELECT size, variant, cover, count_pages FROM projects WHERE projects_id = ($1);", pID).Scan(&size, &variant, &cover, &countPages)
	
	if err != nil {
		log.Printf("Error happened when retrieving project data from pgx table. Err: %s", err)
		return totalBaseprice, err
	}

	err = storeDB.QueryRow(ctx, "SELECT baseprice, extrapage FROM prices WHERE size = ($1) AND variant = ($2) AND cover = ($3);", size, variant, cover).Scan(&basePrice, &extraPriceperpage)
	
	if err != nil {
		log.Printf("Error happened when retrieving base price from pgx table. Err: %s", err)
		return totalBaseprice, err
	}
	
	extraPrice := extraPriceperpage*float64((countPages-23))
	totalBaseprice = basePrice + extraPrice


	return totalBaseprice, nil
	
}

func CheckUser(ctx context.Context, storeDB *pgxpool.Pool, email string) bool {

	var userBool bool
	err := storeDB.QueryRow(ctx, "SELECT EXISTS (SELECT users_id FROM users WHERE email = ($1));", email).Scan(&userBool)
	if err != nil {
		log.Printf("Error happened when checking if user is in db. Err: %s", err)
		return userBool
	}

	return userBool
}

func CheckUserHasProject(ctx context.Context, storeDB *pgxpool.Pool, userID uint, projectID uint) bool {

	var checkProject bool
	var email string
	userCat, err := CheckUserCategory(ctx, storeDB , userID)
	if userCat == "ADMIN" {
		return true
	}
	err = storeDB.QueryRow(ctx, "SELECT email FROM users WHERE users_id = ($1);", userID).Scan(&email)
	if err != nil {
			log.Printf("Error happened when retrieving user email data from db. Err: %s", err)
			return false
	}
	err = storeDB.QueryRow(ctx, "SELECT CASE WHEN EXISTS (SELECT * FROM users_edit_projects WHERE projects_id = ($1) AND email = ($2)) THEN TRUE ELSE FALSE END;", projectID, email).Scan(&checkProject)
	if err != nil {
		log.Printf("Error happened when checking if user can edit project in db. Err: %s", err)
		return false
	}
	log.Println(checkProject)

	return checkProject
}

func GetUserData(ctx context.Context, storeDB *pgxpool.Pool, userID uint) (models.UserInfo, error) {

	var dbUser models.UserInfo
	err := storeDB.QueryRow(ctx, "SELECT users_id, username, email, tokenhash FROM users WHERE users_id = ($1);", userID).Scan(&dbUser.ID, &dbUser.Name, &dbUser.Email, &dbUser.TokenHash)
	if err != nil {
		log.Printf("Error happened when checking if user is in db. Err: %s", err)
		return dbUser, err
	}
	var orderID uint
	err = storeDB.QueryRow(ctx, "SELECT orders_id FROM orders WHERE users_id = ($1) and status = ($2) ORDER BY created_at;", userID, "AWAITING PAYMENT").Scan(&orderID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				log.Printf("Error happened when retrieving unpaid order info from pgx table. Err: %s", err)
				return dbUser, err
	}

	err = storeDB.QueryRow(ctx, "SELECT COUNT(projects_id) FROM orders_has_projects WHERE orders_id = ($1);", orderID).Scan(&dbUser.CartObjects)
	if err != nil && err != pgx.ErrNoRows{
				log.Printf("Error happened when counting projects for order in pgx table. Err: %s", err)
				return dbUser, err
	}
	return dbUser, nil
}

func GetUserID(ctx context.Context, storeDB *pgxpool.Pool, userEmail string) (uint, error) {

	var userID uint
	err := storeDB.QueryRow(ctx, "SELECT users_id FROM users WHERE email = ($1);", userEmail).Scan(&userID)
	if err != nil {
		log.Printf("Error happened when checking if user is in db. Err: %s", err)
		return userID, err
	}

	return userID, nil
}

func CreateUser(ctx context.Context, storeDB *pgxpool.Pool, u models.SignUpUser) (uint, error) {

	var userID uint
	t := time.Now()
	tokenHash := emailutils.GenerateRandomString(15)
	pwdHash, err := Hash(fmt.Sprintf("%s:password", u.Password), tokenHash)
	if err != nil {
		log.Printf("Error happened when hashing received value. Err: %s", err)
		return userID, err
	}
	

	_, err = storeDB.Exec(ctx, "INSERT INTO users (username, password, email, tokenhash, category, status, isverified, subscription, last_edited_at, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);",
		u.Name,
		pwdHash,
		u.Email,
		tokenHash,
		"CUSTOMER",
		"UNVERIFIED",
		models.UnverifiedStatus,
		true,
		t,
		t,
	)
	if err != nil {
		log.Printf("Error happened when inserting a new user entry into pgx table. Err: %s", err)
		return userID, err
	}
	err = storeDB.QueryRow(ctx, "SELECT users_id FROM users WHERE email=($1);", u.Email).Scan(&userID)
	if err != nil {
		log.Printf("Error happened when retrieving usersid from the db. Err: %s", err)
		return userID, err
	}
	return userID, nil
}

func UpdateUser(ctx context.Context, storeDB *pgxpool.Pool, password string, userID uint) (error) {

	t := time.Now()
	tokenHash := emailutils.GenerateRandomString(15)
	pwdHash, err := Hash(fmt.Sprintf("%s:password", password), tokenHash)
	if err != nil {
			log.Printf("Error happened when hashing received value. Err: %s", err)
			return err
	}
	
	_, err = storeDB.Exec(ctx, "UPDATE users SET password = ($1), tokenhash = ($2), last_edited_at = ($3) WHERE users_id = ($4);",
			pwdHash,
			tokenHash,
			t,
			userID,
	)
	if err != nil {
			log.Printf("Error happened when updating user entry into pgx table. Err: %s", err)
			return err
	}
	
	
	return nil
}

func CheckCredentials(ctx context.Context, storeDB *pgxpool.Pool, u models.User) (models.User, error) {

	var dbUser models.User
	
	err := storeDB.QueryRow(ctx, "SELECT username, email, password, tokenhash, users_id FROM users WHERE email=($1);", u.Email).Scan(&dbUser.Name, &dbUser.Email, &dbUser.Password, &dbUser.TokenHash, &dbUser.ID)
	if err != nil {
		log.Printf("Error happened when retrieving credentials from the db. Err: %s", err)
		return dbUser, err
	}


	return dbUser, nil
}

func CheckCredentialsByID(ctx context.Context, storeDB *pgxpool.Pool, userID uint) (models.User, error) {

	var dbUser models.User
	
	err := storeDB.QueryRow(ctx, "SELECT username, email, password, tokenhash, users_id FROM users WHERE users_id=($1);", userID).Scan(&dbUser.Name, &dbUser.Email, &dbUser.Password, &dbUser.TokenHash, &dbUser.ID)
	if err != nil {
		log.Printf("Error happened when retrieving credentials from the db. Err: %s", err)
		return dbUser, err
	}


	return dbUser, nil
}

func CheckUserCategory(ctx context.Context, storeDB *pgxpool.Pool, userID uint) (string, error) {

	var userCategory string
	
	err = storeDB.QueryRow(ctx, "SELECT category FROM users WHERE users_id=($1);", userID).Scan(&userCategory)
	if err != nil {
		log.Printf("Error happened when retrieving user category from the db. Err: %s", err)
		return userCategory, err
	}
	return userCategory, nil
}


func UpdateUserCategory(ctx context.Context, storeDB *pgxpool.Pool, u models.User) (uint, error) {

	if !CheckUser(ctx, storeDB, u.Email) {
		return u.ID, nil
	}

	_, err = storeDB.Exec(ctx, "UPDATE users SET category = ($1) WHERE username = ($2);",
		u.Category,
		u.Name,
	)
	if err != nil {
		log.Printf("Error happened when updating user category into pgx table. Err: %s", err)
		return u.ID, err
	}
	err = storeDB.QueryRow(ctx, "SELECT users_id FROM users WHERE username=($1);", u.Name).Scan(&u.ID)
	if err != nil {
		log.Printf("Error happened when retrieving usersid from the db. Err: %s", err)
		return u.ID, err
	}
	return u.ID, nil
}

func MakeUserAdmin(ctx context.Context, storeDB *pgxpool.Pool, userID uint) {


	_, err = storeDB.Exec(ctx, "UPDATE users SET category = ($1) WHERE users_id = ($2);",
		"ADMIN",
		userID,
	)
	if err != nil {
		log.Printf("Error happened when updating user category into pgx table. Err: %s", err)
	}
}

func UpdateUserStatus(ctx context.Context, storeDB *pgxpool.Pool, u models.User) (uint, error) {

	if !CheckUser(ctx, storeDB, u.Email) {
		return u.ID, nil
	}

	_, err = storeDB.Exec(ctx, "UPDATE users SET status = ($1) WHERE username = ($2);",
		u.Status,
		u.Name,
	)
	if err != nil {
		log.Printf("Error happened when updating user status into pgx table. Err: %s", err)
		return u.ID, err
	}
	err = storeDB.QueryRow(ctx, "SELECT users_id FROM users WHERE username=($1);", u.Name).Scan(&u.ID)
	if err != nil {
		log.Printf("Error happened when retrieving usersid from the db. Err: %s", err)
		return u.ID, err
	}
	return u.ID, nil
}

// UpdateUserVerificationStatus updates user verification status to true
func UpdateUserVerificationStatus(ctx context.Context, storeDB *pgxpool.Pool, email string) error {

	t := time.Now()
	_, err = storeDB.Exec(ctx, "UPDATE users SET isverified = ($1), last_edited_at = ($2) WHERE email = ($3);",
		models.VerifiedStatus,
		t,
		email,
	)
	if err != nil {
		log.Printf("Error happened when updating user verification status into pgx table. Err: %s", err)
		return err
	}
	return nil
}

// UpdateUsername updates the username of the given user
func UpdateUsername(ctx context.Context, storeDB *pgxpool.Pool, userName string, userID uint) error {

	_, err = storeDB.Exec(ctx, "UPDATE users SET username = ($1) WHERE users_id = ($2);",
		userName,
		userID,
	)
	if err != nil {
		log.Printf("Error happened when updating username into pgx table. Err: %s", err)
		return err
	}
	return nil
}


func DeleteUser(ctx context.Context, storeDB *pgxpool.Pool, userID uint) (uint, error) {

	_, err = storeDB.Exec(ctx, "DELETE FROM users WHERE users_id=($1);", userID)

	if err != nil {
		log.Printf("Error happened when deleting user entry from pgx table. Err: %s", err)
		return userID, err
	}

	return userID, nil
}


func RetrieveUsers(ctx context.Context, storeDB *pgxpool.Pool) ([]models.User, error) {

	var users []models.User
	rows, err := storeDB.Query(ctx, "SELECT username, password, email, category, status FROM users;")
	if err != nil {
		log.Printf("Error happened when retrieving users from pgx table. Err: %s", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var user models.User
		if err = rows.Scan(&user.Name, &user.Password, &user.Email, &user.Category,
			&user.Status); err != nil {
			log.Printf("Error happened when retrieving users from pgx table. Err: %s", err)
			return nil, err
		}
		
		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error happened when retrieving users from pgx table. Err: %s", err)
		return nil, err
	}
	return users, nil

}


// StoreMailVerificationData adds a mail verification data to db
func StoreVerificationData(ctx context.Context, storeDB *pgxpool.Pool, verificationData *models.VerificationData) error {


	_, err = storeDB.Exec(ctx, "INSERT INTO verifications (email, code, expires_at, type) VALUES ($1, $2, $3, $4);",
		verificationData.Email,
		verificationData.Code,
		verificationData.ExpiresAt,
		verificationData.Type,
		
	)
	if err != nil {
		log.Printf("Error happened when inserting a new verification entry into pgx table. Err: %s", err)
		return err
	}

	return nil
}

// GetMailVerificationCode retrieves the stored verification code.
func GetVerificationData(ctx context.Context, storeDB *pgxpool.Pool, email string, verificationDataType int) (*models.VerificationData, error) {
	
	var verificationData models.VerificationData

	err = storeDB.QueryRow(ctx, "SELECT * FROM verifications WHERE email = $1 and type = $2", email, verificationDataType).Scan(&verificationData.ID, &verificationData.Email, &verificationData.Code, &verificationData.ExpiresAt, &verificationData.Type)
	if err != nil {
		log.Printf("Error happened when retrieving verifications from the db. Err: %s", err)
		return &verificationData, err
	}
	
	return &verificationData, nil
}

// DeleteMailVerificationData deletes a used verification data
func DeleteVerificationData(ctx context.Context, storeDB *pgxpool.Pool, email string, verificationDataType int) error {

	_, err = storeDB.Exec(ctx, "DELETE FROM verifications WHERE email = $1 and type = $2", email, verificationDataType)
	if err != nil {
		log.Printf("Error happened when deleting verifications from the db. Err: %s", err)
		return err
	}
	
	return nil
}

func CreateCertificate(ctx context.Context, storeDB *pgxpool.Pool, c *models.GiftCertificate) (uint, error) {

	var cID uint
	t := time.Now()
	code := GenerateRandomString(12)

	err = storeDB.QueryRow(ctx, "INSERT INTO giftcertificates (code, initialdeposit, currentdeposit, status, created_at, receipientemail, reciepientname, buyerfirstname, buyerlastname, buyeremail, buyerphone, mail_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) RETURNING giftcertificates_id;",
		code,
		c.Deposit,
		c.Deposit,
		"CREATED",
		t,
		c.Recipientemail,
		c.Recipientname,
		c.Buyerfirstname,
		c.Buyerlastname,
		c.Buyeremail,
		c.Buyerphone,
		c.MailAt,
	).Scan(&cID)
	if err != nil {
		log.Printf("Error happened when inserting a new gift certificate entry into pgx table. Err: %s", err)
		return cID, err
	}

	return cID, nil
}

func PurchaseCertificate(ctx context.Context, storeDB *pgxpool.Pool, cID uint, tID uint) (uint, error) {

	_, err = storeDB.Exec(ctx, "INSERT INTO giftcertificates_has_transactions (giftcertificates_id, transactions_id) VALUES ($1, $2);",
		cID,
		tID,
	)

	_, err = storeDB.Exec(ctx, "UPDATE giftcertificates SET status = ($1) WHERE giftcertificates_id = ($2);",
		"PAID",
		cID,
		
	)
	if err != nil {
		log.Printf("Error happened when inserting a new gift certificate entry into pgx table. Err: %s", err)
		return cID, err
	}

	return cID, nil
}

func CreatePromooffer(ctx context.Context, storeDB *pgxpool.Pool, p *models.NewPromooffer) (error) {

	_, err = storeDB.Exec(ctx, "INSERT INTO promooffers (code, discount, category, is_onetime, expires_at, users_id, is_used) VALUES ($1, $2, $3, $4, $5, $6, $7);",
		p.Code,
		p.Discount,
		p.Category,
		p.IsOnetime,
		p.ExpiresAt,
		p.UsersID,
		false,
	)
	if err != nil {
		log.Printf("Error happened when inserting a new promocode entry into pgx table. Err: %s", err)
		return err
	}

	return nil
}

func CheckPromocode(ctx context.Context, storeDB *pgxpool.Pool, code string, usersID uint) (models.CheckPromocode, error) {

	var promooffer models.CheckPromocode
	var responseP models.ResponsePromocode
	var userID uint
	var isUsed bool
	var isOnetime bool
	now:=time.Now()
		
	err = storeDB.QueryRow(ctx, "SELECT discount, category, is_onetime, is_used, expires_at, users_id FROM promooffers WHERE code=($1);", code).Scan(&responseP.Discount, &responseP.Category, &isOnetime, &isUsed, &responseP.ExpiresAt, &userID)
	tm := time.Unix(responseP.ExpiresAt, 0)

	if err != nil && err != pgx.ErrNoRows { 
		log.Printf("Error happened when retrieving promooffer data from the db. Err: %s", err)
		return promooffer, err
	}
	if err == pgx.ErrNoRows {
		promooffer.Status = "INVALID"
		return promooffer, nil
	}
	if userID != 0 {
		if userID != usersID {
			promooffer.Status = "FORBIDDEN"
			return promooffer, nil
		}

	} else if isOnetime == true && isUsed == true {
		promooffer.Status = "ALREADY USED"
		return promooffer, nil

	} else if now.After(tm) || now.Equal(tm) {
		promooffer.Status = "EXPIRED"
		return promooffer, nil
	}
	promooffer.Status = "VALID"
	promooffer.Promocode = responseP
	return promooffer, nil
}


func UsePromocode(ctx context.Context, storeDB *pgxpool.Pool, requestP models.RequestPromooffer) (models.ResponsePromocodeUse, error) {

	var responseP models.ResponsePromocodeUse
	var err error
	var categoryPC string
	var discount float64
	var totalPrice float64
	var totalBasePrice float64
	err = storeDB.QueryRow(ctx, "SELECT promooffers_id, category, discount FROM promooffers WHERE code=($1);", requestP.Code).Scan(&responseP.PromocodeID, &categoryPC, &discount)
	if err != nil && err != pgx.ErrNoRows { 
		log.Printf("Error happened when retrieving promooffer category from the db. Err: %s", err)
		return responseP, err
	}
	
	for _, projectID := range requestP.Projects {
		var projectP float64
		var categoryP string
		projectP, err = CalculateBasePriceByID(ctx, storeDB, projectID)
        err = storeDB.QueryRow(ctx, "SELECT category FROM projects WHERE projects_id=($1);", projectID).Scan(&categoryP)
		if err != nil && err != pgx.ErrNoRows { 
			log.Printf("Error happened when retrieving promooffer category from the db. Err: %s", err)
			return responseP, err
		}
		totalBasePrice = totalBasePrice + projectP
		if categoryPC != "" {
			responseP.Category = categoryPC
			if categoryPC == categoryP {
				responseP.Discount = discount
				projectP = projectP*(1-discount)
			}
		} else {
			responseP.Discount = discount
			projectP = projectP*(1-discount)
		}
		totalPrice = totalPrice + projectP
	}
	responseP.BasePrice = totalBasePrice
	responseP.DiscountedPrice = totalPrice
	log.Println(totalBasePrice)
	log.Println(totalPrice)
	
	return responseP, nil

}

func CheckCertificate(ctx context.Context, storeDB *pgxpool.Pool, code string, users_id uint) (string, float64, error) {

	var status string
	var deposit float64
	var email string
	var recipientEmail string

	status = "INVALID"

	err := storeDB.QueryRow(ctx, "SELECT email FROM users WHERE users_id=($1);", users_id).Scan(&email)
	if err != nil && err != pgx.ErrNoRows { 
		log.Printf("Error happened when retrieving user email data from the db. Err: %s", err)
		return status, deposit, err
	}
	
	err = storeDB.QueryRow(ctx, "SELECT currentdeposit, status, receipientemail FROM giftcertificates WHERE code=($1);", code).Scan(&deposit, &status, &recipientEmail)
	if err != nil && err != pgx.ErrNoRows { 
		log.Printf("Error happened when retrieving gift certificate data from the db. Err: %s", err)
		return status, deposit, err
	}

	if recipientEmail != email {
		status = "FORBIDDEN"

	}

	if deposit == 0 {
		status = "DEPLETED"

	}
	return status, deposit, err
}

func UseCertificate(ctx context.Context, storeDB *pgxpool.Pool, code string, userID uint) (float64, string, error) {

	status, deposit, err := CheckCertificate(ctx, storeDB, code, userID)
	if err != nil && err != pgx.ErrNoRows { 
		log.Printf("Error happened when checking gift certificate in the db. Err: %s", err)
		return 0, "INVALID", err
	}

	if status == "FORBIDDEN" {
		return 0, "INVALID", nil
	} else if status == "DEPLETED" {
		return 0, "DEPLETED", nil
		
	} else if status == "PAID" {

		return deposit, "ACTIVE", nil
	
	}
	return 0, "INVALID", err
	

}

// LoadPromocodes function performs the operation of retrieving prices from pgx database with a query.
func LoadPromocodes(ctx context.Context, storeDB *pgxpool.Pool) ([]models.Promooffer, error) {

	promocodes := []models.Promooffer{}
	now:=time.Now()

	rows, err := storeDB.Query(ctx, "SELECT code, discount, category, expires_at FROM promooffers WHERE users_id = ($1);", 0)
	if err != nil {
			log.Printf("Error happened when retrieving promocodes from pgx table. Err: %s", err)
			return promocodes, err
	}
	defer rows.Close()

	for rows.Next() {

			var pObj models.Promooffer
			if err = rows.Scan(&pObj.Code, &pObj.Discount, &pObj.Category, &pObj.ExpiresAt); err != nil {
				log.Printf("Error happened when scanning promooffers. Err: %s", err)
				return promocodes, err
			}
			var templateSet models.ResponseTemplates
			templateSet, err  = projectstorage.LoadPromocodeTemplates(ctx, storeDB, pObj.Category)
			if err != nil {
				log.Printf("Error happened when retrieving promocode templates from pgx table. Err: %s", err)
				return promocodes, err
			}
			pObj.Templates = templateSet.Templates
			tm := time.Unix(pObj.ExpiresAt, 0)
			if now.Before(tm) {
				promocodes = append(promocodes, pObj)
			}
		
	}

	return promocodes, nil

}


func LoadUnSentCertificate(ctx context.Context, storeDB *pgxpool.Pool) ([]models.GiftCertificate, error) {

	var certificates []models.GiftCertificate

	rows, err := storeDB.Query(ctx, "SELECT giftcertificates_id, mail_at FROM giftcertificates WHERE mail_sent = ($1);", false)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				log.Printf("Error happened when retrieving unmailed certificates from pgx table. Err: %s", err)
				return certificates, err
	}

	
	for rows.Next() {
		var certificate models.GiftCertificate
		var mailATStorage time.Time
		if err = rows.Scan(&certificate.ID, &certificate.Code, &certificate.Deposit, &certificate.Recipientemail, &certificate.Recipientname, &mailATStorage); err != nil {
			log.Printf("Error happened when scanning certificates. Err: %s", err)
			return certificates, err
		}
		certificate.MailAt = mailATStorage.Unix()

		certificates = append(certificates, certificate)
	}
	return certificates, nil

}

// MailCertificate function performs the operation of sending gift certificate mail from pgx database with a query.
func MailCertificate(ctx context.Context, storeDB *pgxpool.Pool, certificate models.GiftCertificate) (error) {

	now:=time.Now()
	if now.After(time.Unix(certificate.MailAt,0)) || now.Equal(time.Unix(certificate.MailAt,0))  {
		// Send gif certificate email
		from := "support@memoryprint.ru"
		to := []string{certificate.Recipientemail}
		subject := "Вам подарили сертификат на фотокнигу!"
		mailType := emailutils.MailGiftCertificate
		mailData := &emailutils.MailData{
			Username: certificate.Recipientname,
			Code: certificate.Code,
		}

		ms := &emailutils.SGMailService{config.YandexApiKey}
		mailReq := emailutils.NewMail(from, to, subject, mailType, mailData)
		err = emailutils.SendMail(mailReq, ms)
		if err != nil {
			log.Printf("unable to send mail", "error", err)
			return err
		}
		_, err = storeDB.Exec(ctx, "UPDATE giftcertificates SET mail_sent = ($1) WHERE giftcertificates_id = ($2);",
			true,
			certificate.ID,
		)
		if err != nil {
				log.Printf("Error happened when updating gift certificate status into pgx table. Err: %s", err)
				return err
		}
	}
	
	return nil

}

func CancelSubscription(ctx context.Context, storeDB *pgxpool.Pool, code string) (error) {


	email, err := GetAESDecrypted(code)
	if err != nil {
		log.Printf("Error happened when decrypting user subscription data. Err: %s", err)
		return err
	}
	_, err = storeDB.Exec(ctx, "UPDATE users SET subscription = ($1) WHERE email = ($2);",
			false,
			email,
	)
	if err != nil {
			log.Printf("Error happened when updating user subscription into pgx table. Err: %s", err)
			return err
	}
	
	return nil
}

func RenewSubscription(ctx context.Context, storeDB *pgxpool.Pool, code string) (error) {

	email, err := GetAESDecrypted(code) 
	if err != nil {
		log.Printf("Error happened when decrypting user subscription data. Err: %s", err)
		return err
	}
	_, err = storeDB.Exec(ctx, "UPDATE users SET subscription = ($1) WHERE email = ($2);",
			true,
			email,
	)
	if err != nil {
			log.Printf("Error happened when updating user subscription into pgx table. Err: %s", err)
			return err
	}
	
	return nil
}