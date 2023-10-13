// Storage package contains functions for storing photos and projects in a SQL database.
//
// Available at https://github.com/SiberianMonster/memoryprint-rus/tree/development/internal/initstorage
package initstorage

import (
	"context"
	"log"
	"os"
	"time"
	"github.com/sirupsen/logrus"
	"github.com/jackc/pgx/v5/pgxpool"
    "github.com/jackc/pgx/v4/log/logrusadapter"

	"github.com/jackc/pgx/v5"
)


// SetUpDbConnection initializes database connection.
func SetUpDBConnection(ctx context.Context, connStr *string) (*pgxpool.Pool, bool) {

	log.Println("Start db connection.")


	config, err := pgxpool.ParseConfig(*connStr)
	if err != nil {
		log.Printf("Error happened when parsing db config. Err: %s", err)
		return nil, false
	}
	looger := &logrus.Logger{
		Out:          os.Stderr,
		Formatter:    new(logrus.JSONFormatter),
		Hooks:        make(logrus.LevelHooks),
		Level:        logrus.InfoLevel,
		ExitFunc:     os.Exit,
		ReportCaller: false,
	}
	config.ConnConfig.Logger = logrusadapter.NewLogger(looger)
	config.ConnConfig.RuntimeParams = map[string]string{
		"standard_conforming_strings": "on",
	}
	config.ConnConfig.PreferSimpleProtocol = true 
	config.ConnConfig.CustomCancel = func(_ *pgx.Conn) error { return nil }

	db, err := *pgxpool.ConnectConfig(ctx, config)
	if err != nil {
		log.Printf("Error happened when initiating connection to the db. Err: %s", err)
		return nil, false
	}
	log.Println("Connection initialised successfully.")

	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifeTime(5 * time.Minute)
	db.SetConnIdleTime(5 * time.Minute)

	// users table
	_, err = db.Exec(ctx,
		"CREATE TABLE IF NOT EXISTS users (users_id int GENERATED ALWAYS AS IDENTITY PRIMARY KEY, username varchar NOT NULL UNIQUE, password varchar NOT NULL, email varchar NOT NULL, tokenhash varchar, category varchar NOT NULL, isverified varchar NOT NULL, status varchar NOT NULL)")
	if err != nil {
		log.Printf("Error happened when creating users table. Err: %s", err)
		return nil, false

	}

	// verification table
	_, err = db.Exec(ctx,
		"CREATE TABLE IF NOT EXISTS verifications (verifications_id int GENERATED ALWAYS AS IDENTITY PRIMARY KEY, email varchar NOT NULL, code  varchar NOT NULL, expires_at timestamp NOT NULL, type varchar NOT NULL)")
	if err != nil {
		log.Printf("Error happened when creating verification table. Err: %s", err)
		return nil, false

	}

	// orders table
	_, err = db.Exec(ctx,
		"CREATE TABLE IF NOT EXISTS orders (orders_id int GENERATED ALWAYS AS IDENTITY PRIMARY KEY, link varchar NOT NULL, status varchar NOT NULL, pagesnum int, covertype varchar, bindingtype varchar, orientation varchar, uploaded_at timestamp NOT NULL, last_updated_at timestamp NOT NULL, users_id int REFERENCES users(users_id))")
	if err != nil {
		log.Printf("Error happened when creating orders table. Err: %s", err)
		return nil, false

	}

	// photos table
	_, err = db.Exec(ctx,
		"CREATE TABLE IF NOT EXISTS photos (photo_id int GENERATED ALWAYS AS IDENTITY PRIMARY KEY, link varchar NOT NULL, uploaded_at timestamp NOT NULL, users_id int REFERENCES users(users_id))")
	if err != nil {
		log.Printf("Error happened when creating photos table. Err: %s", err)
		return nil, false

	}

	// background table
	_, err = db.Exec(ctx,
			"CREATE TABLE IF NOT EXISTS backgrounds (background_id int GENERATED ALWAYS AS IDENTITY PRIMARY KEY, link varchar NOT NULL, category varchar NOT NULL)")
	if err != nil {
			log.Printf("Error happened when creating background table. Err: %s", err)
			return nil, false
	
	}

	// layout table
	_, err = db.Exec(ctx,
		"CREATE TABLE IF NOT EXISTS layouts (layout_id int GENERATED ALWAYS AS IDENTITY PRIMARY KEY, link varchar NOT NULL, category varchar NOT NULL)")
	if err != nil {
			log.Printf("Error happened when creating layout table. Err: %s", err)
			return nil, false

	}

	// decorations table
	_, err = db.Exec(ctx,
		"CREATE TABLE IF NOT EXISTS decorations (decoration_id int GENERATED ALWAYS AS IDENTITY PRIMARY KEY, link varchar NOT NULL, type varchar NOT NULL, category varchar NOT NULL)")
	if err != nil {
			log.Printf("Error happened when creating decorations table. Err: %s", err)
			return nil, false

	}

	// photobooks table
	_, err = db.Exec(ctx,
		"CREATE TABLE IF NOT EXISTS projects (project_id int GENERATED ALWAYS AS IDENTITY PRIMARY KEY, name varchar, cover_image varchar, status varchar NOT NULL, last_edited_at timestamp NOT NULL, last_editor int, created_at timestamp NOT NULL, users_id int REFERENCES users(users_id))")
	if err != nil {
		log.Printf("Error happened when creating photobooks table. Err: %s", err)
		return nil, false

	}

	// photoalbums table
	_, err = db.Exec(ctx,
		"CREATE TABLE IF NOT EXISTS albums (album_id int GENERATED ALWAYS AS IDENTITY PRIMARY KEY, name varchar, last_edited_at timestamp NOT NULL, created_at timestamp NOT NULL, users_id int REFERENCES users(users_id))")
	if err != nil {
		log.Printf("Error happened when creating photoalbums table. Err: %s", err)
		return nil, false

	}

	// pages table
	_, err = db.Exec(ctx,
		"CREATE TABLE IF NOT EXISTS pages (page_id int GENERATED ALWAYS AS IDENTITY PRIMARY KEY, number int NOT NULL, last_edited_at timestamp NOT NULL, project_id int REFERENCES projects(project_id))")
	if err != nil {
		log.Printf("Error happened when creating pages table. Err: %s", err)
		return nil, false

	}

	// mapping photobooks and users with editing permission
	_, err = db.Exec(ctx,
		"CREATE TABLE IF NOT EXISTS users_edit_projects (users_id int, email varchar, project_id int NOT NULL, category varchar NOT NULL)")
	if err != nil {
		log.Printf("Error happened when creating users_edit_projects table. Err: %s", err)
		return nil, false

	}

	// mapping orders and printing agency with editing permission table
	_, err = db.Exec(ctx,
		"CREATE TABLE IF NOT EXISTS pa_has_orders (users_id int NOT NULL, order_id int NOT NULL )")
	if err != nil {
		log.Printf("Error happened when creating pa_has_orders table. Err: %s", err)
		return nil, false

	}

	// mapping photoalbums and users with editing permission
	_, err = db.Exec(ctx,
			"CREATE TABLE IF NOT EXISTS users_edit_albums (users_id int NOT NULL, album_id int NOT NULL, category varchar NOT NULL)")
	if err != nil {
			log.Printf("Error happened when creating users_edit_albums table. Err: %s", err)
			return nil, false
	
	}


	// mapping photobooks and pages
	_, err = db.Exec(ctx,
		"CREATE TABLE IF NOT EXISTS project_has_pages (project_id int NOT NULL, page_id int NOT NULL)")
	if err != nil {
		log.Printf("Error happened when creating project_has_pages table. Err: %s", err)
		return nil, false

	}

	// mapping pages and photos
	_, err = db.Exec(ctx,
		"CREATE TABLE IF NOT EXISTS page_has_photos (page_id int NOT NULL, photo_id int NOT NULL, ptop double precision, pleft double precision, style varchar, last_edited_at timestamp NOT NULL)")
	if err != nil {
		log.Printf("Error happened when creating page_has_photos table. Err: %s", err)
		return nil, false

	}

	// mapping pages and layout
	_, err = db.Exec(ctx,
		"CREATE TABLE IF NOT EXISTS page_has_layout (page_id int NOT NULL UNIQUE, layout_id int NOT NULL, last_edited_at timestamp NOT NULL)")
	if err != nil {
		log.Printf("Error happened when creating page_has_layout table. Err: %s", err)
		return nil, false

	}

	// mapping pages and background
	_, err = db.Exec(ctx,
		"CREATE TABLE IF NOT EXISTS page_has_background (page_id int NOT NULL UNIQUE, background_id int NOT NULL, last_edited_at timestamp NOT NULL)")
	if err != nil {
		log.Printf("Error happened when creating page_has_background table. Err: %s", err)
		return nil, false

	}

	// mapping pages and decoration
	_, err = db.Exec(ctx,
		"CREATE TABLE IF NOT EXISTS page_has_decoration (page_id int NOT NULL, decoration_id int NOT NULL, ptop double precision, pleft double precision, style varchar, last_edited_at timestamp NOT NULL)")
	if err != nil {
		log.Printf("Error happened when creating page_has_background table. Err: %s", err)
		return nil, false

	}

	// custom user text
	_, err = db.Exec(ctx,
		"CREATE TABLE IF NOT EXISTS page_has_text (page_id int NOT NULL, custom_text varchar NOT NULL, ptop double precision, pleft double precision, style varchar)")
	if err != nil {
		log.Printf("Error happened when creating page_has_text table. Err: %s", err)
		return nil, false

	}

	// prices table
	_, err = db.Exec(ctx,
			"CREATE TABLE IF NOT EXISTS prices (prices_id int GENERATED ALWAYS AS IDENTITY PRIMARY KEY, covertype varchar, bindingtype varchar, price double precision)")
	if err != nil {
			log.Printf("Error happened when creating prices table. Err: %s", err)
			return nil, false
	
	}

	// pass default settings
	

	log.Println("Initialised data table.")
	

	return db, true

	
}


