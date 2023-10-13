// Storage package contains functions for storing photos and projects in a pgx database.
//
// Available at https://github.com/SiberianMonster/memoryprint/tree/development/internal/projectstorage
package projectstorage

import (
	"context"
	"errors"
	"github.com/SiberianMonster/memoryprint/internal/objectsstorage"
	"github.com/SiberianMonster/memoryprint/internal/models"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var err error
type ArtObjects []models.ArtObject
type Photos []models.Photo

func makeRange(min, max int) []int {
    a := make([]int, max-min+1)
    for i := range a {
        a[i] = min + i
    }
    return a
}

func CheckUserHasProject(ctx context.Context, storeDB *pgxpool.Pool, userID uint, projectID uint) bool {

	var checkProject bool
	var email string
	err := storeDB.QueryRow(ctx, "SELECT email FROM users WHERE users_id = ($1);", userID).Scan(&email)
	if err != nil {
			log.Printf("Error happened when retrieving user email data from db. Err: %s", err)
			return false
	}
	err = storeDB.QueryRow(ctx, "SELECT CASE WHEN EXISTS (SELECT * FROM users_edit_projects WHERE project_id = ($1) AND email = ($2)) THEN TRUE ELSE FALSE END;", projectID, email).Scan(&checkProject)
	if err != nil && err != pgx.ErrNoRows {
		log.Printf("Error happened when checking if user can edit project in db. Err: %s", err)
		return false
	}

	return checkProject
}

// CreateProject function performs the operation of creating a new photobook project in pgx database with a query.
func CreateProject(ctx context.Context, storeDB *pgxpool.Pool, userID uint, pageNumber int, projectname string) (uint, error) {

	t := time.Now()
	var pID uint
	var email string
	err := storeDB.QueryRow(ctx, "INSERT INTO projects (name, created_at, last_edited_at, status, last_editor, users_id) VALUES ($1, $2, $3, $4, $5, $6) RETURNING project_id;",
		projectname,
		t,
		t,
		"EDITED",
		userID,
		userID,
	).Scan(&pID)
	if err != nil {
		log.Printf("Error happened when inserting a new project into pgx table. Err: %s", err)
		return 0, err
	}

	pagesRange := makeRange(1, pageNumber)

	for _, num := range pagesRange {
		_, err = storeDB.Exec(ctx, "INSERT INTO pages (last_edited_at, number, project_id) VALUES ($1, $2, $3);",
			t,
			num,
			pID,
		)
		if err != nil {
			log.Printf("Error happened when inserting a new page into pgx table. Err: %s", err)
			return pID, err
		}
    }

	err = storeDB.QueryRow(ctx, "SELECT email FROM users WHERE users_id = ($1);", userID).Scan(&email)
	if err != nil {
			log.Printf("Error happened when retrieving user email data from db. Err: %s", err)
			return pID, err
	}

	_, err = AddProjectEditor(ctx, storeDB, email, pID, "OWNER")
	if err != nil {
		log.Printf("Error happened when adding project editor. Err: %s", err)
		return pID, err
	}

	log.Printf("added new project to DB")
	return pID, nil
}

// RetrieveProjectPages function performs the operation of retrieving a photobook project from pgx database with a query.
func RetrieveProjectPages(ctx context.Context, storeDB *pgxpool.Pool, projectID uint) ([]uint, error) {

	var pageslice []uint
	rows, err := storeDB.Query(ctx, "SELECT page_id FROM pages WHERE project_id = ($1) ORDER BY number;", projectID)
	if err != nil {
		log.Printf("Error happened when retrieving pages from pgx table. Err: %s", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var page uint
		if err = rows.Scan(&page); err != nil {
			log.Printf("Error happened when scanning pages. Err: %s", err)
			return nil, err
		}
		pageslice = append(pageslice, page)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error happened when retrieving pages from pgx table. Err: %s", err)
		return nil, err
	}
	return pageslice, nil

}

// AddProjectPage function performs the operation of adding a photobook project page to pgx database with a query.
func AddProjectPage(ctx context.Context, storeDB *pgxpool.Pool, projectID uint) (error) {

	var pageslice []models.Page
	rows, err := storeDB.Query(ctx, "SELECT page_id, background FROM pages WHERE project_id = ($1);", projectID)
	if err != nil {
		log.Printf("Error happened when retrieving pages from pgx table. Err: %s", err)
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var page models.Page
		if err = rows.Scan(&page.PageID, &page.Background); err != nil {
			log.Printf("Error happened when scanning pages. Err: %s", err)
			return err
		}
		pageslice = append(pageslice, page)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error happened when retrieving pages from pgx table. Err: %s", err)
		return err
	}

	t := time.Now()
	newPageNum := len(pageslice) + 1
	_, err = storeDB.Exec(ctx, "INSERT INTO pages (created_at, number, background, project_id) VALUES ($1, $2, $3, $4);",
			t,
			newPageNum,
			"",
			projectID,
		)
		if err != nil {
			log.Printf("Error happened when inserting a new page into pgx table. Err: %s", err)
			return err
		}
	return nil

}

// RetrieveProjectPhotos function performs the operation of retrieving photos related to existing project from pgx database with a query.
func RetrieveProjectPhotos(ctx context.Context, storeDB *pgxpool.Pool, projectID uint) ([]models.Photo, error) {

	var photoslice []models.Photo
	rows, err := storeDB.Query(ctx, "SELECT photo_id FROM project_has_photos WHERE project_id = ($1);", projectID)
	if err != nil {
		log.Printf("Error happened when retrieving project photos from pgx table. Err: %s", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var photoID uint
		if err = rows.Scan(&photoID); err != nil {
			return nil, err
		}

		photo, err := objectsstorage.RetrievePhoto(ctx, storeDB, photoID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			log.Printf("Error happened when retrieving photo from pgx table. Err: %s", err)
		}
		photoslice = append(photoslice, models.Photo{PhotoID: photoID, Link: photo})
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error happened when retrieving project photos from pgx table. Err: %s", err)
		return nil, err
	}
	return photoslice, nil

}

// SavePagePhotos function performs the operation of saving edited photos related to existing project.
func SavePagePhotos(ctx context.Context, storeDB *pgxpool.Pool, pageID uint, images ArtObjects) error {

	var err error

	_, err = storeDB.Exec(ctx, "DELETE FROM page_has_photos WHERE page_id=($1);",
		pageID,
	)
	if err != nil && err != pgx.ErrNoRows {
		log.Printf("Error happened when deleting old page photos from pgx table. Err: %s", err)
		return err
	}

	t := time.Now()
	tx, err := storeDB.Begin(ctx)
	if err != nil {
		log.Printf("Error happened when initiating pgx transaction. Err: %s", err)
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Prepare(ctx, "my-query","INSERT INTO page_has_photos (page_id, photo_id, ptop, pleft, style, last_edited_at) VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT DO UPDATE SET (page_id, photo_id, ptop, pleft, style, last_edited_at) = ($1, $2, $3, $4, $5, $6);")
	if err != nil {
		log.Printf("Error happened when preparing pgx transaction context. Err: %s", err)
		return err
	}

	for _, v := range images {
		if _, err = tx.Exec(ctx, pageID, v.ObjectID, v.Ptop, v.Pleft, v.Style, t); err != nil {
			log.Printf("Error happened when declaring transaction. Err: %s", err)
			return err
		}
	}

	return tx.Commit(ctx)

}

// SavePageDecorations function performs the operation of saving edited decorations related to existing project.
func SavePageDecorations(ctx context.Context, storeDB *pgxpool.Pool, pageID uint, images ArtObjects) error {

	var err error

	_, err = storeDB.Exec(ctx, "DELETE FROM page_has_decorations WHERE page_id=($1);",
		pageID,
	)
	if err != nil && err != pgx.ErrNoRows {
		log.Printf("Error happened when deleting old page decorations from pgx table. Err: %s", err)
		return err
	}

	t := time.Now()
	tx, err := storeDB.Begin(ctx)
	if err != nil {
		log.Printf("Error happened when initiating pgx transaction. Err: %s", err)
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Prepare(ctx, "my-query", "INSERT INTO page_has_decorations (page_id, decorations_id, ptop, pleft, style, last_edited_at) VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT DO UPDATE SET (page_id, decorations_id, ptop, pleft, style, last_edited_at) = ($1, $2, $3, $4, $5, $6);")
	if err != nil {
		log.Printf("Error happened when preparing pgx transaction context. Err: %s", err)
		return err
	}

	for _, v := range images {
		if _, err = tx.Exec(ctx, pageID, v.ObjectID, v.Ptop, v.Pleft, v.Style, t); err != nil {
			log.Printf("Error happened when declaring transaction. Err: %s", err)
			return err
		}
	}

	return tx.Commit(ctx)

}

// SavePageLayout function performs the operation of saving layout related to existing project.
func SavePageLayout(ctx context.Context, storeDB *pgxpool.Pool, pageID uint, images ArtObjects) error {

	var err error

	_, err = storeDB.Exec(ctx, "DELETE FROM page_has_layout WHERE page_id=($1);",
		pageID,
	)
	if err != nil && err != pgx.ErrNoRows {
		log.Printf("Error happened when deleting old page layout from pgx table. Err: %s", err)
		return err
	}

	t := time.Now()
	tx, err := storeDB.Begin(ctx)
	if err != nil {
		log.Printf("Error happened when initiating pgx transaction. Err: %s", err)
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Prepare(ctx, "my-query", "INSERT INTO page_has_layout (page_id, layout_id, last_edited_at) VALUES ($1, $2, $3) ON CONFLICT DO UPDATE SET (page_id, layout_id, last_edited_at) = ($1, $2, $3);")
	if err != nil {
		log.Printf("Error happened when preparing pgx transaction context. Err: %s", err)
		return err
	}

	for _, v := range images {
		if _, err = tx.Exec(ctx, pageID, v.ObjectID, t); err != nil {
			log.Printf("Error happened when declaring transaction. Err: %s", err)
			return err
		}
	}

	return tx.Commit(ctx)

}

// SavePageBackground function performs the operation of saving background related to existing project.
func SavePageBackground(ctx context.Context, storeDB *pgxpool.Pool, pageID uint, images ArtObjects) error {

	var err error

	_, err = storeDB.Exec(ctx, "DELETE FROM page_has_background WHERE page_id=($1);",
		pageID,
	)
	if err != nil && err != pgx.ErrNoRows {
		log.Printf("Error happened when deleting old page background from pgx table. Err: %s", err)
		return err
	}

	t := time.Now()
	tx, err := storeDB.Begin(ctx)
	if err != nil {
		log.Printf("Error happened when initiating pgx transaction. Err: %s", err)
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Prepare(ctx, "my-query", "INSERT INTO page_has_background (page_id, background_id, last_edited_at) VALUES ($1, $2, $3) ON CONFLICT DO UPDATE SET (page_id, background_id, last_edited_at) = ($1, $2, $3);")
	if err != nil {
		log.Printf("Error happened when preparing pgx transaction context. Err: %s", err)
		return err
	}

	for _, v := range images {
		if _, err = tx.Exec(ctx, pageID, v.ObjectID, t); err != nil {
			log.Printf("Error happened when declaring transaction. Err: %s", err)
			return err
		}
	}

	return tx.Commit(ctx)

}

// SavePageText function performs the operation of saving custom text related to existing project.
func SavePageText(ctx context.Context, storeDB *pgxpool.Pool, pageID uint, textObjs []models.TextObject) error {

	var err error

	_, err = storeDB.Exec(ctx, "DELETE FROM page_has_text WHERE page_id=($1);",
		pageID,
	)
	if err != nil && err != pgx.ErrNoRows {
		log.Printf("Error happened when deleting old page text from pgx table. Err: %s", err)
		return err
	}

	t := time.Now()
	tx, err := storeDB.Begin(ctx)
	if err != nil {
		log.Printf("Error happened when initiating pgx transaction. Err: %s", err)
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Prepare(ctx, "my-query", "INSERT INTO page_has_text (page_id, custom_text, ptop, pleft, style, last_edited_at) VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT DO UPDATE SET (page_id, custom_text, ptop, pleft, style, last_edited_at) = ($1, $2, $3, $4, $5, $6);")
	if err != nil {
		log.Printf("Error happened when preparing pgx transaction context. Err: %s", err)
		return err
	}

	for _, v := range textObjs {
		if _, err = tx.Exec(ctx, pageID, v.CustomText, v.Ptop, v.Pleft, v.Style, t); err != nil {
			log.Printf("Error happened when declaring transaction. Err: %s", err)
			return err
		}
	}

	return tx.Commit(ctx)

}

// RetrievePagePhotos function performs the operation of retrieving photos related to existing project from pgx database with a query.
func RetrievePagePhotos(ctx context.Context, storeDB *pgxpool.Pool, pageID uint) ([]models.ArtObject, error) {

	var images = []models.ArtObject{}
	rows, err := storeDB.Query(ctx, "SELECT photo_id, ptop, pleft, style FROM page_has_photos WHERE page_id = ($1);", pageID)
	if err != nil {
		log.Printf("Error happened when retrieving project photos from pgx table. Err: %s", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var photoID uint
		var image models.ArtObject
		if err = rows.Scan(&photoID, &image.Ptop, &image.Pleft, &image.Style); err != nil {
			return nil, err
		}

		photo, err := objectsstorage.RetrievePhoto(ctx, storeDB, photoID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			log.Printf("Error happened when retrieving photo from pgx table. Err: %s", err)
			return nil, err
		}
		image.Name = photo
		image.ObjectID = photoID
		images = append(images, image)

	}
	
	return images, nil
}

// RetrievePageDecorations function performs the operation of retrieving decorations related to existing project from pgx database with a query.
func RetrievePageDecorations(ctx context.Context, storeDB *pgxpool.Pool, pageID uint) ([]models.ArtObject, error) {

	var images = []models.ArtObject{}
	rows, err := storeDB.Query(ctx, "SELECT decorations_id, ptop, pleft, style FROM page_has_decorations WHERE page_id = ($1);", pageID)
	if err != nil {
		log.Printf("Error happened when retrieving project decorations from pgx table. Err: %s", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var objID uint
		var image models.ArtObject
		if err = rows.Scan(&objID, &image.Ptop, &image.Pleft, &image.Style); err != nil {
			return nil, err
		}

		obj, err := objectsstorage.RetrieveDecoration(ctx, storeDB, objID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			log.Printf("Error happened when retrieving decoration from pgx table. Err: %s", err)
			return nil, err
		}
		image.Name = obj
		image.ObjectID = objID
		images = append(images, image)

	}
	return images, nil
}

// RetrievePageLayout function performs the operation of retrieving layout related to existing project from pgx database with a query.
func RetrievePageLayout(ctx context.Context, storeDB *pgxpool.Pool, pageID uint) ([]models.ArtObject, error) {

	var images = []models.ArtObject{}
	rows, err := storeDB.Query(ctx, "SELECT layout_id, FROM page_has_layouts WHERE page_id = ($1);", pageID)
	if err != nil {
		log.Printf("Error happened when retrieving project layout from pgx table. Err: %s", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var objID uint
		var image models.ArtObject
		if err = rows.Scan(&objID); err != nil {
			return nil, err
		}

		obj, err := objectsstorage.RetrieveLayout(ctx, storeDB, objID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			log.Printf("Error happened when retrieving layout from pgx table. Err: %s", err)
			return nil, err
		}
		image.Name = obj
		image.ObjectID = objID
		images = append(images, image)

	}
	return images, nil
}

// RetrievePageBackground function performs the operation of retrieving background related to existing project from pgx database with a query.
func RetrievePageBackground(ctx context.Context, storeDB *pgxpool.Pool, pageID uint) ([]models.ArtObject, error) {

	var images = []models.ArtObject{}
	rows, err := storeDB.Query(ctx, "SELECT background_id, FROM page_has_backgrounds WHERE page_id = ($1);", pageID)
	if err != nil {
		log.Printf("Error happened when retrieving project background from pgx table. Err: %s", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var objID uint
		var image models.ArtObject
		if err = rows.Scan(&objID); err != nil {
			return nil, err
		}

		obj, err := objectsstorage.RetrieveBackground(ctx, storeDB, objID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			log.Printf("Error happened when retrieving background from pgx table. Err: %s", err)
			return nil, err
		}
		image.Name = obj
		image.ObjectID = objID
		images = append(images, image)

	}
	return images, nil
}

// RetrievePageText function performs the operation of retrieving custom text related to existing project from pgx database with a query.
func RetrievePageText(ctx context.Context, storeDB *pgxpool.Pool, pageID uint) ([]models.TextObject, error) {

	var texts = []models.TextObject{}
	rows, err := storeDB.Query(ctx, "SELECT custom_text, ptop, pleft, style FROM page_has_text WHERE page_id = ($1);", pageID)
	if err != nil {
		log.Printf("Error happened when retrieving page text from pgx table. Err: %s", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var textObj models.TextObject
		if err = rows.Scan(&textObj.CustomText, &textObj.Ptop, &textObj.Pleft, &textObj.Style); err != nil {
			return nil, err
		}

		texts = append(texts, textObj)

	}
	return texts, nil
}


// DeletePage function performs the operation of deleting page from pgx database with a query.
func DeletePage(ctx context.Context, storeDB *pgxpool.Pool, pageID uint) (error) {

	_, err = storeDB.Exec(ctx, "DELETE FROM pages WHERE page_id=($1);",
		pageID,
	)
	if err != nil {
		log.Printf("Error happened when deleting page from pgx table. Err: %s", err)
		return err
	}

	_, err = storeDB.Exec(ctx, "DELETE FROM page_has_photos WHERE page_id=($1);",
		pageID,
	)
	if err != nil && err != pgx.ErrNoRows {
		log.Printf("Error happened when deleting page photos from pgx table. Err: %s", err)
		return err
	}

	_, err = storeDB.Exec(ctx, "DELETE FROM page_has_decorations WHERE page_id=($1);",
		pageID,
	)
	if err != nil && err != pgx.ErrNoRows {
		log.Printf("Error happened when deleting page decorations from pgx table. Err: %s", err)
		return err
	}

	_, err = storeDB.Exec(ctx, "DELETE FROM page_has_layout WHERE page_id=($1);",
		pageID,
	)
	if err != nil && err != pgx.ErrNoRows {
		log.Printf("Error happened when deleting page layout from pgx table. Err: %s", err)
		return err
	}

	_, err = storeDB.Exec(ctx, "DELETE FROM page_has_background WHERE page_id=($1);",
		pageID,
	)
	if err != nil && err != pgx.ErrNoRows {
		log.Printf("Error happened when deleting page background from pgx table. Err: %s", err)
		return err
	}

	_, err = storeDB.Exec(ctx, "DELETE FROM page_has_text WHERE page_id=($1);",
		pageID,
	)
	if err != nil && err != pgx.ErrNoRows {
		log.Printf("Error happened when deleting page text from pgx table. Err: %s", err)
		return err
	}

	return nil
}

// DeleteProject function performs the operation of deleting a  photobook project from pgx database with a query.
func DeleteProject(ctx context.Context, storeDB *pgxpool.Pool, projectID uint) (error) {

	rows, err := storeDB.Query(ctx, "SELECT page_id FROM pages WHERE project_id = ($1);", projectID)
	if err != nil {
		log.Printf("Error happened when retrieving pages from pgx table. Err: %s", err)
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var pageID uint
		if err = rows.Scan(&pageID); err != nil {
			log.Printf("Error happened when scanning pages. Err: %s", err)
			return err
		}
		err = DeletePage(ctx, storeDB, pageID)
		if err != nil {
			log.Printf("Error happened when deleting page from pgx table. Err: %s", err)
			return err
		}
	}

	_, err = storeDB.Exec(ctx, "DELETE FROM projects WHERE project_id=($1);",
		projectID,
	)
	if err != nil {
		log.Printf("Error happened when deleting project from pgx table. Err: %s", err)
		return err
	}

	return nil
}

// AddProjectEditor function performs the operation of adding entry into users_edit_projects to the db.
func AddProjectEditor(ctx context.Context, storeDB *pgxpool.Pool, email string, projectID uint, category string) (uint, error) {

	_, err = storeDB.Exec(ctx, "INSERT INTO users_edit_projects (email, project_id, category) VALUES ($1, $2, $3);",
		email,
		projectID,
		category,
	)
	if err != nil {
		log.Printf("Error happened when inserting a new entry into users_edit_projects table. Err: %s", err)
		return projectID, err
	}

	return projectID, nil

}

// RetrieveUserProjects function performs the operation of retrieving user photobook projects from pgx database with a query.
func RetrieveUserProjects(ctx context.Context, storeDB *pgxpool.Pool, userID uint) ([]models.ProjectObj, error) {

	var projectslice []models.ProjectObj
	var email string
	err = storeDB.QueryRow(ctx, "SELECT email FROM users WHERE users_id = ($1);", userID).Scan(&email)
	if err != nil {
			log.Printf("Error happened when retrieving user email data from db. Err: %s", err)
			return nil, err
	}
	rows, err := storeDB.Query(ctx, "SELECT project_id FROM users_edit_projects WHERE email = ($1);", email)
	if err != nil {
		log.Printf("Error happened when retrieving projects from pgx table. Err: %s", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {

		var projectObj models.ProjectObj
		var updateTimeStorage time.Time
		if err = rows.Scan(&projectObj.ProjectID); err != nil {
			log.Printf("Error happened when scanning projects. Err: %s", err)
			return nil, err
		}

		err = storeDB.QueryRow(ctx, "SELECT name, status, cover_image, last_edited_at FROM projects WHERE project_id = ($1) ORDER BY last_edited_at;", projectObj.ProjectID).Scan(&projectObj.Name, &projectObj.Status, &projectObj.CoverImage, &updateTimeStorage)
		if err != nil && err != pgx.ErrNoRows {
			log.Printf("Error happened when retrieving project data from db. Err: %s", err)
			return nil, err
		}
		
		projectObj.LastEditedAt = updateTimeStorage.Format(time.RFC3339)
		projectslice = append(projectslice, projectObj)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error happened when retrieving projects from pgx table. Err: %s", err)
		return nil, err
	}
	return projectslice, nil

}

// UpdateNewUserProjects function performs the operation of updating photobook projects for new use in pgx database with a query.
func UpdateNewUserProjects(ctx context.Context, storeDB *pgxpool.Pool, email string, userID uint) (error) {

	_, err = storeDB.Exec(ctx, "UPDATE users_edit_projects SET users_id = ($1) WHERE email = ($2);",
	userID,
	email,
	)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("Error happened when updating user id for view project pgx table. Err: %s", err)
		return err
	}
	
	return nil

}

