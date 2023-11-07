// Storage package contains functions for storing photos and projects in a SQL database.
//
// Available at https://github.com/SiberianMonster/memoryprint-rus/tree/development/internal/initstorage
package initstorage

import (
	"context"
	"log"
	"time"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Config(connStr *string) (*pgxpool.Config) {
	const defaultMaxConns = int32(4)
	const defaultMinConns = int32(0)
	const defaultMaxConnLifetime = time.Hour
	const defaultMaxConnIdleTime = time.Minute * 30
	const defaultHealthCheckPeriod = time.Minute
	const defaultConnectTimeout = time.Second * 5
	 
   
	dbConfig, err := pgxpool.ParseConfig(*connStr)
	if err!=nil {
	 log.Fatal("Failed to create a config, error: ", err)
	}
   
	dbConfig.MaxConns = defaultMaxConns
	dbConfig.MinConns = defaultMinConns
	dbConfig.MaxConnLifetime = defaultMaxConnLifetime
	dbConfig.MaxConnIdleTime = defaultMaxConnIdleTime
	dbConfig.HealthCheckPeriod = defaultHealthCheckPeriod
	dbConfig.ConnConfig.ConnectTimeout = defaultConnectTimeout
   
	dbConfig.BeforeAcquire = func(ctx context.Context, c *pgx.Conn) bool {
	 log.Println("Before acquiring the connection pool to the database!!")
	 return true
	}
   
	dbConfig.AfterRelease = func(c *pgx.Conn) bool {
	 log.Println("After releasing the connection pool to the database!!")
	 return true
	}
   
	dbConfig.BeforeClose = func(c *pgx.Conn) {
	 log.Println("Closed the connection pool to the database!!")
	}
   
	return dbConfig
   }

// SetUpDbConnection initializes database connection.
func SetUpDBConnection(ctx context.Context, connStr *string) (*pgxpool.Pool, bool) {

	log.Println("Start db connection.")


	// Create database connection
	db, err := pgxpool.NewWithConfig(ctx, Config(connStr))
	if err!=nil {
	 log.Fatal("Error while creating connection to the database!!")
	} 
   
	connection, err := db.Acquire(ctx)
	if err!=nil {
		log.Println(err)
	 	log.Fatal("Error while acquiring connection from the database pool!!")
	} 
	defer connection.Release()
   
	err = connection.Ping(ctx)
	if err!=nil{
	 log.Fatal("Could not ping database")
	}

	
	log.Println("Connection initialised successfully.")


	// users table
	_, err = db.Exec(ctx,
		"CREATE TABLE IF NOT EXISTS users (users_id int GENERATED ALWAYS AS IDENTITY PRIMARY KEY, username varchar NOT NULL UNIQUE, password varchar NOT NULL, email varchar NOT NULL, tokenhash varchar, category varchar NOT NULL, isverified varchar NOT NULL, status varchar NOT NULL, last_edited_at timestamp NOT NULL, created_at timestamp NOT NULL)")
	if err != nil {
		log.Printf("Error happened when creating users table. Err: %s", err)
		return nil, false

	}

	// verification table
	_, err = db.Exec(ctx,
		"CREATE TABLE IF NOT EXISTS verifications (verifications_id int GENERATED ALWAYS AS IDENTITY PRIMARY KEY, email varchar NOT NULL, code  varchar NOT NULL, expires_at timestamp NOT NULL, type int NOT NULL)")
	if err != nil {
		log.Printf("Error happened when creating verification table. Err: %s", err)
		return nil, false

	}

	// prices table
	_, err = db.Exec(ctx,
			"CREATE TABLE IF NOT EXISTS prices (prices_id int GENERATED ALWAYS AS IDENTITY PRIMARY KEY, price double precision NOT NULL, pagesnum int NOT NULL, priceperpage double precision NOT NULL, covertype varchar, bindingtype varchar, papertype varchar)")
	if err != nil {
			log.Printf("Error happened when creating orders table. Err: %s", err)
			return nil, false
	
	}

	// promooffers table
	_, err = db.Exec(ctx,
			"CREATE TABLE IF NOT EXISTS promooffers (promooffers_id int GENERATED ALWAYS AS IDENTITY PRIMARY KEY, code varchar NOT NULL UNIQUE, discount double precision NOT NULL, is_onetime boolean, is_used boolean, expires_at timestamp NOT NULL, used_at timestamp NOT NULL)")
	if err != nil {
			log.Printf("Error happened when creating promooffers table. Err: %s", err)
			return nil, false
	
	}

	// orders table
	_, err = db.Exec(ctx,
		"CREATE TABLE IF NOT EXISTS orders (orders_id int GENERATED ALWAYS AS IDENTITY PRIMARY KEY, link varchar NOT NULL, status varchar NOT NULL, pagesnum int, covertype varchar, bindingtype varchar, papertype varchar, created_at timestamp NOT NULL, last_updated_at timestamp NOT NULL, promooffers_id int REFERENCES promooffers(promooffers_id), users_id int REFERENCES users(users_id))")
	if err != nil {
		log.Printf("Error happened when creating orders table. Err: %s", err)
		return nil, false

	}

	// photos table
	_, err = db.Exec(ctx,
		"CREATE TABLE IF NOT EXISTS photos (photos_id int GENERATED ALWAYS AS IDENTITY PRIMARY KEY, link varchar NOT NULL, uploaded_at timestamp NOT NULL, users_id int REFERENCES users(users_id))")
	if err != nil {
		log.Printf("Error happened when creating photos table. Err: %s", err)
		return nil, false

	}

	// background table
	_, err = db.Exec(ctx,
			"CREATE TABLE IF NOT EXISTS backgrounds (backgrounds_id int GENERATED ALWAYS AS IDENTITY PRIMARY KEY, link varchar NOT NULL, category varchar NOT NULL)")
	if err != nil {
			log.Printf("Error happened when creating background table. Err: %s", err)
			return nil, false
	
	}

	// layout table
	_, err = db.Exec(ctx,
		"CREATE TABLE IF NOT EXISTS layouts (layouts_id int GENERATED ALWAYS AS IDENTITY PRIMARY KEY, orientation varchar NOT NULL, link varchar NOT NULL, category varchar NOT NULL)")
	if err != nil {
			log.Printf("Error happened when creating layout table. Err: %s", err)
			return nil, false

	}

	// decorations table
	_, err = db.Exec(ctx,
		"CREATE TABLE IF NOT EXISTS decorations (decorations_id int GENERATED ALWAYS AS IDENTITY PRIMARY KEY, link varchar NOT NULL, type varchar NOT NULL, category varchar NOT NULL)")
	if err != nil {
			log.Printf("Error happened when creating decorations table. Err: %s", err)
			return nil, false

	}

	// projects table
	_, err = db.Exec(ctx,
		"CREATE TABLE IF NOT EXISTS projects (projects_id int GENERATED ALWAYS AS IDENTITY PRIMARY KEY, name varchar, orientation varchar NOT NULL, cover_image varchar, status varchar NOT NULL, is_template boolean, category varchar, last_edited_at timestamp NOT NULL, last_editor int, created_at timestamp NOT NULL, users_id int REFERENCES users(users_id))")
	if err != nil {
		log.Printf("Error happened when creating photobooks table. Err: %s", err)
		return nil, false

	}

	// photoalbums table
	_, err = db.Exec(ctx,
		"CREATE TABLE IF NOT EXISTS albums (albums_id int GENERATED ALWAYS AS IDENTITY PRIMARY KEY, name varchar, last_edited_at timestamp NOT NULL, created_at timestamp NOT NULL, users_id int REFERENCES users(users_id))")
	if err != nil {
		log.Printf("Error happened when creating photoalbums table. Err: %s", err)
		return nil, false

	}

	// pages table
	_, err = db.Exec(ctx,
		"CREATE TABLE IF NOT EXISTS pages (pages_id int GENERATED ALWAYS AS IDENTITY PRIMARY KEY, number int NOT NULL, last_edited_at timestamp NOT NULL, projects_id int REFERENCES projects(projects_id))")
	if err != nil {
		log.Printf("Error happened when creating pages table. Err: %s", err)
		return nil, false

	}

	// mapping photobooks and users with editing permission
	_, err = db.Exec(ctx,
		"CREATE TABLE IF NOT EXISTS users_edit_projects (users_id int, email varchar, projects_id int NOT NULL, category varchar NOT NULL)")
	if err != nil {
		log.Printf("Error happened when creating users_edit_projects table. Err: %s", err)
		return nil, false

	}

	// mapping orders and printing agency with editing permission table
	_, err = db.Exec(ctx,
		"CREATE TABLE IF NOT EXISTS pa_has_orders (users_id int NOT NULL, orders_id int NOT NULL )")
	if err != nil {
		log.Printf("Error happened when creating pa_has_orders table. Err: %s", err)
		return nil, false

	}

	// mapping photoalbums and users with editing permission
	_, err = db.Exec(ctx,
			"CREATE TABLE IF NOT EXISTS users_edit_albums (users_id int NOT NULL, albums_id int NOT NULL, category varchar NOT NULL)")
	if err != nil {
			log.Printf("Error happened when creating users_edit_albums table. Err: %s", err)
			return nil, false
	
	}


	// mapping photobooks and pages
	_, err = db.Exec(ctx,
		"CREATE TABLE IF NOT EXISTS project_has_pages (projects_id int NOT NULL, pages_id int NOT NULL)")
	if err != nil {
		log.Printf("Error happened when creating project_has_pages table. Err: %s", err)
		return nil, false

	}

	// mapping pages and photos
	_, err = db.Exec(ctx,
		"CREATE TABLE IF NOT EXISTS page_has_photos (pages_id int NOT NULL, photos_id int NOT NULL, ptop double precision, pleft double precision, style varchar, last_edited_at timestamp NOT NULL)")
	if err != nil {
		log.Printf("Error happened when creating page_has_photos table. Err: %s", err)
		return nil, false

	}

	// mapping pages and layout
	_, err = db.Exec(ctx,
		"CREATE TABLE IF NOT EXISTS page_has_layout (pages_id int NOT NULL UNIQUE, layouts_id int NOT NULL, last_edited_at timestamp NOT NULL)")
	if err != nil {
		log.Printf("Error happened when creating page_has_layout table. Err: %s", err)
		return nil, false

	}

	// mapping pages and background
	_, err = db.Exec(ctx,
		"CREATE TABLE IF NOT EXISTS page_has_background (pages_id int NOT NULL UNIQUE, backgrounds_id int NOT NULL, last_edited_at timestamp NOT NULL)")
	if err != nil {
		log.Printf("Error happened when creating page_has_background table. Err: %s", err)
		return nil, false

	}

	// mapping pages and decoration
	_, err = db.Exec(ctx,
		"CREATE TABLE IF NOT EXISTS page_has_decoration (pages_id int NOT NULL, decorations_id int NOT NULL, ptop double precision, pleft double precision, style varchar, last_edited_at timestamp NOT NULL)")
	if err != nil {
		log.Printf("Error happened when creating page_has_background table. Err: %s", err)
		return nil, false

	}

	// custom user text
	_, err = db.Exec(ctx,
		"CREATE TABLE IF NOT EXISTS page_has_text (pages_id int NOT NULL, custom_text varchar NOT NULL, ptop double precision, pleft double precision, style varchar)")
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


