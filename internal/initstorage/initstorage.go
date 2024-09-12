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
		"CREATE TABLE IF NOT EXISTS users (users_id int GENERATED ALWAYS AS IDENTITY PRIMARY KEY, username varchar NOT NULL, password varchar NOT NULL, email varchar NOT NULL, tokenhash varchar, category varchar NOT NULL, isverified varchar NOT NULL, subscription boolean NOT NULL, status varchar NOT NULL, last_edited_at timestamp NOT NULL, created_at timestamp NOT NULL)")
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

	// photos table
	_, err = db.Exec(ctx,
		"CREATE TABLE IF NOT EXISTS photos (photos_id int GENERATED ALWAYS AS IDENTITY PRIMARY KEY, link varchar NOT NULL, small_image varchar, uploaded_at timestamp NOT NULL, users_id int REFERENCES users(users_id))")
	if err != nil {
		log.Printf("Error happened when creating photos table. Err: %s", err)
		return nil, false

	}

	// background table
	_, err = db.Exec(ctx,
			"CREATE TABLE IF NOT EXISTS backgrounds (backgrounds_id int GENERATED ALWAYS AS IDENTITY PRIMARY KEY, link varchar NOT NULL, small_image varchar, category varchar)")
	if err != nil {
			log.Printf("Error happened when creating background table. Err: %s", err)
			return nil, false
	
	}

	// layout table
	_, err = db.Exec(ctx,
		"CREATE TABLE IF NOT EXISTS layouts (layouts_id int GENERATED ALWAYS AS IDENTITY PRIMARY KEY, count_images int NOT NULL, link varchar NOT NULL, small_image varchar, size varchar NOT NULL, data text NOT NULL)")
	if err != nil {
			log.Printf("Error happened when creating layout table. Err: %s", err)
			return nil, false

	}


	// decorations table
	_, err = db.Exec(ctx,
		"CREATE TABLE IF NOT EXISTS decorations (decorations_id int GENERATED ALWAYS AS IDENTITY PRIMARY KEY, small_image varchar, link varchar NOT NULL, type varchar, category varchar)")
	if err != nil {
			log.Printf("Error happened when creating decorations table. Err: %s", err)
			return nil, false

	}

	// projects table
	_, err = db.Exec(ctx,
		"CREATE TABLE IF NOT EXISTS projects (projects_id int GENERATED ALWAYS AS IDENTITY PRIMARY KEY, name varchar, category varchar, size varchar NOT NULL, variant varchar NOT NULL, cover varchar NOT NULL, paper varchar NOT NULL, preview_image_link varchar, count_pages int, status varchar NOT NULL, last_edited_at timestamp NOT NULL, last_editor int, created_at timestamp NOT NULL, preview_link varchar, print_link varchar, creating_spine_link varchar, preview_spine_link varchar, leather_id int, users_id int REFERENCES users(users_id))")
	if err != nil {
		log.Printf("Error happened when creating photobooks table. Err: %s", err)
		return nil, false

	}

	// templates table
	_, err = db.Exec(ctx,
			"CREATE TABLE IF NOT EXISTS templates (templates_id int GENERATED ALWAYS AS IDENTITY PRIMARY KEY, name varchar, status varchar NOT NULL, category varchar, size varchar, creating_spine_link varchar, preview_spine_link varchar, last_edited_at timestamp NOT NULL, last_editor int, created_at timestamp NOT NULL)")
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
		"CREATE TABLE IF NOT EXISTS pages (pages_id int GENERATED ALWAYS AS IDENTITY PRIMARY KEY, is_template boolean NOT NULL, last_edited_at timestamp NOT NULL, data text, sort int, preview_link varchar, creating_image_link varchar, type varchar, projects_id int NOT NULL)")
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


	// mapping decorations uploaded by users and users 
	_, err = db.Exec(ctx,
			"CREATE TABLE IF NOT EXISTS users_has_decoration (users_id int NOT NULL, decorations_id int NOT NULL, is_favourite boolean NOT NULL, is_personal boolean NOT NULL)")
	if err != nil {
			log.Printf("Error happened when creating users_has_decoration table. Err: %s", err)
			return nil, false
	
	}

	// mapping backgrounds uploaded by users and users 
	_, err = db.Exec(ctx,
			"CREATE TABLE IF NOT EXISTS users_has_backgrounds (users_id int NOT NULL, backgrounds_id int NOT NULL, is_favourite boolean NOT NULL, is_personal boolean NOT NULL)")
	if err != nil {
			log.Printf("Error happened when creating users_has_backgrounds table. Err: %s", err)
			return nil, false
	
	}

	// mapping layouts favoured by users and users 
	_, err = db.Exec(ctx,
			"CREATE TABLE IF NOT EXISTS users_has_layouts (users_id int NOT NULL, layouts_id int NOT NULL, is_favourite boolean NOT NULL)")
	if err != nil {
			log.Printf("Error happened when creating users_has_layouts table. Err: %s", err)
			return nil, false
	
	}

	// prices table
	_, err = db.Exec(ctx,
			"CREATE TABLE IF NOT EXISTS prices (prices_id int GENERATED ALWAYS AS IDENTITY PRIMARY KEY, cover varchar NOT NULL, variant varchar NOT NULL, surface varchar NOT NULL, size varchar NOT NULL, baseprice double precision NOT NULL, extrapage double precision NOT NULL)")
	if err != nil {
			log.Printf("Error happened when creating prices table. Err: %s", err)
			return nil, false
	
	}

	// table with leather colours
	_, err = db.Exec(ctx,
			"CREATE TABLE IF NOT EXISTS leather (leather_id int GENERATED ALWAYS AS IDENTITY PRIMARY KEY, colourlink varchar NOT NULL, hexcode varchar NOT NULL, description varchar)")
	if err != nil {
			log.Printf("Error happened when creating leather table. Err: %s", err)
			return nil, false

	}

	// promooffers table
	_, err = db.Exec(ctx,
			"CREATE TABLE IF NOT EXISTS promooffers (promooffers_id int GENERATED ALWAYS AS IDENTITY PRIMARY KEY, code varchar NOT NULL UNIQUE, discount double precision NOT NULL, category varchar NOT NULL, is_onetime boolean, is_used boolean, expires_at int NOT NULL, used_at timestamp, is_personal boolean, users_id int)")
	if err != nil {
			log.Printf("Error happened when creating promooffers table. Err: %s", err)
			return nil, false
	
	}

	// gift certificates table
	_, err = db.Exec(ctx,
		"CREATE TABLE IF NOT EXISTS giftcertificates (giftcertificates_id int GENERATED ALWAYS AS IDENTITY PRIMARY KEY, code varchar NOT NULL UNIQUE, initialdeposit float NOT NULL, status varchar NOT NULL, currentdeposit float NOT NULL, created_at timestamp NOT NULL, used_at timestamp, receipientemail varchar NOT NULL, reciepientname varchar NOT NULL, buyerfirstname varchar NOT NULL, buyerlastname varchar NOT NULL, buyeremail varchar NOT NULL, buyerphone varchar NOT NULL, mail_at int NOT NULL, mail_sent boolean)")
	if err != nil {
			log.Printf("Error happened when creating giftcertificates table. Err: %s", err)
			return nil, false

	}

	// orders table
	_, err = db.Exec(ctx,
		"CREATE TABLE IF NOT EXISTS orders (orders_id int GENERATED ALWAYS AS IDENTITY PRIMARY KEY, status varchar NOT NULL, created_at timestamp NOT NULL, last_updated_at timestamp NOT NULL, firstname varchar, lastname varchar, email varchar, phone varchar, commentary varchar, baseprice double precision, finalprice double precision, videolink varchar, package_box bool, promooffers_id int, giftcertificates_id int, giftcertificates_deposit float, delivery_id int, users_id int)")
	if err != nil {
		log.Printf("Error happened when creating orders table. Err: %s", err)
		return nil, false

	}

	// mapping orders with projects
	_, err = db.Exec(ctx,
		"CREATE TABLE IF NOT EXISTS orders_has_projects (orders_id int NOT NULL, projects_id int NOT NULL)")
	if err != nil {
			log.Printf("Error happened when creating orders_has_projects table. Err: %s", err)
			return nil, false

	}

	// transactions table
	_, err = db.Exec(ctx,
		"CREATE TABLE IF NOT EXISTS transactions (transactions_id int GENERATED ALWAYS AS IDENTITY PRIMARY KEY, status varchar NOT NULL, created_at timestamp NOT NULL, paymentmethod varchar NOT NULL, amount double precision NOT NULL, bankorderid varchar NOT NULL, bankstatus varchar)")
	if err != nil {
		log.Printf("Error happened when creating transactions table. Err: %s", err)
		return nil, false

	}


	// mapping giftcertificates with transactions
	_, err = db.Exec(ctx,
			"CREATE TABLE IF NOT EXISTS giftcertificates_has_transactions (giftcertificates_id int NOT NULL, transactions_id int NOT NULL)")
	if err != nil {
				log.Printf("Error happened when creating giftcertificates_has_transactions table. Err: %s", err)
				return nil, false
	
	}

	// mapping orders with transactions
	_, err = db.Exec(ctx,
			"CREATE TABLE IF NOT EXISTS orders_has_transactions (orders_id int NOT NULL, transactions_id int NOT NULL)")
	if err != nil {
				log.Printf("Error happened when creating orders_has_transactions table. Err: %s", err)
				return nil, false
	
	}



	// delivery table
	_, err = db.Exec(ctx,"CREATE TABLE IF NOT EXISTS delivery (delivery_id int GENERATED ALWAYS AS IDENTITY PRIMARY KEY, status varchar NOT NULL, created_at timestamp NOT NULL, expected_delivery_from timestamp, expected_delivery_to timestamp, method varchar NOT NULL, address varchar, postal_code varchar, code varchar, amount double precision NOT NULL, deliveryid varchar, trackingnumber varchar, deliverystatus varchar)")
	if err != nil {	

			log.Printf("Error happened when creating transactions table. Err: %s", err)
			return nil, false
	
	}
	

	//pass default settings
	//_, err = db.Exec(ctx, "ALTER TABLE photos ADD COLUMN small_image varchar;")
	//if err != nil {
	//		log.Printf("Error happened when creating layout table. Err: %s", err)
	//		return nil, false
	//}
	//_, err = db.Exec(ctx, "ALTER TABLE backgrounds ADD COLUMN small_image varchar;")
	//if err != nil {
	//		log.Printf("Error happened when creating layout table. Err: %s", err)
	//		return nil, false
	//}
	//_, err = db.Exec(ctx, "ALTER TABLE decorations ADD COLUMN small_image varchar;")
	//if err != nil {
	//		log.Printf("Error happened when creating layout table. Err: %s", err)
	//		return nil, false
	//}
	_, err = db.Exec(ctx, "ALTER TABLE decorations ADD COLUMN small_image varchar;")
	if err != nil {
			log.Printf("Error happened when creating layout table. Err: %s", err)
			return nil, false
	}
	//_, err = db.Exec(ctx, "ALTER TABLE projects ADD COLUMN creating_spine_link varchar;")
	//if err != nil {
	//		log.Printf("Error happened when creating layout table. Err: %s", err)
	//		return nil, false
	//}
	//_, err = db.Exec(ctx, "ALTER TABLE projects ADD COLUMN preview_spine_link varchar;")
	//if err != nil {
	//		log.Printf("Error happened when creating layout table. Err: %s", err)
	//		return nil, false
	//}

	//_, err = db.Exec(ctx, "ALTER TABLE templates ADD COLUMN creating_spine_link varchar;")
	//if err != nil {
	//		log.Printf("Error happened when creating layout table. Err: %s", err)
	//		return nil, false
	//}
	//_, err = db.Exec(ctx, "ALTER TABLE templates ADD COLUMN preview_spine_link varchar;")
	//if err != nil {
	//		log.Printf("Error happened when creating layout table. Err: %s", err)
	//		return nil, false
	//}
	
	//_, err = db.Exec(ctx, "ALTER TABLE users ADD COLUMN subscription boolean;")
	//if err != nil {
	//		log.Printf("Error happened when creating subscription column. Err: %s", err)
	//		return nil, false
	//}

	//_, err = db.Exec(ctx, "ALTER TABLE promooffers ADD COLUMN category varchar;")
	//if err != nil {
	//		log.Printf("Error happened when creating subscription column. Err: %s", err)
	//		return nil, false
	//}
	//_, err = db.Exec(ctx, "ALTER TABLE promooffers ADD COLUMN users_id int;")
	//if err != nil {
	//		log.Printf("Error happened when creating subscription column. Err: %s", err)
	//		return nil, false
	//}

	//_, err = db.Exec(ctx, "ALTER TABLE promooffers ADD COLUMN expires_at int;")
	//if err != nil {
	//		log.Printf("Error happened when creating subscription column. Err: %s", err)
	//		return nil, false
	//}

	//_, err = db.Exec(ctx, "ALTER TABLE projects ADD COLUMN category varchar;")
	//if err != nil {
	//		log.Printf("Error happened when creating subscription column. Err: %s", err)
	//		return nil, false
	//}

	//_, err = db.Exec(ctx, "ALTER TABLE projects ADD COLUMN leather_id int;")
	//if err != nil {
	//		log.Printf("Error happened when creating subscription column. Err: %s", err)
	//		return nil, false
	//}

	log.Println("Initialised data table.")
	

	return db, true

	
}


