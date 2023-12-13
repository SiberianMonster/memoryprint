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
	"log"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var err error


func Hash(value, key string) (string, error) {
	mac := hmac.New(sha256.New, []byte(key))
	_, err := mac.Write([]byte(value))
	return fmt.Sprintf("%x", mac.Sum(nil)), err
}

func CheckUser(ctx context.Context, storeDB *pgxpool.Pool, u models.User) bool {

	var userBool bool
	err := storeDB.QueryRow(ctx, "SELECT EXISTS (SELECT users_id FROM users WHERE email = ($1));", u.Email).Scan(&userBool)
	if err != nil {
		log.Printf("Error happened when checking if user is in db. Err: %s", err)
		return userBool
	}

	return userBool
}

func GetUserData(ctx context.Context, storeDB *pgxpool.Pool, userID uint) (models.User, error) {

	var dbUser models.User
	err := storeDB.QueryRow(ctx, "SELECT username, email, tokenhash FROM users WHERE users_id = ($1);", userID).Scan(&dbUser.Username, &dbUser.Email, &dbUser.TokenHash)
	if err != nil {
		log.Printf("Error happened when checking if user is in db. Err: %s", err)
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

func CreateUser(ctx context.Context, storeDB *pgxpool.Pool, u models.User) (uint, error) {

	var userID uint
	t := time.Now()
	pwdHash, err := Hash(fmt.Sprintf("%s:password", u.Password), config.Key)
	if err != nil {
		log.Printf("Error happened when hashing received value. Err: %s", err)
		return userID, err
	}
	tokenHash := emailutils.GenerateRandomString(15)

	_, err = storeDB.Exec(ctx, "INSERT INTO users (username, password, email, tokenhash, category, status, isverified, last_edited_at, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);",
		u.Username,
		pwdHash,
		u.Email,
		tokenHash,
		u.Category,
		u.Status,
		models.UnverifiedStatus,
		t,
		t,
	)
	if err != nil {
		log.Printf("Error happened when inserting a new user entry into pgx table. Err: %s", err)
		return userID, err
	}
	err = storeDB.QueryRow(ctx, "SELECT users_id FROM users WHERE username=($1);", u.Username).Scan(&userID)
	if err != nil {
		log.Printf("Error happened when retrieving usersid from the db. Err: %s", err)
		return userID, err
	}
	return userID, nil
}

func CheckCredentials(ctx context.Context, storeDB *pgxpool.Pool, u models.User) (models.User, error) {

	var dbUser models.User
	
	err := storeDB.QueryRow(ctx, "SELECT username, email, password, tokenhash, users_id FROM users WHERE email=($1);", u.Email).Scan(&dbUser.Username, &dbUser.Email, &dbUser.Password, &dbUser.TokenHash, &dbUser.ID)
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


// UpdatePassword updates the user password
func UpdatePassword(ctx context.Context, storeDB *pgxpool.Pool, user models.User) (uint, error) {

	if !CheckUser(ctx, storeDB, user) {
		return user.ID, nil
	}
	t := time.Now()
	pwdHash, err := Hash(fmt.Sprintf("%s:password", user.Password), config.Key)
	if err != nil {
		log.Printf("Error happened when hashing received value. Err: %s", err)
		return user.ID, err
	}

	_, err = storeDB.Exec(ctx, "UPDATE users SET password = ($1), tokenhash = ($2), last_edited_at = ($3) WHERE users_id = ($4);",
		pwdHash,
		user.TokenHash,
		t,
		user.ID,
	)
	if err != nil {
		log.Printf("Error happened when updating user credentials into pgx table. Err: %s", err)
		return user.ID, err
	}
	err = storeDB.QueryRow(ctx, "SELECT users_id FROM users WHERE username=($1);", user.Username).Scan(&user.ID)
	if err != nil {
		log.Printf("Error happened when retrieving usersid from the db. Err: %s", err)
		return user.ID, err
	}
	return user.ID, nil
}

func UpdateUserCategory(ctx context.Context, storeDB *pgxpool.Pool, u models.User) (uint, error) {

	if !CheckUser(ctx, storeDB, u) {
		return u.ID, nil
	}

	_, err = storeDB.Exec(ctx, "UPDATE users SET category = ($1) WHERE username = ($2);",
		u.Category,
		u.Username,
	)
	if err != nil {
		log.Printf("Error happened when updating user category into pgx table. Err: %s", err)
		return u.ID, err
	}
	err = storeDB.QueryRow(ctx, "SELECT users_id FROM users WHERE username=($1);", u.Username).Scan(&u.ID)
	if err != nil {
		log.Printf("Error happened when retrieving usersid from the db. Err: %s", err)
		return u.ID, err
	}
	return u.ID, nil
}

func UpdateUserStatus(ctx context.Context, storeDB *pgxpool.Pool, u models.User) (uint, error) {

	if !CheckUser(ctx, storeDB, u) {
		return u.ID, nil
	}

	_, err = storeDB.Exec(ctx, "UPDATE users SET status = ($1) WHERE username = ($2);",
		u.Status,
		u.Username,
	)
	if err != nil {
		log.Printf("Error happened when updating user status into pgx table. Err: %s", err)
		return u.ID, err
	}
	err = storeDB.QueryRow(ctx, "SELECT users_id FROM users WHERE username=($1);", u.Username).Scan(&u.ID)
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
		if err = rows.Scan(&user.Username, &user.Password, &user.Email, &user.Category,
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
