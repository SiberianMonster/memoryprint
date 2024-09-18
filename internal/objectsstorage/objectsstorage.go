// Storage package contains functions for storing photos and projects in a pgx database.
//
// Available at https://github.com/SiberianMonster/memoryprint/tree/development/internal/objectsstorage
package objectsstorage

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strconv"
	"time"
	"github.com/SiberianMonster/memoryprint/internal/models"
	"database/sql"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var err error

func CheckUserOwnsPhoto(ctx context.Context, storeDB *pgxpool.Pool, userID uint, photoID uint) bool {

	var checkPhoto bool
	err := storeDB.QueryRow(ctx, "SELECT CASE WHEN EXISTS (SELECT * FROM photos WHERE photos_id = ($1) AND users_id = ($2)) THEN TRUE ELSE FALSE END;", photoID, userID).Scan(&checkPhoto)
	if err != nil && err != pgx.ErrNoRows {
		log.Printf("Error happened when checking if user can edit photo in db. Err: %s", err)
		return false
	}

	return checkPhoto
}

func CheckDecorISPersonal(ctx context.Context, storeDB *pgxpool.Pool, decorationID uint) bool {
	var checkDecor bool
	err := storeDB.QueryRow(ctx, "SELECT CASE WHEN EXISTS (SELECT * FROM users_has_decoration WHERE decorations_id = ($1) AND is_personal = ($2)) THEN TRUE ELSE FALSE END;", decorationID, true).Scan(&checkDecor)
	if err != nil && err != pgx.ErrNoRows {
		log.Printf("Error happened when checking if decor is personal in db. Err: %s", err)
		return false
	}

	return checkDecor
}

func CheckBackgroundISPersonal(ctx context.Context, storeDB *pgxpool.Pool, backgroundID uint) bool {
	var checkB bool
	err := storeDB.QueryRow(ctx, "SELECT CASE WHEN EXISTS (SELECT * FROM users_has_backgrounds WHERE backgrounds_id = ($1) AND is_personal = ($2)) THEN TRUE ELSE FALSE END;", backgroundID, true).Scan(&checkB)
	if err != nil && err != pgx.ErrNoRows {
		log.Printf("Error happened when checking if decor is personal in db. Err: %s", err)
		return false
	}

	return checkB
}

// AddPhoto function performs the operation of adding photos to the db.
func AddPhoto(ctx context.Context, storeDB *pgxpool.Pool, photoLink string, smallImage string, userID uint) (uint, error) {

	var photoID uint
	t := time.Now()
	err = storeDB.QueryRow(ctx, "INSERT INTO photos (link, small_image, uploaded_at, users_id) VALUES ($1, $2, $3, $4) RETURNING photos_id;",
		photoLink,
		smallImage,
		t,
		false,
		userID,
	).Scan(&photoID)
	if err != nil {
		log.Printf("Error happened when inserting a new photo entry into pgx table. Err: %s", err)
		return photoID, err
	}

	return photoID, nil

}

// AddDecoration function performs the operation of adding decoration to the db.
func AddDecoration(ctx context.Context, storeDB *pgxpool.Pool, newDecor models.PersonalisedObject, userID uint) (uint, error) {

	var decorID uint
	_, err = storeDB.Exec(ctx, "INSERT INTO decorations (link, small_image, category, type) VALUES ($1, $2, $3, $4);",
		newDecor.Link,
		newDecor.SmallImage,
		newDecor.Category,
		newDecor.Type,
	)
	if err != nil {
		log.Printf("Error happened when inserting a new decoration entry into pgx table. Err: %s", err)
		return decorID, err
	}
	err = storeDB.QueryRow(ctx, "SELECT decorations_id FROM decorations WHERE link=($1);", newDecor.Link).Scan(&decorID)
	if err != nil {
		log.Printf("Error happened when retrieving usersid from the db. Err: %s", err)
		return decorID, err
	}
	_, err = storeDB.Exec(ctx, "INSERT INTO users_has_decoration (users_id, decorations_id, is_favourite, is_personal) VALUES ($1, $2, $3, $4);",
		userID,
		decorID,
		newDecor.IsFavourite,
		true,
	)
	if err != nil {
		log.Printf("Error happened when inserting a new entry into user_has_decoration pgx table. Err: %s", err)
		return decorID, err
	}

	return decorID, nil

}

// AdminDeleteDecoration function performs the operation of deleting decoration from the db.
func AdminDeleteDecoration(ctx context.Context, storeDB *pgxpool.Pool, dID uint) (error) {

	_, err = storeDB.Exec(ctx, "DELETE FROM decorations WHERE decorations_id=($1);",
		dID,
	)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("Error happened when deleting decoration from decorations pgx table. Err: %s", err)
		return err
	}

	_, err = storeDB.Exec(ctx, "DELETE FROM users_has_decoration WHERE decorations_id=($1);",
		dID,
	)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("Error happened when deleting decoration from users_has_decoration pgx table. Err: %s", err)
		return err
	}


	return nil

}

// DeleteDecoration function performs the operation of deleting decoration from the db.
func DeleteDecoration(ctx context.Context, storeDB *pgxpool.Pool, userID uint, decorID uint) (error) {

	var isPersonal bool

	err = storeDB.QueryRow(ctx, "SELECT is_personal FROM users_has_decoration WHERE decorations_id=($1) AND users_id=($2);", decorID, userID).Scan(&isPersonal)
	if err != nil || !isPersonal{
		log.Printf("The decoration does not belong to user. Err: %s", err)
		return err
	}

	_, err = storeDB.Exec(ctx, "DELETE FROM decorations WHERE decorations_id=($1);",
		decorID,
	)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("Error happened when deleting decoration from decorations pgx table. Err: %s", err)
		return err
	}

	_, err = storeDB.Exec(ctx, "DELETE FROM users_has_decoration WHERE decorations_id=($1);",
		decorID,
	)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("Error happened when deleting decoration from users_has_decoration pgx table. Err: %s", err)
		return err
	}


	return nil

}


// AddBackground function performs the operation of adding background to the db.
func AddBackground(ctx context.Context, storeDB *pgxpool.Pool, newDecor models.PersonalisedObject, userID uint) (uint, error) {

	var bID uint
	_, err = storeDB.Exec(ctx, "INSERT INTO backgrounds (link, small_image, category) VALUES ($1, $2, $3);",
		newDecor.Link,
		newDecor.SmallImage,
		newDecor.Type,
	)
	if err != nil {
		log.Printf("Error happened when inserting a new background entry into pgx table. Err: %s", err)
		return bID, err
	}
	err = storeDB.QueryRow(ctx, "SELECT backgrounds_id FROM backgrounds WHERE link=($1);", newDecor.Link).Scan(&bID)
	if err != nil {
		log.Printf("Error happened when retrieving usersid from the db. Err: %s", err)
		return bID, err
	}
	_, err = storeDB.Exec(ctx, "INSERT INTO users_has_backgrounds (users_id, backgrounds_id, is_favourite, is_personal) VALUES ($1, $2, $3, $4);",
		userID,
		bID,
		newDecor.IsFavourite,
		true,
	)
	if err != nil {
		log.Printf("Error happened when inserting a new entry into user_has_decoration pgx table. Err: %s", err)
		return bID, err
	}

	return bID, nil

}

// AdminDeleteBackground function performs the operation of deleting background from the db.
func AdminDeleteBackground(ctx context.Context, storeDB *pgxpool.Pool, bID uint) (error) {

	_, err = storeDB.Exec(ctx, "DELETE FROM backgrounds WHERE backgrounds_id=($1);",
		bID,
	)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("Error happened when deleting background from backgrounds pgx table. Err: %s", err)
		return err
	}

	_, err = storeDB.Exec(ctx, "DELETE FROM users_has_backgrounds WHERE backgrounds_id=($1);",
		bID,
	)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("Error happened when deleting background from users_has_backgrounds pgx table. Err: %s", err)
		return err
	}


	return nil

}


// DeleteBackground function performs the operation of deleting background from the db.
func DeleteBackground(ctx context.Context, storeDB *pgxpool.Pool, userID uint, bID uint) (error) {

	var isPersonal bool

	err = storeDB.QueryRow(ctx, "SELECT is_personal FROM users_has_backgrounds WHERE backgrounds_id=($1) AND users_id=($2);", bID, userID).Scan(&isPersonal)
	if err != nil || !isPersonal{
		log.Printf("The background does not belong to user. Err: %s", err)
		return err
	}
	
	_, err = storeDB.Exec(ctx, "DELETE FROM backgrounds WHERE backgrounds_id=($1);",
		bID,
	)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("Error happened when deleting background from backgrounds pgx table. Err: %s", err)
		return err
	}

	_, err = storeDB.Exec(ctx, "DELETE FROM users_has_backgrounds WHERE backgrounds_id=($1);",
		bID,
	)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("Error happened when deleting background from users_has_backgrounds pgx table. Err: %s", err)
		return err
	}


	return nil

}


// DeletePhoto function performs the operation of deleting photos by id from pgx database with a query.
func DeletePhoto(ctx context.Context, storeDB *pgxpool.Pool, photoID uint) (uint, error) {

	_, err = storeDB.Exec(ctx, "DELETE FROM photos WHERE photos_id=($1);",
		photoID,
	)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("Error happened when deleting photo from pgx table. Err: %s", err)
		return photoID, err
	}

	return photoID, nil

}

// RetrieveUserPhotos function performs the operation of retrieving user photos from pgx database with a query.
func RetrieveUserPhotos(ctx context.Context, storeDB *pgxpool.Pool, userID uint, sorting string, offset uint, limit uint) (models.ResponsePhotos, error) {

	var responsePhoto models.ResponsePhotos
	responsePhoto.Photos = []models.Photo{}
	rows, err := storeDB.Query(ctx, "SELECT photos_id, link, small_image, uploaded_at FROM photos WHERE users_id = ($1) ORDER BY photos_id DESC LIMIT ($2) OFFSET ($3);", userID, limit, offset)
	if err != nil && !errors.Is(err, pgx.ErrNoRows){
		log.Printf("Error happened when retrieving photos from pgx table. Err: %s", err)
		return responsePhoto, err
	}
	defer rows.Close()
	if sorting == "ASC" {
		rows, err := storeDB.Query(ctx, "SELECT photos_id, link, small_image, uploaded_at FROM photos WHERE users_id = ($1) ORDER BY photos_id ASC LIMIT ($2) OFFSET ($3);", userID, limit, offset)
		if err != nil && !errors.Is(err, pgx.ErrNoRows){
			log.Printf("Error happened when retrieving photos from pgx table. Err: %s", err)
			return responsePhoto, err
		}
		defer rows.Close()
	}

	for rows.Next() {
		var photo models.Photo
		var uploadTimeStorage time.Time
		if err = rows.Scan(&photo.PhotoID, &photo.Link, &photo.SmallImage, &uploadTimeStorage); err != nil {
			log.Printf("Error happened when scanning photos. Err: %s", err)
			return responsePhoto, err
		}
		photo.UploadedAt = uploadTimeStorage.Unix()
		responsePhoto.Photos = append(responsePhoto.Photos, photo)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error happened when retrieving photos from pgx table. Err: %s", err)
		return responsePhoto, err
	}

	var countAllString string
	err = storeDB.QueryRow(ctx, "SELECT COUNT(photos_id) FROM photos WHERE users_id = ($1);", userID).Scan(&countAllString)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			log.Printf("Error happened when counting backgrounds. Err: %s", err)
			return responsePhoto, err
		}
	responsePhoto.CountAll, _ = strconv.Atoi(countAllString)

	return responsePhoto, nil

}

// LoadBackgrounds function performs the operation of retrieving all backgrounds from the db for project editing.
func LoadBackgrounds(ctx context.Context, storeDB *pgxpool.Pool, userID uint, offset uint, limit uint, btype string, isfavourite bool, ispersonal bool) (models.ResponseBackground, error) {

	var responseBackground models.ResponseBackground
	responseBackground.Backgrounds = []models.Background{}
	if isfavourite != true && ispersonal != true {
		rows, err := storeDB.Query(ctx, "SELECT backgrounds_id, link, small_image, category FROM backgrounds WHERE category <> '' ORDER BY backgrounds_id DESC LIMIT ($1) OFFSET ($2);", limit, offset)
		if err != nil {
			log.Printf("Error happened when retrieving backgrounds from pgx table. Err: %s", err)
			return responseBackground, err
		}
		defer rows.Close()
		if btype != "" {
			rows, err = storeDB.Query(ctx, "SELECT backgrounds_id, link, small_image, category FROM backgrounds WHERE category = ($1) ORDER BY backgrounds_id DESC LIMIT ($2) OFFSET ($3);", btype, limit, offset)
			if err != nil {
				log.Printf("Error happened when retrieving backgrounds from pgx table. Err: %s", err)
				return responseBackground, err
			}
			defer rows.Close()
			}
		for rows.Next() {
			var background models.Background
			if err = rows.Scan(&background.BackgroundID, &background.Link, &background.SmallImage, &background.Type); err != nil {
				log.Printf("Error happened when scanning backgrounds. Err: %s", err)
				return responseBackground, err
			}
			err := storeDB.QueryRow(ctx, "SELECT is_personal, is_favourite FROM users_has_backgrounds WHERE backgrounds_id = ($1) AND users_id=($2);", background.BackgroundID, userID).Scan(&background.IsPersonal, &background.IsFavourite)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				log.Printf("Error happened when retrieving background from user_background table. Err: %s", err)
				return responseBackground, err
			}
			responseBackground.Backgrounds = append(responseBackground.Backgrounds, background)
		}

	} else if isfavourite == true {
		rows, err := storeDB.Query(ctx, "SELECT backgrounds_id, is_personal, is_favourite FROM users_has_backgrounds WHERE users_id = ($1) AND is_favourite = ($2) ORDER BY backgrounds_id DESC LIMIT ($3) OFFSET ($4);", userID, true, limit, offset)
				if err != nil {
					log.Printf("Error happened when retrieving user_decorations from pgx table. Err: %s", err)
					return responseBackground, err
				}
		defer rows.Close()
		for rows.Next() {
			var background models.Background
			var dbtype string
			if err = rows.Scan(&background.BackgroundID, &background.IsPersonal, &background.IsFavourite); err != nil {
				log.Printf("Error happened when scanning backgrounds. Err: %s", err)
				return responseBackground, err
			}
			err := storeDB.QueryRow(ctx, "SELECT link, small_image, category FROM backgrounds WHERE backgrounds_id = ($1);", background.BackgroundID).Scan(&background.Link, &background.SmallImage, &btype)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				log.Printf("Error happened when retrieving backgrounds from backgrounds table. Err: %s", err)
				return responseBackground, err
			}
			if dbtype != "" {
				background.Type = &dbtype
			}
			if btype != "" {
					if btype == dbtype {
						responseBackground.Backgrounds = append(responseBackground.Backgrounds, background)
					}
				
			} else {
				responseBackground.Backgrounds = append(responseBackground.Backgrounds, background)
			}
		}
	} else if ispersonal == true {
		rows, err := storeDB.Query(ctx, "SELECT backgrounds_id, is_personal, is_favourite FROM users_has_backgrounds WHERE users_id = ($1) AND is_personal = ($2) ORDER BY backgrounds_id DESC LIMIT ($3) OFFSET ($4);", userID, true, limit, offset)
				if err != nil {
					log.Printf("Error happened when retrieving user_decorations from pgx table. Err: %s", err)
					return responseBackground, err
				}
		defer rows.Close()
		for rows.Next() {
			var background models.Background
			if err = rows.Scan(&background.BackgroundID, &background.IsPersonal, &background.IsFavourite); err != nil {
				log.Printf("Error happened when scanning backgrounds. Err: %s", err)
				return responseBackground, err
			}
			err := storeDB.QueryRow(ctx, "SELECT link, small_image FROM backgrounds WHERE backgrounds_id = ($1);", background.BackgroundID).Scan(&background.Link, &background.SmallImage)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				log.Printf("Error happened when retrieving backgrounds from backgrounds table. Err: %s", err)
				return responseBackground, err
			}
			if btype != "" {
				if btype == *background.Type {
					responseBackground.Backgrounds = append(responseBackground.Backgrounds, background)
				}
			
			} else {
				responseBackground.Backgrounds = append(responseBackground.Backgrounds, background)
			}
		}
	}

	var countFavouriteString string
	err = storeDB.QueryRow(ctx, "SELECT COUNT(backgrounds_id) FROM users_has_backgrounds WHERE is_favourite = ($1) AND users_id=($2);", true, userID).Scan(&countFavouriteString)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				log.Printf("Error happened when counting backgrounds. Err: %s", err)
				return responseBackground, err
	}
	responseBackground.CountFavourite, _ = strconv.Atoi(countFavouriteString)
	if btype != "" {
		counter := 0
		rows, err := storeDB.Query(ctx, "SELECT backgrounds_id FROM users_has_backgrounds WHERE users_id = ($1) AND is_favourite = ($2) ORDER BY backgrounds_id;", userID, true)
				if err != nil {
					log.Printf("Error happened when retrieving users_has_backgrounds from pgx table. Err: %s", err)
					return responseBackground, err
				}
		defer rows.Close()
		for rows.Next() {
			var bId uint
			var bType string
			if err = rows.Scan(&bId); err != nil {
				log.Printf("Error happened when scanning backgrounds. Err: %s", err)
				return responseBackground, err
			}
			err := storeDB.QueryRow(ctx, "SELECT category FROM backgrounds WHERE backgrounds_id = ($1);", bId).Scan(&bType)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				log.Printf("Error happened when retrieving backgrounds from backgrounds table. Err: %s", err)
				return responseBackground, err
			}
			log.Println(btype)
			log.Println(bType)
			if btype==bType {
						counter = counter + 1
					}
			
		}
		responseBackground.CountFavourite = counter
	}
	var countPersonalString string
	err = storeDB.QueryRow(ctx, "SELECT COUNT(backgrounds_id) FROM users_has_backgrounds WHERE is_personal = ($1) AND users_id=($2);", true, userID).Scan(&countPersonalString)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				log.Printf("Error happened when counting backgrounds. Err: %s", err)
				return responseBackground, err
	}
	responseBackground.CountPersonal, _ = strconv.Atoi(countPersonalString)

	
	var countAllString string
	if isfavourite == true {
		responseBackground.CountAll = responseBackground.CountFavourite
	} else if ispersonal == true {
			responseBackground.CountAll = responseBackground.CountPersonal
	} else {
		err = storeDB.QueryRow(ctx, "SELECT COUNT(backgrounds_id) FROM backgrounds WHERE category <> '';").Scan(&countAllString)
		if err != nil {
					log.Printf("Error happened when counting backgrounds from pgx table. Err: %s", err)
					return responseBackground, err
		}
		if btype != "" {
				err = storeDB.QueryRow(ctx, "SELECT COUNT(backgrounds_id) FROM backgrounds WHERE category = ($1);", btype).Scan(&countAllString)
				if err != nil {
					log.Printf("Error happened when counting backgrounds from pgx table. Err: %s", err)
					return responseBackground, err
				}
			
		} 
		responseBackground.CountAll, _ = strconv.Atoi(countAllString)
	}

	return responseBackground, nil

}

// AddAdminBackground function performs the operation of adding background to the db.
func AddAdminBackground(ctx context.Context, storeDB *pgxpool.Pool, newB models.Background) (uint, error) {

	var bID uint
	_, err = storeDB.Exec(ctx, "INSERT INTO backgrounds (link, small_image, category) VALUES ($1, $2, $3);",
		newB.Link,
		newB.SmallImage,
		newB.Type,
	)
	if err != nil {
		log.Printf("Error happened when inserting a new admin background entry into pgx table. Err: %s", err)
		return bID, err
	}
	err = storeDB.QueryRow(ctx, "SELECT backgrounds_id FROM backgrounds WHERE link=($1);", newB.Link).Scan(&bID)
	if err != nil {
		log.Printf("Error happened when retrieving bid from the db. Err: %s", err)
		return bID, err
	}
	
	return bID, nil

}

// UpdateBackground function performs the operation of updating background to the db.
func UpdateBackground(ctx context.Context, storeDB *pgxpool.Pool, bID uint, newB models.Background) ( error) {

	_, err = storeDB.Exec(ctx, "UPDATE backgrounds SET link = ($1), small_image = ($2), category = ($3) WHERE backgrounds_id = ($4);",
		newB.Link,
		newB.SmallImage,
		newB.Type,
		bID,
	)
	if err != nil {
		log.Printf("Error happened when updating admin background entry into pgx table. Err: %s", err)
		return err
	}
	
	return nil

}

// UpdateDecoration function performs the operation of updating decoration to the db.
func UpdateDecoration(ctx context.Context, storeDB *pgxpool.Pool, dID uint, newD models.Decoration) ( error) {

	_, err = storeDB.Exec(ctx, "UPDATE decorations SET link = ($1), small_image = ($2), category = ($3), type = ($4) WHERE decorations_id = ($5);",
		newD.Link,
		newD.SmallImage,
		newD.Type,
		newD.Category,
		dID,
	)
	if err != nil {
		log.Printf("Error happened when updating admin decoration entry into pgx table. Err: %s", err)
		return err
	}
	
	return nil

}

// FavourBackground function performs the operation of updating background favourite bool in the db.
func FavourBackground(ctx context.Context, storeDB *pgxpool.Pool, newDecor models.PersonalisedObject, userID uint) (error) {

	var favourBool bool
	err = storeDB.QueryRow(ctx, "SELECT is_favourite FROM users_has_backgrounds WHERE backgrounds_id=($1) AND users_id=($2);", newDecor.ObjectID, userID).Scan(&favourBool)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			log.Printf("Error happened when searching for background in users_has_backgrounds pgx table. Err: %s", err)
			return err
		} else {
			_, err = storeDB.Exec(ctx, "INSERT INTO users_has_backgrounds (users_id, backgrounds_id, is_favourite, is_personal) VALUES ($1, $2, $3, $4);",
				userID,
				newDecor.ObjectID,
				newDecor.IsFavourite,
				false,
			)
			if err != nil {
				log.Printf("Error happened when inserting a new background entry into pgx table. Err: %s", err)
				return err
			}
			return nil
		}
	}
	_, err = storeDB.Exec(ctx, "UPDATE users_has_backgrounds SET is_favourite = ($1) WHERE backgrounds_id=($2) AND users_id=($3);",
		newDecor.IsFavourite,
		newDecor.ObjectID,
		userID,
	)
	if err != nil {
		log.Printf("Error happened when inserting a new entry into user_has_decoration pgx table. Err: %s", err)
		return err
	}

	return nil

}

// LoadDecorations function performs the operation of retrieving all decorations from the db for project editing.
func LoadDecorations(ctx context.Context, storeDB *pgxpool.Pool, userID uint, offset uint, limit uint, dtype string, dcategory string, isfavourite bool, ispersonal bool) (models.ResponseDecoration, error) {

	var responseDecoration models.ResponseDecoration
	responseDecoration.Decorations = []models.Decoration{}
	if isfavourite != true && ispersonal != true {
		rows, err := storeDB.Query(ctx, "SELECT * FROM (SELECT decorations_id, link, small_image, category, type FROM decorations WHERE category <> '') AS selectedD ORDER BY selectedD.decorations_id DESC LIMIT ($1) OFFSET ($2);", limit, offset)
				if err != nil {
					log.Printf("Error happened when retrieving decorations from pgx table. Err: %s", err)
					return responseDecoration, err
				}
		defer rows.Close()
		
		if dtype != "" {
			if dcategory != "" {
				rows, err = storeDB.Query(ctx, "SELECT decorations_id, link, small_image, category, type FROM decorations WHERE category = ($1) AND type = ($2) ORDER BY decorations_id DESC LIMIT ($3) OFFSET ($4);", dtype, dcategory, limit, offset)
				if err != nil {
					log.Printf("Error happened when retrieving decorations from pgx table. Err: %s", err)
					return responseDecoration, err
				}
				defer rows.Close()
			} else {
				rows, err = storeDB.Query(ctx, "SELECT decorations_id, link, small_image, category, type FROM decorations WHERE category = ($1) ORDER BY decorations_id DESC LIMIT ($2) OFFSET ($3);",  dtype, limit, offset)
				if err != nil {
					log.Printf("Error happened when retrieving decorations from pgx table. Err: %s", err)
					return responseDecoration, err
				}
				defer rows.Close()
			}
		} else {
			if dcategory != "" {
				rows, err = storeDB.Query(ctx, "SELECT decorations_id, link, small_image, category, type FROM decorations WHERE type = ($1) ORDER BY decorations_id DESC LIMIT ($2) OFFSET ($3);", dcategory, limit, offset)
				if err != nil {
					log.Printf("Error happened when retrieving decorations from pgx table. Err: %s", err)
					return responseDecoration, err
				}
				defer rows.Close()
			} 
		}

		for rows.Next() {
			var decoration models.Decoration
			if err = rows.Scan(&decoration.DecorationID, &decoration.Link, &decoration.SmallImage, &decoration.Type, &decoration.Category); err != nil {
				log.Printf("Error happened when scanning decorations. Err: %s", err)
				return responseDecoration, err
			}
			err := storeDB.QueryRow(ctx, "SELECT is_personal, is_favourite FROM users_has_decoration WHERE decorations_id = ($1) AND users_id=($2);", decoration.DecorationID, userID).Scan(&decoration.IsPersonal, &decoration.IsFavourite)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				log.Printf("Error happened when retrieving decorations from user_decorations table. Err: %s", err)
				return responseDecoration, err
			}
			responseDecoration.Decorations = append(responseDecoration.Decorations, decoration)
		}
	} else if isfavourite == true {
		rows, err := storeDB.Query(ctx, "SELECT decorations_id, is_personal, is_favourite FROM users_has_decoration WHERE users_id = ($1) AND is_favourite = ($2) ORDER BY decorations_id DESC LIMIT ($3) OFFSET ($4);", userID, true, limit, offset)
				if err != nil {
					log.Printf("Error happened when retrieving user_decorations from pgx table. Err: %s", err)
					return responseDecoration, err
				}
		defer rows.Close()
		for rows.Next() {
			var decoration models.Decoration
			if err = rows.Scan(&decoration.DecorationID, &decoration.IsPersonal, &decoration.IsFavourite); err != nil {
				log.Printf("Error happened when scanning decorations. Err: %s", err)
				return responseDecoration, err
			}
			if decoration.IsPersonal == true {
				err := storeDB.QueryRow(ctx, "SELECT link, small_image FROM decorations WHERE decorations_id = ($1);", decoration.DecorationID).Scan(&decoration.Link, &decoration.SmallImage)
				if err != nil && !errors.Is(err, pgx.ErrNoRows) {
					log.Printf("Error happened when retrieving decorations from decorations table. Err: %s", err)
					return responseDecoration, err
				}
			} else {
				err := storeDB.QueryRow(ctx, "SELECT link, small_image, category, type FROM decorations WHERE decorations_id = ($1);", decoration.DecorationID).Scan(&decoration.Link, &decoration.SmallImage, &decoration.Type, &decoration.Category)
				if err != nil && !errors.Is(err, pgx.ErrNoRows) {
					log.Printf("Error happened when retrieving decorations from decorations table. Err: %s", err)
					return responseDecoration, err
				}
			}
			if dtype != "" {
				if dcategory != "" {
					if decoration.IsPersonal == false {
						if dtype == *decoration.Type && dcategory == *decoration.Category {
							responseDecoration.Decorations = append(responseDecoration.Decorations, decoration)
						}
					}
					
				} else {
					if decoration.IsPersonal == false {
						if dtype == *decoration.Type {
							responseDecoration.Decorations = append(responseDecoration.Decorations, decoration)
						}
					}
				}
			} else {
				if dcategory != "" {
					if decoration.IsPersonal == false {
						if dcategory == *decoration.Category {
							responseDecoration.Decorations = append(responseDecoration.Decorations, decoration)
						}
					}
				} else {
					responseDecoration.Decorations = append(responseDecoration.Decorations, decoration)
				}
			}
			
		}
	} else if ispersonal == true {
		rows, err := storeDB.Query(ctx, "SELECT decorations_id,  is_personal, is_favourite FROM users_has_decoration WHERE users_id = ($1) AND is_personal = ($2) ORDER BY decorations_id DESC LIMIT ($3) OFFSET ($4);", userID, true, limit, offset)
				if err != nil {
					log.Printf("Error happened when retrieving user_decorations from pgx table. Err: %s", err)
					return responseDecoration, err
				}
		defer rows.Close()
		for rows.Next() {
			var decoration models.Decoration
			if err = rows.Scan(&decoration.DecorationID, &decoration.IsPersonal, &decoration.IsFavourite); err != nil {
				log.Printf("Error happened when scanning decorations. Err: %s", err)
				return responseDecoration, err
			}
			err := storeDB.QueryRow(ctx, "SELECT link, small_image FROM decorations WHERE decorations_id = ($1);", decoration.DecorationID).Scan(&decoration.Link, &decoration.SmallImage)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				log.Printf("Error happened when retrieving decorations from decorations table. Err: %s", err)
				return responseDecoration, err
			}
			if dtype != "" {
				if dcategory != "" {
					if decoration.IsPersonal == false {
						if dtype == *decoration.Type && dcategory == *decoration.Category {
							responseDecoration.Decorations = append(responseDecoration.Decorations, decoration)
						}
					}
					
				} else {
					if decoration.IsPersonal == false {
						if dtype == *decoration.Type {
							responseDecoration.Decorations = append(responseDecoration.Decorations, decoration)
						}
					}
				}
			} else {
				if dcategory != "" {
					if decoration.IsPersonal == false {
						if dcategory == *decoration.Category {
							responseDecoration.Decorations = append(responseDecoration.Decorations, decoration)
						}
					}
				} else {
					responseDecoration.Decorations = append(responseDecoration.Decorations, decoration)
				}
			}
		}
	}


	var countFavouriteString string
	err = storeDB.QueryRow(ctx, "SELECT COUNT(decorations_id) FROM users_has_decoration WHERE is_favourite = ($1) AND users_id=($2);", true, userID).Scan(&countFavouriteString)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				log.Printf("Error happened when counting decorations. Err: %s", err)
				return responseDecoration, err
	}
	responseDecoration.CountFavourite, _ = strconv.Atoi(countFavouriteString)
	if dtype != "" || dcategory != "" {
		counter := 0
		rows, err := storeDB.Query(ctx, "SELECT decorations_id FROM users_has_decoration WHERE users_id = ($1) AND is_favourite = ($2) ORDER BY decorations_id;", userID, true)
				if err != nil {
					log.Printf("Error happened when retrieving users_has_decoration from pgx table. Err: %s", err)
					return responseDecoration, err
				}
		defer rows.Close()
		for rows.Next() {
			var dId uint
			var dType string
			var dCategory string
			if err = rows.Scan(&dId); err != nil {
				log.Printf("Error happened when scanning decorations. Err: %s", err)
				return responseDecoration, err
			}
			err := storeDB.QueryRow(ctx, "SELECT category, type FROM decorations WHERE decorations_id = ($1);", dId).Scan(&dType, &dCategory)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				log.Printf("Error happened when retrieving decorations from decorations table. Err: %s", err)
				return responseDecoration, err
			}
			if dtype != "" {
				if dcategory != "" {
					if dtype==dType && dcategory == dCategory {
						counter = counter + 1
					}
				} else {
					if dtype==dType {
						counter = counter + 1
					}
				}
			} else if dcategory != "" {
				if dcategory == dCategory {
					counter = counter + 1
				}
			}
		}
		responseDecoration.CountFavourite = counter
	}
	var countPersonalString string
	err = storeDB.QueryRow(ctx, "SELECT COUNT(decorations_id) FROM users_has_decoration WHERE is_personal = ($1) AND users_id=($2);", true, userID).Scan(&countPersonalString)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				log.Printf("Error happened when counting decorations. Err: %s", err)
				return responseDecoration, err
	}
	responseDecoration.CountPersonal, _ = strconv.Atoi(countPersonalString)
	if dtype != "" || dcategory != "" {
		counter := 0
		rows, err := storeDB.Query(ctx, "SELECT decorations_id FROM users_has_decoration WHERE users_id = ($1) AND is_personal = ($2) ORDER BY decorations_id;", userID, true)
				if err != nil {
					log.Printf("Error happened when retrieving users_has_decoration from pgx table. Err: %s", err)
					return responseDecoration, err
				}
		defer rows.Close()
		for rows.Next() {
			var dId uint
			var dType string
			var dCategory string
			if err = rows.Scan(&dId); err != nil {
				log.Printf("Error happened when scanning decorations. Err: %s", err)
				return responseDecoration, err
			}
			err := storeDB.QueryRow(ctx, "SELECT category, type FROM decorations WHERE decorations_id = ($1);", dId).Scan(&dType, &dCategory)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				log.Printf("Error happened when retrieving decorations from decorations table. Err: %s", err)
				return responseDecoration, err
			}
			if dtype != "" {
				if dcategory != "" {
					if dtype==dType && dcategory == dCategory {
						counter = counter + 1
					}
				} else {
					if dtype==dType {
						counter = counter + 1
					}
				}
			} else if dcategory != "" {
				if dcategory == dCategory {
					counter = counter + 1
				}
			}
		}
		responseDecoration.CountPersonal = counter
	}

	var countAllString string
	if isfavourite == true {
		responseDecoration.CountAll = responseDecoration.CountFavourite
	} else if ispersonal == true {
			responseDecoration.CountAll = responseDecoration.CountPersonal
	} else {
		err = storeDB.QueryRow(ctx, "SELECT COUNT(decorations_id) FROM decorations WHERE category <> '';").Scan(&countAllString)
		if err != nil {
					log.Printf("Error happened when counting decorations from pgx table. Err: %s", err)
					return responseDecoration, err
		}
		if dtype != "" {
			if dcategory != "" {
				err = storeDB.QueryRow(ctx, "SELECT COUNT(decorations_id) FROM decorations WHERE category = ($1) AND type = ($2);", dtype, dcategory).Scan(&countAllString)
				if err != nil {
					log.Printf("Error happened when counting decorations from pgx table. Err: %s", err)
					return responseDecoration, err
				}
			} else {
				err = storeDB.QueryRow(ctx, "SELECT COUNT(decorations_id) FROM decorations WHERE category = ($1);", dtype).Scan(&countAllString)
				if err != nil {
					log.Printf("Error happened when counting decorations from pgx table. Err: %s", err)
					return responseDecoration, err
				}
			}
		} else {
			if dcategory != "" {
				err = storeDB.QueryRow(ctx, "SELECT COUNT(decorations_id) FROM decorations WHERE type = ($1);", dcategory).Scan(&countAllString)
				if err != nil {
					log.Printf("Error happened when counting decorations from pgx table. Err: %s", err)
					return responseDecoration, err
				}
			} 
		}
		responseDecoration.CountAll, _ = strconv.Atoi(countAllString)
	}
	
	return responseDecoration, nil

}

// AddAdminDecoration function performs the operation of adding decoration to the db.
func AddAdminDecoration(ctx context.Context, storeDB *pgxpool.Pool, newD models.Decoration) (uint, error) {

	var dID uint
	_, err = storeDB.Exec(ctx, "INSERT INTO decorations (link, small_image, category, type) VALUES ($1, $2, $3, $4);",
		newD.Link,
		newD.SmallImage,
		newD.Type,
		newD.Category,
	)
	if err != nil {
		log.Printf("Error happened when inserting a new admin decoration entry into pgx table. Err: %s", err)
		return dID, err
	}
	err = storeDB.QueryRow(ctx, "SELECT decorations_id FROM decorations WHERE link=($1);", newD.Link).Scan(&dID)
	if err != nil {
		log.Printf("Error happened when retrieving did from the db. Err: %s", err)
		return dID, err
	}
	
	return dID, nil

}

// FavourDecoration function performs the operation of updating decoration favourite bool in the db.
func FavourDecoration(ctx context.Context, storeDB *pgxpool.Pool, newDecor models.PersonalisedObject, userID uint) (error) {

	var favourBool bool
	err = storeDB.QueryRow(ctx, "SELECT is_favourite FROM users_has_decoration WHERE decorations_id=($1) AND users_id=($2);", newDecor.ObjectID, userID).Scan(&favourBool)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			log.Printf("Error happened when searching for decorations in users_has_decoration pgx table. Err: %s", err)
			return err
		} else {
			_, err = storeDB.Exec(ctx, "INSERT INTO users_has_decoration (users_id, decorations_id, is_favourite, is_personal) VALUES ($1, $2, $3, $4);",
				userID,
				newDecor.ObjectID,
				newDecor.IsFavourite,
				false,
			)
			if err != nil {
				log.Printf("Error happened when inserting a new decoration entry into pgx table. Err: %s", err)
				return err
			}
			return nil
		}
	}
	_, err = storeDB.Exec(ctx, "UPDATE users_has_decoration SET is_favourite = ($1) WHERE decorations_id=($2) AND users_id=($3);",
		newDecor.IsFavourite,
		newDecor.ObjectID,
		userID,
	)
	if err != nil {
		log.Printf("Error happened when inserting a new entry into user_has_decoration pgx table. Err: %s", err)
		return err
	}

	return nil

}

// LoadLayouts function performs the operation of retrieving all layouts from the db for project editing.
func LoadLayouts(ctx context.Context, storeDB *pgxpool.Pool, userID uint, offset uint, limit uint, size string, countimages uint, isfavourite bool) (models.ResponseLayout, error) {

	var responseLayout models.ResponseLayout
	responseLayout.Layouts = []models.Layout{}

	if isfavourite != true {
		rows, err := storeDB.Query(ctx, "SELECT * FROM (SELECT layouts_id, link, data, size, count_images FROM layouts) AS selectedL ORDER BY selectedL.layouts_id DESC LIMIT ($1) OFFSET ($2);", limit, offset)
				if err != nil {
					log.Printf("Error happened when retrieving layouts from pgx table. Err: %s", err)
					return responseLayout, err
				}
		defer rows.Close()
		
		if size != "" {
			if countimages != 0 {
				rows, err = storeDB.Query(ctx, "SELECT layouts_id, link, data, size, count_images FROM layouts WHERE size = ($1) AND count_images = ($2) ORDER BY layouts_id DESC LIMIT ($3) OFFSET ($4);", size, countimages, limit, offset)
				if err != nil {
					log.Printf("Error happened when retrieving layouts from pgx table. Err: %s", err)
					return responseLayout, err
				}
				defer rows.Close()
			} else {
				rows, err = storeDB.Query(ctx, "SELECT layouts_id, link, data, size, count_images FROM layouts WHERE size = ($1) ORDER BY layouts_id DESC LIMIT ($2) OFFSET ($3);",  size, limit, offset)
				if err != nil {
					log.Printf("Error happened when retrieving layouts from pgx table. Err: %s", err)
					return responseLayout, err
				}
				defer rows.Close()
			}
		} else {
			if countimages != 0 {
				rows, err = storeDB.Query(ctx, "SELECT layouts_id, link, data, size, count_images FROM layouts WHERE count_images = ($1) ORDER BY layouts_id DESC LIMIT ($2) OFFSET ($3);", countimages, limit, offset)
				if err != nil {
					log.Printf("Error happened when retrieving layouts from pgx table. Err: %s", err)
					return responseLayout, err
				}
				defer rows.Close()
			} 
		}

		for rows.Next() {
			var layout models.Layout
			var strdata sql.NullString
			var countImages uint

			if err = rows.Scan(&layout.LayoutID, &layout.Link, &strdata, &layout.Size, &countImages); err != nil {
				log.Printf("Error happened when scanning layouts. Err: %s", err)
				return responseLayout, err
			}

			layout.CountImages = countImages
			if strdata.Valid {
				layout.Data = json.RawMessage(strdata.String)
			} else {
				layout.Data = nil
			}
			err := storeDB.QueryRow(ctx, "SELECT is_favourite FROM users_has_layouts WHERE layouts_id = ($1) AND users_id=($2);", layout.LayoutID, userID).Scan(&layout.IsFavourite)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				log.Printf("Error happened when retrieving layouts from users_has_layouts table. Err: %s", err)
				return responseLayout, err
			}
			responseLayout.Layouts = append(responseLayout.Layouts, layout)
		}
	} else if isfavourite == true {
		rows, err := storeDB.Query(ctx, "SELECT layouts_id, is_favourite FROM users_has_layouts WHERE users_id = ($1) AND is_favourite = ($2) ORDER BY layouts_id DESC LIMIT ($3) OFFSET ($4);", userID, true, limit, offset)
				if err != nil {
					log.Printf("Error happened when retrieving users_has_layouts from pgx table. Err: %s", err)
					return responseLayout, err
				}
		defer rows.Close()
		for rows.Next() {
			var layout models.Layout
			var strdata sql.NullString
			var countImages uint
			if err = rows.Scan(&layout.LayoutID, &layout.IsFavourite); err != nil {
				log.Printf("Error happened when scanning layouts. Err: %s", err)
				return responseLayout, err
			}
			err := storeDB.QueryRow(ctx, "SELECT link, data, size, count_images FROM layouts WHERE layouts_id = ($1);", layout.LayoutID).Scan(&layout.Link, &strdata, &layout.Size, &countImages)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				log.Printf("Error happened when retrieving layouts from layouts table. Err: %s", err)
				return responseLayout, err
			}
			layout.CountImages = countImages
			if strdata.Valid {
				layout.Data = json.RawMessage(strdata.String)
			} else {
				layout.Data = nil
			}
			if size != "" {
				if countimages != 0 {
					if layout.Size == size && countImages == countimages {
						responseLayout.Layouts = append(responseLayout.Layouts, layout)
					}
				} else {
					if layout.Size == size {
						responseLayout.Layouts = append(responseLayout.Layouts, layout)
					}
				}
			} else if countimages != 0 {
				if countImages == countimages {
					responseLayout.Layouts = append(responseLayout.Layouts, layout)
				}
			} else {
				responseLayout.Layouts = append(responseLayout.Layouts, layout)
			}

		}
	} 

	var countFavouriteString string
	err = storeDB.QueryRow(ctx, "SELECT COUNT(layouts_id) FROM users_has_layouts WHERE is_favourite = ($1) AND users_id=($2);", true, userID).Scan(&countFavouriteString)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				log.Printf("Error happened when counting layouts. Err: %s", err)
				return responseLayout, err
	}
	responseLayout.CountFavourite, _ = strconv.Atoi(countFavouriteString)
	if size != "" || countimages != 0 {
		counter := 0
		rows, err := storeDB.Query(ctx, "SELECT layouts_id FROM users_has_layouts WHERE users_id = ($1) AND is_favourite = ($2) ORDER BY layouts_id;", userID, true)
				if err != nil {
					log.Printf("Error happened when retrieving users_has_layouts from pgx table. Err: %s", err)
					return responseLayout, err
				}
		defer rows.Close()
		for rows.Next() {
			var lId uint
			var cImages uint
			var lSize string
			if err = rows.Scan(&lId); err != nil {
				log.Printf("Error happened when scanning layouts. Err: %s", err)
				return responseLayout, err
			}
			err := storeDB.QueryRow(ctx, "SELECT size, count_images FROM layouts WHERE layouts_id = ($1);", lId).Scan(&lSize, &cImages)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				log.Printf("Error happened when retrieving layouts from layouts table. Err: %s", err)
				return responseLayout, err
			}
			if size != "" {
				if countimages != 0 {
					if size==lSize && countimages == cImages {
						counter = counter + 1
					}
				} else {
					if size==lSize {
						counter = counter + 1
					}
				}
			} else if countimages != 0{
				if countimages == cImages {
					counter = counter + 1
				}
			}
		}
		responseLayout.CountFavourite = counter
	}

	var countAllString string
	if isfavourite == true {
		responseLayout.CountAll = responseLayout.CountFavourite
	}  else {
		err = storeDB.QueryRow(ctx, "SELECT COUNT(layouts_id) FROM layouts;").Scan(&countAllString)
		if err != nil {
					log.Printf("Error happened when counting layouts from pgx table. Err: %s", err)
					return responseLayout, err
		}
		if size != "" {
			if countimages != 0 {
				err = storeDB.QueryRow(ctx, "SELECT COUNT(layouts_id) FROM layouts WHERE size = ($1) AND count_images = ($2);", size, countimages).Scan(&countAllString)
				if err != nil {
					log.Printf("Error happened when counting layouts from pgx table. Err: %s", err)
					return responseLayout, err
				}
			} else {
				err = storeDB.QueryRow(ctx, "SELECT COUNT(layouts_id) FROM layouts WHERE size = ($1);", size).Scan(&countAllString)
				if err != nil {
					log.Printf("Error happened when counting layouts from pgx table. Err: %s", err)
					return responseLayout, err
				}
			}
		} else {
			if countimages != 0 {
				err = storeDB.QueryRow(ctx, "SELECT COUNT(layouts_id) FROM layouts WHERE count_images = ($1);", countimages).Scan(&countAllString)
				if err != nil {
					log.Printf("Error happened when counting layouts from pgx table. Err: %s", err)
					return responseLayout, err
				}
			} 
		}
		responseLayout.CountAll, _ = strconv.Atoi(countAllString)
	}
	
	return responseLayout, nil

}

// AddAdminLayout function performs the operation of adding layout to the db.
func AddAdminLayout(ctx context.Context, storeDB *pgxpool.Pool, newL models.Layout) (uint, error) {

	var lID uint
	strdata := string(newL.Data)
	_, err = storeDB.Exec(ctx, "INSERT INTO layouts (link, count_images, data, size) VALUES ($1, $2, $3, $4);",
		newL.Link,
		newL.CountImages,
		strdata,
		newL.Size,
	)
	if err != nil {
		log.Printf("Error happened when inserting a new admin layout entry into pgx table. Err: %s", err)
		return lID, err
	}
	err = storeDB.QueryRow(ctx, "SELECT layouts_id FROM layouts WHERE link=($1);", newL.Link).Scan(&lID)
	if err != nil {
		log.Printf("Error happened when retrieving lid from the db. Err: %s", err)
		return lID, err
	}
	
	return lID, nil

}

// AdminDeleteLayout function performs the operation of deleting layout from the db.
func AdminDeleteLayout(ctx context.Context, storeDB *pgxpool.Pool, lID uint) (error) {

	_, err = storeDB.Exec(ctx, "DELETE FROM layouts WHERE layouts_id=($1);",
		lID,
	)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("Error happened when deleting layout from layouts pgx table. Err: %s", err)
		return err
	}

	_, err = storeDB.Exec(ctx, "DELETE FROM users_has_layouts WHERE layouts_id=($1);",
		lID,
	)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("Error happened when deleting layout from users_has_layouts pgx table. Err: %s", err)
		return err
	}


	return nil

}

// FavourLayout function performs the operation of updating layout favourite bool in the db.
func FavourLayout(ctx context.Context, storeDB *pgxpool.Pool, newDecor models.PersonalisedObject, userID uint) (error) {

	var favourBool bool
	err = storeDB.QueryRow(ctx, "SELECT is_favourite FROM users_has_layouts WHERE layouts_id=($1) AND users_id=($2);", newDecor.ObjectID, userID).Scan(&favourBool)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			log.Printf("Error happened when searching for layouts in users_has_layouts pgx table. Err: %s", err)
			return err
		} else {
			_, err = storeDB.Exec(ctx, "INSERT INTO users_has_layouts (users_id, layouts_id, is_favourite) VALUES ($1, $2, $3);",
				userID,
				newDecor.ObjectID,
				newDecor.IsFavourite,
			)
			if err != nil {
				log.Printf("Error happened when inserting a new layout entry into pgx table. Err: %s", err)
				return err
			}
			return nil
		}
	}
	_, err = storeDB.Exec(ctx, "UPDATE users_has_layouts SET is_favourite = ($1) WHERE layouts_id=($2) AND users_id=($3);",
		newDecor.IsFavourite,
		newDecor.ObjectID,
		userID,
	)
	if err != nil {
		log.Printf("Error happened when inserting a new entry into users_has_layouts pgx table. Err: %s", err)
		return err
	}

	return nil

}


// AddPrices function performs the operation of adding prices to the db.
func AddPrices(ctx context.Context, storeDB *pgxpool.Pool, newP []models.Price) (error) {

	for _, price := range newP {
        _, err = storeDB.Exec(ctx, "INSERT INTO prices (cover, variant, surface, size, baseprice, extrapage) VALUES ($1, $2, $3, $4, $5, $6);",
		price.Cover,
		price.Variant,
		price.Surface,
		price.Size,
		price.BasePrice,
		price.ExtraPage,
		)
		if err != nil {
			log.Printf("Error happened when inserting prices into pgx table. Err: %s", err)
			return err
		}
	}

	return nil

}

// DeletePrices function performs the operation of deleting prices from the db.
func DeletePrices(ctx context.Context, storeDB *pgxpool.Pool) (error) {

	_, err = storeDB.Exec(ctx, "DELETE * FROM prices;")
	if err != nil {
			log.Printf("Error happened when deleting prices from pgx table. Err: %s", err)
			return err
	}
	

	return nil

}

// RetrievePrices function performs the operation of retrieving prices from pgx database with a query.
func RetrievePrices(ctx context.Context, storeDB *pgxpool.Pool) ([]models.Price, error) {

	prices := []models.Price{}

	rows, err := storeDB.Query(ctx, "SELECT cover, variant, size, surface, baseprice, extrapage FROM prices;")
	if err != nil {
			log.Printf("Error happened when retrieving prices from pgx table. Err: %s", err)
			return prices, err
	}
	defer rows.Close()

	for rows.Next() {

			var priceObj models.Price
			if err = rows.Scan(&priceObj.Cover, &priceObj.Variant, &priceObj.Size, &priceObj.Surface, &priceObj.BasePrice, &priceObj.ExtraPage); err != nil {
				log.Printf("Error happened when scanning prices. Err: %s", err)
				return prices, err
			}

			prices = append(prices, priceObj)
			
	}

	return prices, nil

}


// AddCover function performs the operation of adding cover to the db.
func AddCover(ctx context.Context, storeDB *pgxpool.Pool, newC models.Colour) (error) {

	_, err = storeDB.Exec(ctx, "INSERT INTO leather (colourlink, hexcode) VALUES ($1, $2);",
		newC.LeatherImage,
		newC.HexCode,
	)
	if err != nil {
			log.Printf("Error happened when inserting cover into pgx table. Err: %s", err)
			return err
	}
	

	return nil

}

// AdminDeleteCover function performs the operation of deleting cover from the db.
func AdminDeleteCover(ctx context.Context, storeDB *pgxpool.Pool, cID uint) (error) {

	_, err = storeDB.Exec(ctx, "DELETE FROM leather WHERE leather_id=($1);",
		cID,
	)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("Error happened when deleting cover from leather pgx table. Err: %s", err)
		return err
	}

	return nil

}

// RetrieveCovers function performs the operation of retrieving covers from pgx database with a query.
func RetrieveCovers(ctx context.Context, storeDB *pgxpool.Pool) ([]models.Colour, error) {

	covers := []models.Colour{}

	rows, err := storeDB.Query(ctx, "SELECT leather_id, colourlink, hexcode FROM leather;")
	if err != nil {
			log.Printf("Error happened when retrieving covers from pgx table. Err: %s", err)
			return covers, err
	}
	defer rows.Close()

	for rows.Next() {

			var coverObj models.Colour
			if err = rows.Scan(&coverObj.ID, &coverObj.LeatherImage, &coverObj.HexCode); err != nil {
				log.Printf("Error happened when scanning covers. Err: %s", err)
				return covers, err
			}

			covers = append(covers, coverObj)
			
	}

	return covers, nil

}


