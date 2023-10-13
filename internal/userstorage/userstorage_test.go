package userstorage

import (
	"context"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/SiberianMonster/memoryprint/internal/config"
	"github.com/SiberianMonster/memoryprint/internal/models"
	"regexp"
	"testing"
)

func TestCheckUser(t *testing.T) {

	newUser := models.User{
		Username: "Name",
		Password: "MyPass123",
	}

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), config.ContextDBTimeout)
	// не забываем освободить ресурс
	defer cancel()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT username FROM users WHERE username = ($1)`)).WithArgs(newUser.Username)

	// now we execute our method
	_ = CheckUser(ctx, db, newUser)

	// we make sure that all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}