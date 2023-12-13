// Storage package contains functions for storing photos and projects in a pgx database.
//
// Available at https://github.com/SiberianMonster/memoryprint/tree/development/internal/objectsstorage
package objectsstorage

import (
	"context"
	"errors"
	"log"
	"time"
	"github.com/SiberianMonster/memoryprint/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var err error

func CheckUserOwnsPhoto(ctx context.Context, storeDB *pgxpool.Pool, userID uint, photoID uint) bool {

	var checkPhoto bool
	err := storeDB.QueryRow(ctx, "SELECT CASE WHEN EXISTS (SELECT * FROM photos WHERE photos_id = ($1) AND users_id = ($2)) THEN 'TRUE' ELSE 'FALSE' END;", photoID, userID).Scan(&checkPhoto)
	if err != nil && err != pgx.ErrNoRows {
		log.Printf("Error happened when checking if user can edit photo in db. Err: %s", err)
		return false
	}

	return checkPhoto
}


// AddPhoto function performs the operation of adding photos to the db.
func AddPhoto(ctx context.Context, storeDB *pgxpool.Pool, photoLink string, userID uint) (uint, error) {

	var photoID uint
	t := time.Now()
	_, err = storeDB.Exec(ctx, "INSERT INTO photos (link, uploaded_at, users_id) VALUES ($1, $2, $3);",
		photoLink,
		t,
		userID,
	)
	if err != nil {
		log.Printf("Error happened when inserting a new photo entry into pgx table. Err: %s", err)
		return userID, err
	}
	err = storeDB.QueryRow(ctx, "SELECT photos_id FROM photos WHERE link=($1);", photoLink).Scan(&photoID)
	if err != nil {
		log.Printf("Error happened when retrieving usersid from the db. Err: %s", err)
		return photoID, err
	}

	return photoID, nil

}


// RetrievePhoto function performs the operation of retrieving photos by id from pgx database with a query.
func RetrievePhoto(ctx context.Context, storeDB *pgxpool.Pool, photoID uint) (string, error) {

	var photo string
	err := storeDB.QueryRow(ctx, "SELECT link FROM photos WHERE photos_id = ($1);", photoID).Scan(&photo)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("Error happened when retrieving photo from pgx table. Err: %s", err)
		return "", err
	}

	return photo, nil

}

// AddDecoration function performs the operation of adding decoration to the db.
func AddDecoration(ctx context.Context, storeDB *pgxpool.Pool, newDecor models.PersonalisedObject, userID uint) (uint, error) {

	var decorID uint
	_, err = storeDB.Exec(ctx, "INSERT INTO decorations (link, category, type) VALUES ($1, $2, $3);",
		newDecor.Link,
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
		newDecor.IsPersonal,
	)
	if err != nil {
		log.Printf("Error happened when inserting a new entry into user_has_decoration pgx table. Err: %s", err)
		return decorID, err
	}

	return decorID, nil

}

// DeleteDecoration function performs the operation of deleting decoration from the db.
func DeleteDecoration(ctx context.Context, storeDB *pgxpool.Pool, decorID uint) (error) {

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

// RetrieveDecoration function performs the operation of retrieving decoration by id from pgx database with a query.
func RetrieveDecoration(ctx context.Context, storeDB *pgxpool.Pool, objID uint) (string, error) {

	var link string
	err := storeDB.QueryRow(ctx, "SELECT link FROM decorations WHERE decorations_id = ($1);", objID).Scan(&link)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("Error happened when retrieving decoration from pgx table. Err: %s", err)
		return "", err
	}

	return link, nil

}

// RetrieveLayout function performs the operation of retrieving layout by id from pgx database with a query.
func RetrieveLayout(ctx context.Context, storeDB *pgxpool.Pool, objID uint) (string, error) {

	var link string
	err := storeDB.QueryRow(ctx, "SELECT link FROM layouts WHERE layouts_id = ($1);", objID).Scan(&link)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("Error happened when retrieving decoration from pgx table. Err: %s", err)
		return "", err
	}

	return link, nil

}

// AddBackground function performs the operation of adding background to the db.
func AddBackground(ctx context.Context, storeDB *pgxpool.Pool, newDecor models.PersonalisedObject, userID uint) (uint, error) {

	var bID uint
	_, err = storeDB.Exec(ctx, "INSERT INTO backgrounds (link, category) VALUES ($1, $2);",
		newDecor.Link,
		newDecor.Category,
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
		newDecor.IsPersonal,
	)
	if err != nil {
		log.Printf("Error happened when inserting a new entry into user_has_decoration pgx table. Err: %s", err)
		return bID, err
	}

	return bID, nil

}

// DeleteBackground function performs the operation of deleting background from the db.
func DeleteBackground(ctx context.Context, storeDB *pgxpool.Pool, bID uint) (error) {

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

// RetrieveBackground function performs the operation of retrieving background by id from pgx database with a query.
func RetrieveBackground(ctx context.Context, storeDB *pgxpool.Pool, objID uint) (string, error) {

	var link string
	err := storeDB.QueryRow(ctx, "SELECT link FROM backgrounds WHERE backgrounds_id = ($1);", objID).Scan(&link)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("Error happened when retrieving background from pgx table. Err: %s", err)
		return "", err
	}

	return link, nil

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
func RetrieveUserPhotos(ctx context.Context, storeDB *pgxpool.Pool, userID uint) ([]string, error) {

	var photoslice []string
	rows, err := storeDB.Query(ctx, "SELECT link FROM photos WHERE users_id = ($1);", userID)
	if err != nil {
		log.Printf("Error happened when retrieving photos from pgx table. Err: %s", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var photo string
		if err = rows.Scan(&photo); err != nil {
			log.Printf("Error happened when scanning photos. Err: %s", err)
			return nil, err
		}
		photoslice = append(photoslice, photo)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error happened when retrieving photos from pgx table. Err: %s", err)
		return nil, err
	}
	return photoslice, nil

}



// RetrieveAllBackgrounds function performs the operation of retrieving all backgrounds from the db for project editing.
func RetrieveAllBackgrounds(ctx context.Context, storeDB *pgxpool.Pool) ([]models.Background, error) {

	var backgroundslice []models.Background
	rows, err := storeDB.Query(ctx, "SELECT backgrounds_id, link, category FROM backgrounds;")
	if err != nil {
		log.Printf("Error happened when retrieving orders from pgx table. Err: %s", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var background models.Background
		if err = rows.Scan(&background.BackgroundID, &background.Link, &background.Category); err != nil {
			log.Printf("Error happened when scanning backgrounds. Err: %s", err)
			return nil, err
		}
		
		backgroundslice = append(backgroundslice, background)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error happened when retrieving backgrounds from pgx table. Err: %s", err)
		return nil, err
	}

	return backgroundslice, nil

}

// RetrieveAllLayouts function performs the operation of retrieving all layouts from the db for project editing.
func RetrieveAllLayouts(ctx context.Context, storeDB *pgxpool.Pool) ([]models.Layout, error) {

	var layoutslice []models.Layout
	rows, err := storeDB.Query(ctx, "SELECT layouts_id, link, category FROM layouts;")
	if err != nil {
		log.Printf("Error happened when retrieving layouts from pgx table. Err: %s", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var layout models.Layout
		if err = rows.Scan(&layout.LayoutID, &layout.Link, &layout.Category); err != nil {
			log.Printf("Error happened when scanning layouts. Err: %s", err)
			return nil, err
		}
		
		layoutslice = append(layoutslice, layout)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error happened when retrieving layouts from pgx table. Err: %s", err)
		return nil, err
	}

	return layoutslice, nil

}

// RetrieveAllDecorations function performs the operation of retrieving all decorations from the db for project editing.
func RetrieveAllDecorations(ctx context.Context, storeDB *pgxpool.Pool) ([]models.Decoration, error) {

	var decorationslice []models.Decoration
	rows, err := storeDB.Query(ctx, "SELECT decorations_id, link, type, category FROM decorations;")
	if err != nil {
		log.Printf("Error happened when retrieving decorations from pgx table. Err: %s", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var decoration models.Decoration
		if err = rows.Scan(&decoration.DecorationID, &decoration.Link, &decoration.Type, &decoration.Category); err != nil {
			log.Printf("Error happened when scanning decorations. Err: %s", err)
			return nil, err
		}
		
		decorationslice = append(decorationslice, decoration)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error happened when retrieving decorations from pgx table. Err: %s", err)
		return nil, err
	}

	return decorationslice, nil

}


// LoadProjectSession function performs the operation of retrieving all support art objects from the db for project editing.
func LoadProjectSession(ctx context.Context, storeDB *pgxpool.Pool) (models.ProjectSession, error) {
	var pSession models.ProjectSession
	var err error
	pSession.Decorations, err = RetrieveAllDecorations(ctx, storeDB)
	if err != nil {
		log.Printf("Error happened when retrieving decorations. Err: %s", err)
		return pSession, err
	}
	pSession.Background, err = RetrieveAllBackgrounds(ctx, storeDB)
	if err != nil {
		log.Printf("Error happened when retrieving backgrounds. Err: %s", err)
		return pSession, err
	}
	pSession.Layout, err = RetrieveAllLayouts(ctx, storeDB)
	if err != nil {
		log.Printf("Error happened when retrieving layouts. Err: %s", err)
		return pSession, err
	}
	return pSession, nil

}

// RetrieveUserPersonalisedObjects function performs the operation of retrieving user personalised backgrounds and decorations from pgx database with a query.
func RetrieveUserPersonalisedObjects(ctx context.Context, storeDB *pgxpool.Pool, userID uint) (models.RetrievedPersonalisedObj, error) {

	var persObj models.RetrievedPersonalisedObj
	rows, err := storeDB.Query(ctx, "SELECT backgrounds_id, is_favourite, is_personal FROM user_has_background WHERE users_id = ($1);", userID)
	if err != nil {
		log.Printf("Error happened when retrieving user backgrounds from pgx table. Err: %s", err)
		return persObj, err
	}
	defer rows.Close()

	for rows.Next() {
		var bObj models.PersonalisedObject
		if err = rows.Scan(&bObj.ObjectID, bObj.IsFavourite, bObj.IsPersonal); err != nil {
			log.Printf("Error happened when scanning user backgrounds. Err: %s", err)
			return persObj, err
		}
		bObj.Link, err = RetrieveBackground(ctx, storeDB, bObj.ObjectID)
		if err != nil {
			log.Printf("Error happened when retrieving background link. Err: %s", err)
			return persObj, err
		}
		persObj.Backgrounds = append(persObj.Backgrounds, bObj)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error happened when retrieving user backgrounds from pgx table. Err: %s", err)
		return persObj, err
	}

	rows, err = storeDB.Query(ctx, "SELECT decoration_id, is_favourite, is_personal FROM user_has_decoration WHERE users_id = ($1);", userID)
	if err != nil {
		log.Printf("Error happened when retrieving user decorations from pgx table. Err: %s", err)
		return persObj, err
	}
	defer rows.Close()

	for rows.Next() {
		var dObj models.PersonalisedObject
		if err = rows.Scan(&dObj.ObjectID, dObj.IsFavourite, dObj.IsPersonal); err != nil {
			log.Printf("Error happened when scanning user decorations. Err: %s", err)
			return persObj, err
		}
		dObj.Link, err = RetrieveDecoration(ctx, storeDB, dObj.ObjectID)
		if err != nil {
			log.Printf("Error happened when retrieving decorations link. Err: %s", err)
			return persObj, err
		}
		persObj.Decor = append(persObj.Decor, dObj)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error happened when retrieving user decorations from pgx table. Err: %s", err)
		return persObj, err
	}
	return persObj, nil

}