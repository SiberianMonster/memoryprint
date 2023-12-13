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
	err = storeDB.QueryRow(ctx, "SELECT CASE WHEN EXISTS (SELECT * FROM users_edit_projects WHERE projects_id = ($1) AND email = ($2)) THEN TRUE ELSE FALSE END;", projectID, email).Scan(&checkProject)
	if err != nil && err != pgx.ErrNoRows {
		log.Printf("Error happened when checking if user can edit project in db. Err: %s", err)
		return false
	}

	return checkProject
}

// CreateProject function performs the operation of creating a new photobook project in pgx database with a query.
func CreateProject(ctx context.Context, storeDB *pgxpool.Pool, userID uint, pageNumber int, orientation string, coverImage string, projectname string, covertype string, bindingtype string, papertype string, promooffersID uint) (uint, error) {

	t := time.Now()
	var pID uint
	var email string
	err := storeDB.QueryRow(ctx, "INSERT INTO projects (name, created_at, last_edited_at, status, orientation, cover_image, last_editor, users_id, covertype, bindingtype, papertype, promooffers_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) RETURNING projects_id;",
		projectname,
		t,
		t,
		"EDITED",
		orientation,
		coverImage,
		userID,
		userID,
		covertype,
		bindingtype,
		papertype,
		promooffersID,
	).Scan(&pID)
	if err != nil {
		log.Printf("Error happened when inserting a new project into pgx table. Err: %s", err)
		return 0, err
	}

	pagesRange := makeRange(1, pageNumber)

	for _, num := range pagesRange {
		_, err = storeDB.Exec(ctx, "INSERT INTO pages (last_edited_at, number, is_template, projects_id) VALUES ($1, $2, $3, $4);",
			t,
			num,
			false,
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


// CreateTemplate function performs the operation of creating a new photobook template in pgx database with a query.
func CreateTemplate(ctx context.Context, storeDB *pgxpool.Pool, userID uint, pageNumber int, orientation string, coverImage string, category string, hardCopy string, projectname string) (uint, error) {

	t := time.Now()
	var pID uint
	err := storeDB.QueryRow(ctx, "INSERT INTO templates (name, created_at, last_edited_at, status, orientation, category, hardcopy, cover_image, last_editor, users_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING templates_id;",
		projectname,
		t,
		t,
		"EDITED",
		orientation,
		category,
		hardCopy,
		coverImage,
		userID,
		userID,
	).Scan(&pID)
	if err != nil {
		log.Printf("Error happened when inserting a new template into pgx table. Err: %s", err)
		return pID, err
	}

	pagesRange := makeRange(1, pageNumber)

	for _, num := range pagesRange {
		
		_, err = storeDB.Exec(ctx, "INSERT INTO pages (last_edited_at, number, is_template, projects_id) VALUES ($1, $2, $3, $4) RETURNING pages_id;",
			t,
			num,
			true,
			pID,
		)
		if err != nil {
			log.Printf("Error happened when inserting a new page into pgx table. Err: %s", err)
			return pID, err
		}
    }
	return pID, nil
}


// SaveProject function performs the operation of saving data related to existing project.
func SaveProject(ctx context.Context, storeDB *pgxpool.Pool, projectObj models.ProjectObj, userID uint) error {

	var err error

	t := time.Now()

	if projectObj.ProjectID == 0{
		log.Printf("Project object not provided or malformed")
		return errors.New("Project object not provided or malformed")
	}

	_, err = storeDB.Exec(ctx, "UPDATE projects SET (name, last_edited_at, orientation, cover_image, last_editor) = ($1, $2, $3, $4, $5) WHERE projects_id = ($6);",
		projectObj.Name,
		t,
		projectObj.Orientation,
		projectObj.CoverImage,
		userID,
		projectObj.ProjectID,
	)
	if err != nil {
		log.Printf("Error happened when saving a project in pgx table. Err: %s", err)
		return err
	}
	return nil

}

// SaveTemplate function performs the operation of saving data related to existing project.
func SaveTemplate(ctx context.Context, storeDB *pgxpool.Pool, projectObj models.ProjectObj, userID uint) error {

	var err error

	t := time.Now()

	_, err = storeDB.Exec(ctx, "UPDATE templates SET (name, last_edited_at, orientation, cover_image) = ($1, $2, $3, $4) WHERE templates_id = ($5);",
		projectObj.Name,
		t,
		projectObj.Orientation,
		projectObj.CoverImage,
		projectObj.TemplateID,
	)
	if err != nil {
		log.Printf("Error happened when saving template into pgx table. Err: %s", err)
		return err
	}
	return nil

}

// PublishTemplate function performs the operation of saving data related to existing project.
func PublishTemplate(ctx context.Context, storeDB *pgxpool.Pool, projectObj models.TemplateProjectObj) error {

	var err error

	_, err = storeDB.Exec(ctx, "UPDATE templates SET (status, hardcopy) = ($1, $2) WHERE templates_id = ($3);",
		projectObj.HardCopy,
		"PUBLISHED",
		projectObj.TemplateID,
	)
	if err != nil {
		log.Printf("Error happened when inserting template hardcopy into pgx table. Err: %s", err)
		return err
	}
	return nil

}

// RetrieveProjectPages function performs the operation of retrieving a photobook project from pgx database with a query.
func RetrieveProjectPages(ctx context.Context, storeDB *pgxpool.Pool, projectID uint, isTemplate bool) ([]uint, error) {

	var pageslice []uint
	rows, err := storeDB.Query(ctx, "SELECT pages_id FROM pages WHERE projects_id = ($1) AND is_template = ($2) ORDER BY number;", projectID, isTemplate)
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
func AddProjectPage(ctx context.Context, storeDB *pgxpool.Pool, projectID uint) (uint, error) {

	var pageslice []models.Page
	var pID uint
	rows, err := storeDB.Query(ctx, "SELECT pages_id FROM pages WHERE projects_id = ($1);", projectID)
	if err != nil {
		log.Printf("Error happened when retrieving pages from pgx table. Err: %s", err)
		return pID, err
	}
	defer rows.Close()

	for rows.Next() {
		var page models.Page
		if err = rows.Scan(&page.PageID); err != nil {
			log.Printf("Error happened when scanning pages. Err: %s", err)
			return pID, err
		}
		pageslice = append(pageslice, page)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error happened when retrieving pages from pgx table. Err: %s", err)
		return pID, err
	}

	t := time.Now()
	newPageNum := len(pageslice) + 1
	err = storeDB.QueryRow(ctx, "INSERT INTO pages (last_edited_at, number, is_template, projects_id) VALUES ($1, $2, $3, $4) RETURNING pages_id;",
			t,
			newPageNum,
			false,
			projectID,
		).Scan(&pID)
	if err != nil {
			log.Printf("Error happened when inserting a new page into pgx table. Err: %s", err)
			return pID, err
	}
	return pID, nil

}

// DuplicateProjectPage function performs the operation of duplicating existing photobook project page to pgx database with a query.
func DuplicateProjectPage(ctx context.Context, storeDB *pgxpool.Pool, duplicateID uint) (uint, error) {

	var pageslice []models.Page
	var pID uint
	var projectID uint

	err = storeDB.QueryRow(ctx, "SELECT projects_id FROM pages WHERE pages_id = ($1);", duplicateID).Scan(&projectID)
	if err != nil {
			log.Printf("Error happened when retrieving project ID from pgx table. Err: %s", err)
			return pID, err
	}

	rows, err := storeDB.Query(ctx, "SELECT pages_id FROM pages WHERE projects_id = ($1);", projectID)
	if err != nil {
		log.Printf("Error happened when retrieving pages from pgx table. Err: %s", err)
		return pID, err
	}
	defer rows.Close()

	for rows.Next() {
		var page models.Page
		if err = rows.Scan(&page.PageID); err != nil {
			log.Printf("Error happened when scanning pages. Err: %s", err)
			return pID, err
		}
		pageslice = append(pageslice, page)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error happened when retrieving pages from pgx table. Err: %s", err)
		return pID, err
	}

	t := time.Now()
	newPageNum := len(pageslice) + 1
	err = storeDB.QueryRow(ctx, "INSERT INTO pages (last_edited_at, number, projects_id) VALUES ($1, $2, $3) RETURNING pages_id;",
			t,
			newPageNum,
			projectID,
		).Scan(&pID)
	if err != nil {
			log.Printf("Error happened when inserting a new page into pgx table. Err: %s", err)
			return pID, err
	}

	_, err = storeDB.Exec(ctx, "INSERT INTO page_has_photos (pages_id, photos_id, ptop, pleft, style, last_edited_at) SELECT $1, photos_id, ptop, pleft, style, $2 FROM page_has_photos WHERE pages_id = $3;",
			pID,
			t,
			duplicateID,
	)
	if err != nil {
			log.Printf("Error happened when duplicating page photos into pgx table. Err: %s", err)
			return pID, err
	}

	_, err = storeDB.Exec(ctx, "INSERT INTO page_has_decoration (pages_id, decorations_id, ptop, pleft, style, last_edited_at) SELECT $1, decorations_id, ptop, pleft, style, $2 FROM page_has_decoration WHERE pages_id = $3;",
			pID,
			t,
			duplicateID,
	)
	if err != nil {
			log.Printf("Error happened when duplicating page decorations into pgx table. Err: %s", err)
			return pID, err
	}

	_, err = storeDB.Exec(ctx, "INSERT INTO page_has_layout (pages_id, layouts_id, last_edited_at) SELECT $1, layouts_id, $2 FROM page_has_layout WHERE pages_id = $3;",
			pID,
			t,
			duplicateID,
	)
	if err != nil {
			log.Printf("Error happened when duplicating page layout into pgx table. Err: %s", err)
			return pID, err
	}

	_, err = storeDB.Exec(ctx, "INSERT INTO page_has_background (pages_id, backgrounds_id, last_edited_at) SELECT $1, backgrounds_id, $2 FROM page_has_background WHERE pages_id = $3;",
			pID,
			t,
			duplicateID,
	)
	if err != nil {
			log.Printf("Error happened when duplicating page background into pgx table. Err: %s", err)
			return pID, err
	}

	_, err = storeDB.Exec(ctx, "INSERT INTO page_has_text (pages_id, custom_text, ptop, pleft, style) SELECT $1, custom_text, ptop, pleft, style FROM page_has_text WHERE pages_id = $2;",
			pID,
			duplicateID,
	)
	if err != nil {
			log.Printf("Error happened when duplicating page texts into pgx table. Err: %s", err)
			return pID, err
	}

	return pID, nil

}


// CreateProjectFromTemplate function performs the operation of duplicating existing photobook project page to pgx database with a query.
func CreateProjectFromTemplate(ctx context.Context, storeDB *pgxpool.Pool, userID uint, templateID uint, covertype string, bindingtype string, papertype string, promooffersID uint) (uint, []uint, error) {

	var pageslice []models.Page
	var newpageslice []uint
	var pID uint
	var projectID uint
	var name, orientation, coverImage string
	t := time.Now()

	err := storeDB.QueryRow(ctx, "SELECT name, orientation, cover_image FROM templates WHERE templates_id = ($1);", templateID).Scan(&name, &orientation, &coverImage)
	if err != nil {
		log.Printf("Error happened when retrieving template data. Err: %s", err)
		return projectID, newpageslice, err
	}
	err = storeDB.QueryRow(ctx, "INSERT INTO projects (name, created_at, last_edited_at, status, orientation, cover_image, last_editor, users_id, covertype, bindingtype, papertype, promooffers_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) RETURNING projects_id;",
		name,
		t,
		t,
		"EDITED",
		orientation,
		coverImage,
		userID,
		userID,
		covertype,
		bindingtype,
		papertype,
		promooffersID,
	).Scan(&projectID)
	if err != nil {
		log.Printf("Error happened when inserting a new project into pgx table. Err: %s", err)
		return projectID, newpageslice, err
	}

	rows, err := storeDB.Query(ctx, "SELECT pages_id FROM pages WHERE projects_id = ($1) and is_template = ($2);", templateID, true)
	if err != nil {
		log.Printf("Error happened when retrieving pages from pgx table. Err: %s", err)
		return projectID, newpageslice, err
	}
	defer rows.Close()

	for rows.Next() {
		var page models.Page
		if err = rows.Scan(&page.PageID); err != nil {
			log.Printf("Error happened when scanning pages. Err: %s", err)
			return projectID, newpageslice, err
		}
		pageslice = append(pageslice, page)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error happened when retrieving pages from pgx table. Err: %s", err)
		return projectID, newpageslice, err
	}


	pagesRange := makeRange(1, len(pageslice))

	for _, num := range pagesRange {
		err = storeDB.QueryRow(ctx, "INSERT INTO pages (last_edited_at, number, is_template, projects_id) VALUES ($1, $2, $3, $4) RETURNING pages_id;",
			t,
			num,
			false,
			projectID,
		).Scan(&pID)
		if err != nil {
			log.Printf("Error happened when inserting a new page into pgx table. Err: %s", err)
			return projectID, newpageslice, err
		}
		_, err = storeDB.Exec(ctx, "INSERT INTO page_has_decoration (pages_id, decorations_id, ptop, pleft, style, last_edited_at) SELECT $1, decorations_id, ptop, pleft, style, $2 FROM page_has_decoration WHERE pages_id = $3;",
			pID,
			t,
			pageslice[num].PageID,
		)
		if err != nil {
				log.Printf("Error happened when duplicating page decorations into pgx table. Err: %s", err)
				return projectID, newpageslice, err
		}

		_, err = storeDB.Exec(ctx, "INSERT INTO page_has_layout (pages_id, layouts_id, last_edited_at) SELECT $1, layouts_id, $2 FROM page_has_layout WHERE pages_id = $3;",
				pID,
				t,
				pageslice[num].PageID,
		)
		if err != nil {
				log.Printf("Error happened when duplicating page layout into pgx table. Err: %s", err)
				return projectID, newpageslice, err
		}

		_, err = storeDB.Exec(ctx, "INSERT INTO page_has_background (pages_id, backgrounds_id, last_edited_at) SELECT $1, backgrounds_id, $2 FROM page_has_background WHERE pages_id = $3;",
				pID,
				t,
				pageslice[num].PageID,
		)
		if err != nil {
				log.Printf("Error happened when duplicating page background into pgx table. Err: %s", err)
				return projectID, newpageslice, err
		}

		_, err = storeDB.Exec(ctx, "INSERT INTO page_has_text (pages_id, custom_text, ptop, pleft, style) SELECT $1, custom_text, ptop, pleft, style FROM page_has_text WHERE pages_id = $2;",
				pID,
				pageslice[num].PageID,
		)
		if err != nil {
				log.Printf("Error happened when duplicating page texts into pgx table. Err: %s", err)
				return projectID, newpageslice, err
		}
		newpageslice = append(newpageslice, pID)

    }

	

	return projectID, newpageslice, nil

}


// AddProjectPhotos function performs the operation of assigning user photos to a new project from pgx database with a query.
func AddProjectPhotos(ctx context.Context, storeDB *pgxpool.Pool, projectID uint, userID uint) error {

	rows, err := storeDB.Query(ctx, "SELECT photos_id FROM photos WHERE users_id = ($1);", userID)
	if err != nil {
		log.Printf("Error happened when retrieving photos from pgx table. Err: %s", err)
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var photoID uint
		if err = rows.Scan(&photoID); err != nil {
			return err
		}
		_, err = storeDB.Exec(ctx, "INSERT INTO project_has_photos (photos_id, projects_id) VALUES ($1, $2);",
			photoID,
			projectID,
		)
		if err != nil {
			log.Printf("Error happened when inserting a new photo project relation into pgx table. Err: %s", err)
			return err
		}
	}

	return nil

}

// RetrieveProjectPhotos function performs the operation of retrieving photos related to existing project from pgx database with a query.
func RetrieveProjectPhotos(ctx context.Context, storeDB *pgxpool.Pool, projectID uint) ([]string, error) {

	var photoslice []string
	rows, err := storeDB.Query(ctx, "SELECT photos_id FROM project_has_photos WHERE projects_id = ($1);", projectID)
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
		photoslice = append(photoslice, photo)
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

	_, err = storeDB.Exec(ctx, "DELETE FROM page_has_photos WHERE pages_id=($1);",
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

	_, err = tx.Prepare(ctx, "my-query-photo","INSERT INTO page_has_photos (pages_id, photos_id, ptop, pleft, style, last_edited_at) VALUES ($1, $2, $3, $4, $5, $6);")
	if err != nil {
		log.Printf("Error happened when preparing pgx transaction context. Err: %s", err)
		return err
	}

	for _, v := range images {
		if _, err = tx.Exec(ctx, "my-query-photo", pageID, v.ObjectID, v.Ptop, v.Pleft, v.Style, t); err != nil {
			log.Printf("Error happened when declaring transaction. Err: %s", err)
			return err
		}
	}

	return tx.Commit(ctx)

}

// SavePageDecorations function performs the operation of saving edited decorations related to existing project.
func SavePageDecorations(ctx context.Context, storeDB *pgxpool.Pool, pageID uint, images ArtObjects) error {

	var err error

	_, err = storeDB.Exec(ctx, "DELETE FROM page_has_decoration WHERE pages_id=($1);",
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

	_, err = tx.Prepare(ctx, "my-query-decor", "INSERT INTO page_has_decoration (pages_id, decorations_id, ptop, pleft, style, last_edited_at) VALUES ($1, $2, $3, $4, $5, $6);")
	if err != nil {
		log.Printf("Error happened when preparing pgx transaction context. Err: %s", err)
		return err
	}

	for _, v := range images {
		if _, err = tx.Exec(ctx, "my-query-decor", pageID, v.ObjectID, v.Ptop, v.Pleft, v.Style, t); err != nil {
			log.Printf("Error happened when declaring transaction. Err: %s", err)
			return err
		}
	}

	return tx.Commit(ctx)

}

// SavePageLayout function performs the operation of saving layout related to existing project.
func SavePageLayout(ctx context.Context, storeDB *pgxpool.Pool, pageID uint, images ArtObjects) error {

	var err error

	_, err = storeDB.Exec(ctx, "DELETE FROM page_has_layout WHERE pages_id=($1);",
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

	_, err = tx.Prepare(ctx, "my-query-layout", "INSERT INTO page_has_layout (pages_id, layouts_id, last_edited_at) VALUES ($1, $2, $3);")
	if err != nil {
		log.Printf("Error happened when preparing pgx transaction context. Err: %s", err)
		return err
	}

	for _, v := range images {
		if _, err = tx.Exec(ctx, "my-query-layout", pageID, v.ObjectID, t); err != nil {
			log.Printf("Error happened when declaring transaction. Err: %s", err)
			return err
		}
	}

	return tx.Commit(ctx)

}

// SavePageBackground function performs the operation of saving background related to existing project.
func SavePageBackground(ctx context.Context, storeDB *pgxpool.Pool, pageID uint, images ArtObjects) error {

	var err error

	_, err = storeDB.Exec(ctx, "DELETE FROM page_has_background WHERE pages_id=($1);",
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

	_, err = tx.Prepare(ctx, "my-query-b", "INSERT INTO page_has_background (pages_id, backgrounds_id, last_edited_at) VALUES ($1, $2, $3);")
	if err != nil {
		log.Printf("Error happened when preparing pgx transaction context. Err: %s", err)
		return err
	}

	for _, v := range images {
		if _, err = tx.Exec(ctx, "my-query-b", pageID, v.ObjectID, t); err != nil {
			log.Printf("Error happened when declaring transaction. Err: %s", err)
			return err
		}
	}

	return tx.Commit(ctx)

}

// SavePageText function performs the operation of saving custom text related to existing project.
func SavePageText(ctx context.Context, storeDB *pgxpool.Pool, pageID uint, textObjs []models.TextObject) error {

	var err error

	_, err = storeDB.Exec(ctx, "DELETE FROM page_has_text WHERE pages_id=($1);",
		pageID,
	)
	if err != nil && err != pgx.ErrNoRows {
		log.Printf("Error happened when deleting old page text from pgx table. Err: %s", err)
		return err
	}

	tx, err := storeDB.Begin(ctx)
	if err != nil {
		log.Printf("Error happened when initiating pgx transaction. Err: %s", err)
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Prepare(ctx, "my-query-text", "INSERT INTO page_has_text (pages_id, custom_text, ptop, pleft, style) VALUES ($1, $2, $3, $4, $5);")
	if err != nil {
		log.Printf("Error happened when preparing pgx transaction context. Err: %s", err)
		return err
	}

	for _, v := range textObjs {
		if _, err = tx.Exec(ctx, "my-query-text", pageID, v.CustomText, v.Ptop, v.Pleft, v.Style); err != nil {
			log.Printf("Error happened when declaring transaction. Err: %s", err)
			return err
		}
	}

	return tx.Commit(ctx)

}

// RetrievePagePhotos function performs the operation of retrieving photos related to existing project from pgx database with a query.
func RetrievePagePhotos(ctx context.Context, storeDB *pgxpool.Pool, pageID uint) ([]models.ArtObject, error) {

	var images = []models.ArtObject{}
	rows, err := storeDB.Query(ctx, "SELECT photos_id, ptop, pleft, style FROM page_has_photos WHERE pages_id = ($1);", pageID)
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
	rows, err := storeDB.Query(ctx, "SELECT decorations_id, ptop, pleft, style FROM page_has_decoration WHERE pages_id = ($1);", pageID)
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
	rows, err := storeDB.Query(ctx, "SELECT layouts_id FROM page_has_layout WHERE pages_id = ($1);", pageID)
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
	rows, err := storeDB.Query(ctx, "SELECT backgrounds_id FROM page_has_background WHERE pages_id = ($1);", pageID)
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
	rows, err := storeDB.Query(ctx, "SELECT custom_text, ptop, pleft, style FROM page_has_text WHERE pages_id = ($1);", pageID)
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

	_, err = storeDB.Exec(ctx, "DELETE FROM pages WHERE pages_id=($1);",
		pageID,
	)
	if err != nil {
		log.Printf("Error happened when deleting page from pgx table. Err: %s", err)
		return err
	}

	_, err = storeDB.Exec(ctx, "DELETE FROM page_has_photos WHERE pages_id=($1);",
		pageID,
	)
	if err != nil && err != pgx.ErrNoRows {
		log.Printf("Error happened when deleting page photos from pgx table. Err: %s", err)
		return err
	}

	_, err = storeDB.Exec(ctx, "DELETE FROM page_has_decoration WHERE pages_id=($1);",
		pageID,
	)
	if err != nil && err != pgx.ErrNoRows {
		log.Printf("Error happened when deleting page decoration from pgx table. Err: %s", err)
		return err
	}

	_, err = storeDB.Exec(ctx, "DELETE FROM page_has_layout WHERE pages_id=($1);",
		pageID,
	)
	if err != nil && err != pgx.ErrNoRows {
		log.Printf("Error happened when deleting page layout from pgx table. Err: %s", err)
		return err
	}

	_, err = storeDB.Exec(ctx, "DELETE FROM page_has_background WHERE pages_id=($1);",
		pageID,
	)
	if err != nil && err != pgx.ErrNoRows {
		log.Printf("Error happened when deleting page background from pgx table. Err: %s", err)
		return err
	}

	_, err = storeDB.Exec(ctx, "DELETE FROM page_has_text WHERE pages_id=($1);",
		pageID,
	)
	if err != nil && err != pgx.ErrNoRows {
		log.Printf("Error happened when deleting page text from pgx table. Err: %s", err)
		return err
	}

	return nil
}

// DeletePageObjects function performs the operation of deleting page related objects from pgx database with a query.
func DeletePageObjects(ctx context.Context, storeDB *pgxpool.Pool, pageID uint) (error) {

	_, err = storeDB.Exec(ctx, "DELETE FROM page_has_photos WHERE pages_id=($1);",
		pageID,
	)
	if err != nil && err != pgx.ErrNoRows {
		log.Printf("Error happened when deleting page photos from pgx table. Err: %s", err)
		return err
	}

	_, err = storeDB.Exec(ctx, "DELETE FROM page_has_decoration WHERE pages_id=($1);",
		pageID,
	)
	if err != nil && err != pgx.ErrNoRows {
		log.Printf("Error happened when deleting page decoration from pgx table. Err: %s", err)
		return err
	}

	_, err = storeDB.Exec(ctx, "DELETE FROM page_has_layout WHERE pages_id=($1);",
		pageID,
	)
	if err != nil && err != pgx.ErrNoRows {
		log.Printf("Error happened when deleting page layout from pgx table. Err: %s", err)
		return err
	}

	_, err = storeDB.Exec(ctx, "DELETE FROM page_has_background WHERE pages_id=($1);",
		pageID,
	)
	if err != nil && err != pgx.ErrNoRows {
		log.Printf("Error happened when deleting page background from pgx table. Err: %s", err)
		return err
	}

	_, err = storeDB.Exec(ctx, "DELETE FROM page_has_text WHERE pages_id=($1);",
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

	rows, err := storeDB.Query(ctx, "SELECT pages_id FROM pages WHERE projects_id = ($1);", projectID)
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

	_, err = storeDB.Exec(ctx, "DELETE FROM projects WHERE projects_id=($1);",
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

	_, err = storeDB.Exec(ctx, "INSERT INTO users_edit_projects (email, projects_id, category) VALUES ($1, $2, $3);",
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
func RetrieveUserProjects(ctx context.Context, storeDB *pgxpool.Pool, userID uint) ([]models.RetrievedUserProject, error) {

	var projectslice []models.RetrievedUserProject
	var email string
	err = storeDB.QueryRow(ctx, "SELECT email FROM users WHERE users_id = ($1);", userID).Scan(&email)
	if err != nil {
			log.Printf("Error happened when retrieving user email data from db. Err: %s", err)
			return nil, err
	}
	rows, err := storeDB.Query(ctx, "SELECT projects_id, category FROM users_edit_projects WHERE email = ($1);", email)
	if err != nil {
		log.Printf("Error happened when retrieving projects from pgx table. Err: %s", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {

		var userProjectObj models.RetrievedUserProject
		var projectObj models.ProjectObj
		var updateTimeStorage time.Time
		if err = rows.Scan(&projectObj.ProjectID, userProjectObj.Ownership.Category); err != nil {
			log.Printf("Error happened when scanning projects. Err: %s", err)
			return nil, err
		}

		err = storeDB.QueryRow(ctx, "SELECT name, orientation, cover_image, last_edited_at FROM projects WHERE projects_id = ($1) ORDER BY last_edited_at;", projectObj.ProjectID).Scan(&projectObj.Name, &projectObj.Orientation, &projectObj.CoverImage, &updateTimeStorage)
		if err != nil && err != pgx.ErrNoRows {
			log.Printf("Error happened when retrieving project data from db. Err: %s", err)
			return nil, err
		}
		
		projectObj.LastEditedAt = updateTimeStorage.Format(time.RFC3339)
		userProjectObj.Project = projectObj
		projectslice = append(projectslice, userProjectObj)
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

// RetrieveTemplates function performs the operation of retrieving photobook templates from pgx database with a query.
func RetrieveTemplates(ctx context.Context, storeDB *pgxpool.Pool) ([]models.TemplateProjectObj, error) {

	var projectslice []models.TemplateProjectObj
	
	rows, err := storeDB.Query(ctx, "SELECT projects_id FROM projects WHERE is_template = ($1);", true)
	if err != nil {
		log.Printf("Error happened when retrieving templates from pgx table. Err: %s", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {

		var projectObj models.TemplateProjectObj
		if err = rows.Scan(&projectObj.TemplateID); err != nil {
			log.Printf("Error happened when scanning templates. Err: %s", err)
			return nil, err
		}

		err = storeDB.QueryRow(ctx, "SELECT name, category, orientation, cover_image, hardcopy FROM templates WHERE templates_id = ($1) ORDER BY last_edited_at;", projectObj.TemplateID).Scan(&projectObj.Name, &projectObj.Category, &projectObj.Orientation, &projectObj.HardCopy, &projectObj.CoverImage)
		if err != nil && err != pgx.ErrNoRows {
			log.Printf("Error happened when retrieving templates data from db. Err: %s", err)
			return nil, err
		}
		
		projectslice = append(projectslice, projectObj)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error happened when retrieving templates from pgx table. Err: %s", err)
		return nil, err
	}
	return projectslice, nil

}