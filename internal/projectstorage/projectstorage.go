// Storage package contains functions for storing photos and projects in a pgx database.
//
// Available at https://github.com/SiberianMonster/memoryprint/tree/development/internal/projectstorage
package projectstorage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/SiberianMonster/memoryprint/internal/models"
	"log"
	"time"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var err error
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

func CheckPage(ctx context.Context, storeDB *pgxpool.Pool, pageID uint, projectID uint) bool {

	var checkPage bool

	err = storeDB.QueryRow(ctx, "SELECT CASE WHEN EXISTS (SELECT * FROM pages WHERE pages_id = ($1) AND projects_id = ($2)) THEN TRUE ELSE FALSE END;", pageID, projectID).Scan(&checkPage)
	if err != nil && err != pgx.ErrNoRows {
		log.Printf("Error happened when checking for page id in db. Err: %s", err)
		return false
	}

	return checkPage
}

func CheckCoverPage(ctx context.Context, storeDB *pgxpool.Pool, pageID uint) bool {
	var ptype string
	
	err := storeDB.QueryRow(ctx, "SELECT type FROM pages WHERE pages_id = ($1);", pageID).Scan(&ptype)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("Error happened when counting pages. Err: %s", err)
		return true
	}


	if ptype == "front" || ptype == "back"{
		log.Printf("Attempt to change cover page. Err: %s", err)
		return true
	}
	return false
}

func CheckAllPagesPassed(ctx context.Context, storeDB *pgxpool.Pool, slicePassed uint, projectID uint, isTemplate bool) bool {
	var countPage uint
	
	err := storeDB.QueryRow(ctx, "SELECT COUNT(pages_id) FROM pages WHERE projects_id = ($1) AND is_template = ($2);", projectID, isTemplate).Scan(&countPage)
	if err != nil {
		log.Printf("Error happened when counting pages. Err: %s", err)
		return false
	}

	slicePassed = slicePassed + 2
	if slicePassed != countPage {
		log.Printf("Error not all pages passed. Err: %s", err)
		return false
	}
	return true
}

func CheckPagesRange(ctx context.Context, storeDB *pgxpool.Pool, sort uint, projectID uint, isTemplate bool) bool {
	var countPage uint
	
	err := storeDB.QueryRow(ctx, "SELECT COUNT(pages_id) FROM pages WHERE projects_id = ($1) AND is_template = ($2);", projectID, isTemplate).Scan(&countPage)
	if err != nil {
		log.Printf("Error happened when counting pages. Err: %s", err)
		return false
	}
	if sort >= countPage  || sort == 0 {
		log.Printf("Attempt to change cover or non-existing page. Err: %s", err)
		return false
	}
	return true
}


// CreateProject function performs the operation of creating a new photobook project in pgx database with a query.
func CreateProject(ctx context.Context, storeDB *pgxpool.Pool, userID uint, projectObj models.NewBlankProjectObj, promooffersID uint) (uint, error) {

	t := time.Now()
	var pID uint
	var email string
	err := storeDB.QueryRow(ctx, "INSERT INTO projects (name, created_at, last_edited_at, status, size, variant, count_pages, users_id, last_editor, cover, paper, promooffers_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) RETURNING projects_id;",
		projectObj.Name,
		t,
		t,
		"EDITED",
		projectObj.Size,
		projectObj.Variant,
		23,
		userID,
		userID,
		projectObj.Cover,
		projectObj.Surface,
		promooffersID,
	).Scan(&pID)
	if err != nil {
		log.Printf("Error happened when inserting a new project into pgx table. Err: %s", err)
		return 0, err
	}

	err = storeDB.QueryRow(ctx, "SELECT email FROM users WHERE users_id = ($1);", userID).Scan(&email)
	if err != nil {
			log.Printf("Error happened when retrieving user email data from db. Err: %s", err)
			return pID, err
	}

	_, err = storeDB.Exec(ctx, "INSERT INTO users_edit_projects (projects_id, email, users_id, category) VALUES ($1, $2, $3, $4);",
		pID,
		email,
		userID,
		"OWNER",
	)
	if err != nil {
		log.Printf("Error happened when publishing template. Err: %s", err)
		return 0, err
	}

	pagesRange := makeRange(0, 22)
	if projectObj.TemplateID != 0 {
		var templatePages []models.Page

		templatePages, err = RetrieveProjectPages(ctx, storeDB, projectObj.TemplateID, true)
		if err != nil {
			log.Printf("Error happened when retrieving template pages from db. Err: %s", err)
			return 0, err
		}
		for _, page := range templatePages {
			var strdata *string
			err = storeDB.QueryRow(ctx, "SELECT data FROM pages WHERE pages_id = ($1);", page.PageID).Scan(&strdata)
			if err != nil {
					log.Printf("Error happened when retrieving template page data from db. Err: %s", err)
					return 0, err
			}
			_, err = storeDB.Exec(ctx, "INSERT INTO pages (last_edited_at, sort, type, is_template, creating_image_link, data, projects_id) VALUES ($1, $2, $3, $4, $5, $6, $7);",
			t,
			page.Sort,
			page.Type,
			false,
			page.CreatingImageLink,
			strdata,
			pID,
			)
			if err != nil {
				log.Printf("Error happened when inserting a new page from template into pgx table. Err: %s", err)
				return pID, err
			}
			
		}
	} else {
		for _, num := range pagesRange {

			var ptype string
	
			if num == 22 {
				ptype = "back"
			} else {
				ptype = "page"
			}
			if num == 0 {
				ptype = "front"
			}
			_, err = storeDB.Exec(ctx, "INSERT INTO pages (last_edited_at, sort, type, is_template, projects_id) VALUES ($1, $2, $3, $4, $5);",
				t,
				num,
				ptype,
				false,
				pID,
			)
		}
	}

	
	log.Printf("added new project to DB")
	return pID, nil
}


// CreateTemplate function performs the operation of creating a new photobook template in pgx database with a query.
func CreateTemplate(ctx context.Context, storeDB *pgxpool.Pool, name string, size string, category string) (uint, error) {

	t := time.Now()
	var tID uint
	err := storeDB.QueryRow(ctx, "INSERT INTO templates (name, created_at, last_edited_at, status, size, category) VALUES ($1, $2, $3, $4, $5, $6) RETURNING templates_id;",
		name,
		t,
		t,
		"EDITED",
		size,
		category,
	).Scan(&tID)
	if err != nil {
		log.Printf("Error happened when inserting a new template into pgx table. Err: %s", err)
		return tID, err
	}

	pagesRange := makeRange(0, 22)

	for _, num := range pagesRange {
		var ptype string

		if num == 22 {
			ptype = "back"
		} else {
			ptype = "page"
		}
		if num == 0 {
			ptype = "front"
		}
		_, err = storeDB.Exec(ctx, "INSERT INTO pages (last_edited_at, type, sort, is_template, projects_id) VALUES ($1, $2, $3, $4, $5);",
			t,
			ptype,
			num,
			true,
			tID,
		)
		if err != nil {
			log.Printf("Error happened when inserting a new page into pgx table. Err: %s", err)
			return tID, err
		}
	}
	return tID, nil
}


// PublishTemplate function performs the operation of saving data related to existing project.
func PublishTemplate(ctx context.Context, storeDB *pgxpool.Pool, templateID uint) error {

	var err error

	_, err = storeDB.Exec(ctx, "UPDATE templates SET status = ($1) WHERE templates_id = ($2);",
		"PUBLISHED",
		templateID,
	)
	if err != nil {
		log.Printf("Error happened when publishing template. Err: %s", err)
		return err
	}
	return nil

}

// RetrieveUserProjects function performs the operation of retrieving user photobook projects from pgx database with a query.
func RetrieveUserProjects(ctx context.Context, storeDB *pgxpool.Pool, userID uint) ([]models.ResponseProject, error) {

	projectslice := []models.ResponseProject{}
	var email string
	err = storeDB.QueryRow(ctx, "SELECT email FROM users WHERE users_id = ($1);", userID).Scan(&email)
	if err != nil {
			log.Printf("Error happened when retrieving user email data from db. Err: %s", err)
			return nil, err
	}
	rows, err := storeDB.Query(ctx, "SELECT projects_id FROM users_edit_projects WHERE email = ($1);", email)
	if err != nil {
		log.Printf("Error happened when retrieving projects from pgx table. Err: %s", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {

		var projectObj models.ResponseProject
		var pID uint
		var updateTimeStorage time.Time
		if err = rows.Scan(&pID); err != nil {
			log.Printf("Error happened when scanning projects. Err: %s", err)
			return nil, err
		}

		err = storeDB.QueryRow(ctx, "SELECT name, preview_image_link, count_pages, size, variant, last_edited_at FROM projects WHERE projects_id = ($1) ORDER BY last_edited_at DESC;", pID).Scan(&projectObj.Name, &projectObj.PreviewImageLink, &projectObj.CountPages, &projectObj.Size, &projectObj.Variant, &updateTimeStorage)
		if err != nil && err != pgx.ErrNoRows {
			log.Printf("Error happened when retrieving project data from db. Err: %s", err)
			return nil, err
		}
		projectObj.ProjectID = pID
		projectObj.LastEditedAt = updateTimeStorage.Unix()
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


// LoadProject function performs the operation of retrieving project by id from pgx database with a query.
func LoadProject(ctx context.Context, storeDB *pgxpool.Pool, pID uint) (models.ProjectObj, error) {

	var projectObj models.ProjectObj
	var updateTimeStorage time.Time
	var createTimeStorage time.Time
	err := storeDB.QueryRow(ctx, "SELECT name, size, variant, created_at, paper, last_edited_at FROM projects WHERE projects_id = ($1);", pID).Scan(&projectObj.Name, &projectObj.Size, &projectObj.Variant, &createTimeStorage, &projectObj.Surface, &updateTimeStorage)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("Error happened when retrieving project from pgx table. Err: %s", err)
		return projectObj, err
	}
	projectObj.LastEditedAt = updateTimeStorage.Unix()
	projectObj.CreatedAt = createTimeStorage.Unix()

	return projectObj, nil

}

// LoadTemplate function performs the operation of retrieving template by id from pgx database with a query.
func LoadTemplate(ctx context.Context, storeDB *pgxpool.Pool, pID uint) (models.SavedTemplateObj, error) {

	var projectObj models.SavedTemplateObj
	var updateTimeStorage time.Time
	var createTimeStorage time.Time
	err := storeDB.QueryRow(ctx, "SELECT name, size, created_at, last_edited_at FROM templates WHERE templates_id = ($1) AND status = ($2);", pID, "PUBLISHED").Scan(&projectObj.Name, &projectObj.Size, &createTimeStorage, &updateTimeStorage)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("Error happened when retrieving project from pgx table. Err: %s", err)
		return projectObj, err
	}
	projectObj.LastEditedAt = updateTimeStorage.Unix()
	projectObj.CreatedAt = createTimeStorage.Unix()
	projectObj.Pages, err = RetrieveTemplatePages(ctx, storeDB, pID)
	if err != nil {
		log.Printf("Error happened when retrieving template pages from pgx table. Err: %s", err)
		return projectObj, err
	}

	return projectObj, nil

}

// RetrieveProjectPages function performs the operation of retrieving a photobook project from pgx database with a query.
func RetrieveProjectPages(ctx context.Context, storeDB *pgxpool.Pool, projectID uint, isTemplate bool) ([]models.Page, error) {

	var pageslice []models.Page
	rows, err := storeDB.Query(ctx, "SELECT pages_id, type, sort, creating_image_link, data FROM pages WHERE projects_id = ($1) AND is_template = ($2) ORDER BY sort;", projectID, isTemplate)
	if err != nil {
		log.Printf("Error happened when retrieving pages from pgx table. Err: %s", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var page models.Page
		var strdata *string
		
		if err = rows.Scan(&page.PageID, &page.Type, &page.Sort, &page.CreatingImageLink, &strdata); err != nil {
			log.Printf("Error happened when scanning pages. Err: %s", err)
			return nil, err
		}

		if strdata != nil{
			page.Data = json.RawMessage(*strdata)
		} else {
			page.Data = nil
		}
		log.Println(page)
		page.UsedPhotoIDs = []uint{}
		photorows, err := storeDB.Query(ctx, "SELECT photos_id FROM page_has_photos WHERE pages_id = ($1);", page.PageID)
		if err != nil {
			log.Printf("Error happened when retrieving page photos from pgx table. Err: %s", err)
			return nil, err
		}
		defer photorows.Close()

		for photorows.Next() {
			var photoID uint
			if err = photorows.Scan(&photoID); err != nil {
				log.Println(err)
				return nil, err
			}
			page.UsedPhotoIDs = append(page.UsedPhotoIDs, photoID)

		}
		log.Println(page)
		pageslice = append(pageslice, page)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error happened when retrieving pages from pgx table. Err: %s", err)
		return nil, err
	}
	return pageslice, nil

}

// RetrieveTemplatePages function performs the operation of retrieving a photobook project from pgx database with a query.
func RetrieveTemplatePages(ctx context.Context, storeDB *pgxpool.Pool, projectID uint) ([]models.TemplatePage, error) {

	var pageslice []models.TemplatePage
	rows, err := storeDB.Query(ctx, "SELECT pages_id, type, sort, creating_image_link FROM pages WHERE projects_id = ($1) AND is_template = ($2) ORDER BY sort;", projectID, true)
	if err != nil {
		log.Printf("Error happened when retrieving pages from pgx table. Err: %s", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var page models.TemplatePage
		
		if err = rows.Scan(&page.PageID, &page.Type, &page.Sort, &page.CreatingImageLink); err != nil {
			log.Printf("Error happened when scanning pages. Err: %s", err)
			return nil, err
		}
		pageslice = append(pageslice, page)
	}
	
	return pageslice, nil

}


// RetrieveFrontPage function performs the operation of retrieving a photobook project front page from pgx database with a query.
func RetrieveFrontPage(ctx context.Context, storeDB *pgxpool.Pool, projectID uint, isTemplate bool) (models.SavePage, error) {

	var page models.SavePage
	var strData *string
	err := storeDB.QueryRow(ctx, "SELECT creating_image_link, preview_link, data FROM pages WHERE projects_id = ($1) AND is_template = ($2) AND type = ($3);", projectID, isTemplate, "front").Scan(&page.CreatingImageLink, &page.PreviewImageLink, &strData)
	if err != nil && err != pgx.ErrNoRows{
		log.Printf("Error happened when retrieving pages from pgx table. Err: %s", err)
		return page, err
	}
	if strData != nil{
		page.Data = json.RawMessage(*strData)
	} else {
		page.Data = nil
	}

	
	return page, nil

}

// SavePage function performs the operation of updating page data in pgx database with a query.
func SavePage(ctx context.Context, storeDB *pgxpool.Pool, page models.SavePage) (error) {

	strdata := string(page.Data)
	_, err = storeDB.Exec(ctx, "UPDATE pages SET preview_link = ($1), creating_image_link = ($2), data = ($3) WHERE pages_id = ($4);",
	page.PreviewImageLink,
	page.CreatingImageLink,
	strdata,
	page.PageID,
	)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("Error happened when updating user id for view project pgx table. Err: %s", err)
		return err
	}
	
	return nil

}

// AddProjectPage function performs the operation of adding a photobook project page to pgx database with a query.
func AddProjectPage(ctx context.Context, storeDB *pgxpool.Pool, projectID uint, sort uint, isTemplate bool) (models.OrderPage, error) {

	var newPage models.OrderPage
	newPage.Sort = sort

	rows, err := storeDB.Query(ctx, "SELECT pages_id, sort FROM pages WHERE projects_id = ($1) AND is_template =($2);", projectID, isTemplate)
	if err != nil {
		log.Printf("Error happened when retrieving pages from pgx table. Err: %s", err)
		return newPage, err
	}
	defer rows.Close()

	for rows.Next() {
		var sortNum uint
		var newsortNum uint
		var existingID uint
		if err = rows.Scan(&existingID, &sortNum); err != nil {
			log.Printf("Error happened when scanning pages sort. Err: %s", err)
			return newPage, err
		}
		if sortNum >= sort {
			newsortNum = sortNum + 1
			_, err = storeDB.Exec(ctx, "UPDATE pages SET sort = ($1) WHERE pages_id = ($2);",
			newsortNum,
			existingID,
			)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				log.Printf("Error happened when updating page sort in pgx table. Err: %s", err)
				return newPage, err
			}
		}
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error happened when retrieving pages from pgx table. Err: %s", err)
		return newPage, err
	}

	t := time.Now()
	err = storeDB.QueryRow(ctx, "INSERT INTO pages (sort, type, is_template, last_edited_at, projects_id) VALUES ($1, $2, $3, $4, $5) RETURNING pages_id;",
			sort,
			"page",
			isTemplate,
			t,
			projectID,
		).Scan(&newPage.PageID)
	if err != nil {
			log.Printf("Error happened when inserting a new page into pgx table. Err: %s", err)
			return newPage, err
	}

	var oldCount uint
	var newCount uint
	if !isTemplate {
		err = storeDB.QueryRow(ctx, "SELECT count_pages FROM projects WHERE projects_id = ($1);", projectID).Scan(&oldCount)
			if err != nil && err != pgx.ErrNoRows {
				log.Printf("Error happened when retrieving count pages data from db. Err: %s", err)
				return newPage, err
			}
		newCount = oldCount + 1
		_, err = storeDB.Exec(ctx, "UPDATE projects SET count_pages = ($1) WHERE projects_id = ($2);",
				newCount,
				projectID,
				)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
					log.Printf("Error happened when updating page sort in pgx table. Err: %s", err)
					return newPage, err
		}
	}
	return newPage, nil

}

// DuplicatePage function performs the operation of duplicating existing photobook project page to pgx database with a query.
func DuplicatePage(ctx context.Context, storeDB *pgxpool.Pool, duplicateID uint, pageID uint) error {

	var pageObj models.SavePage
	var strData sql.NullString

	err := storeDB.QueryRow(ctx, "SELECT preview_link, creating_image_link, data FROM pages WHERE pages_id = ($1);", duplicateID).Scan(&pageObj.PreviewImageLink, &pageObj.CreatingImageLink, &strData)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("Error happened when retrieving duplicate page from pgx table. Err: %s", err)
		return err
	}

	_, err = storeDB.Exec(ctx, "UPDATE pages SET preview_link = ($1), creating_image_link = ($2), data = ($3) WHERE pages_id = ($4);",
	pageObj.PreviewImageLink,
	pageObj.CreatingImageLink,
	strData.String,
	pageID,
	)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("Error happened when updating duplicate page in pgx table. Err: %s", err)
		return err
	}

	rows, err := storeDB.Query(ctx, "SELECT photos_id FROM page_has_photos WHERE pages_id = ($1);", duplicateID)
	if err != nil {
		log.Printf("Error happened when retrieving page_has_photos from pgx table. Err: %s", err)
		return err
	}
	defer rows.Close()
	t := time.Now()
	for rows.Next() {
		var photoID uint
		if err = rows.Scan(&photoID); err != nil {
			log.Printf("Error happened when scanning pages page_has_photos. Err: %s", err)
			return err
		}
		_, err = storeDB.Exec(ctx, "INSERT INTO page_has_photos (pages_id, photos_id, last_edited_at) VALUES ($1, $2, $3);",
			pageID,
			photoID,
			t,
		)
		if err != nil {
			log.Printf("Error happened when inserting a new entry into page_has_photos table. Err: %s", err)
			return err
		}
	}	

	return nil

}


// DeletePage function performs the operation of deleting page from pgx database with a query.
func DeletePage(ctx context.Context, storeDB *pgxpool.Pool, pageID uint, projectID uint, isTemplate bool) (error) {

	var oldsort uint
	err := storeDB.QueryRow(ctx, "SELECT sort FROM pages WHERE pages_id = ($1) and projects_id = ($2);", pageID, projectID).Scan(&oldsort)
	if err != nil {
		log.Printf("Error happened when retrieving sort number for the page to be removed from pgx table. Err: %s", err)
		return err
	}
	
	_, err = storeDB.Exec(ctx, "DELETE FROM pages WHERE pages_id=($1) and projects_id = ($2);",
		pageID,
		projectID,
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

	rows, err := storeDB.Query(ctx, "SELECT pages_id, sort FROM pages WHERE projects_id = ($1) AND is_template =($2);", projectID, isTemplate)
	if err != nil {
		log.Printf("Error happened when retrieving pages from pgx table. Err: %s", err)
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var sortNum uint
		var newsortNum uint
		var existingID uint
		if err = rows.Scan(&existingID, &sortNum); err != nil {
			log.Printf("Error happened when scanning pages sort. Err: %s", err)
			return err
		}
		if sortNum > oldsort {
			newsortNum = sortNum - 1
			_, err = storeDB.Exec(ctx, "UPDATE pages SET sort = ($1) WHERE pages_id = ($2);",
			newsortNum,
			existingID,
			)
			if err != nil {
				log.Printf("Error happened when updating page sort in pgx table. Err: %s", err)
				return err
			}
		}
	}
	var oldCount uint
	var newCount uint
	err = storeDB.QueryRow(ctx, "SELECT count_pages FROM projects WHERE projects_id = ($1);", projectID).Scan(&oldCount)
		if err != nil && err != pgx.ErrNoRows {
			log.Printf("Error happened when retrieving count pages data from db. Err: %s", err)
			return err
		}
	newCount = oldCount - 1 
	_, err = storeDB.Exec(ctx, "UPDATE projects SET count_pages = ($1) WHERE projects_id = ($2);",
			newCount,
			projectID,
			)
	if err != nil {
				log.Printf("Error happened when updating page sort in pgx table. Err: %s", err)
				return err
	}

	return nil
}

// ReorderPage function performs the operation of changing the sort number of page from pgx database with a query.
func ReorderPage(ctx context.Context, storeDB *pgxpool.Pool, pageID uint, projectID uint, sort uint) (error) {

	_, err = storeDB.Exec(ctx, "UPDATE pages SET sort = ($1) WHERE pages_id = ($2);",
	sort,
	pageID,
	)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("Error happened when updating page sort in pgx table. Err: %s", err)
		return err
	}
	
	return nil

}

// RetrieveTemplates function performs the operation of retrieving templates from pgx database with a query.
func RetrieveTemplates(ctx context.Context, storeDB *pgxpool.Pool, offset uint, limit uint, tcategory string, tsize string) (models.ResponseTemplates, error) {

	templateset := models.ResponseTemplates{}
	templateset.Templates = []models.Template{}

	rows, err := storeDB.Query(ctx, "SELECT templates_id FROM templates WHERE status = ($1) ORDER BY templates_id DESC LIMIT ($2) OFFSET ($3);", "PUBLISHED", limit, offset)
		if err != nil {
			log.Printf("Error happened when retrieving templates from pgx table. Err: %s", err)
			return templateset, err
		}
	defer rows.Close()

	if tcategory != "" {
		if tsize != "" {
			rows, err = storeDB.Query(ctx, "SELECT * FROM (SELECT templates_id FROM templates WHERE status = ($1) AND category = ($2) AND size =($3)) AS selectedT ORDER BY selectedT.templates_id DESC LIMIT ($4) OFFSET ($5);", "PUBLISHED", tcategory, tsize, limit, offset)
			if err != nil {
				log.Printf("Error happened when retrieving templates from pgx table. Err: %s", err)
				return templateset, err
			}
			defer rows.Close()
		} else {
			rows, err = storeDB.Query(ctx, "SELECT * FROM (SELECT templates_id FROM templates WHERE status = ($1) AND category = ($2)) AS selectedT ORDER BY selectedT.templates_id DESC LIMIT ($3) OFFSET ($4);", "PUBLISHED", tcategory, limit, offset)
			if err != nil {
				log.Printf("Error happened when retrieving templates from pgx table. Err: %s", err)
				return templateset, err
			}
			defer rows.Close()
		}
	} else if tsize != "" {
		rows, err = storeDB.Query(ctx, "SELECT * FROM (SELECT templates_id FROM templates WHERE status = ($1) AND size =($2)) AS selectedT ORDER BY selectedT.templates_id DESC LIMIT ($3) OFFSET ($4);", "PUBLISHED", tsize, limit, offset)
		if err != nil {
			log.Printf("Error happened when retrieving templates from pgx table. Err: %s", err)
			return templateset, err
		}
		defer rows.Close()
	}

	for rows.Next() {

		var templateObj models.Template
		var frontPage models.SavePage
		var tID uint
		if err = rows.Scan(&tID); err != nil {
			log.Printf("Error happened when scanning projects. Err: %s", err)
			return templateset, err
		}

		err = storeDB.QueryRow(ctx, "SELECT name, size FROM templates WHERE templates_id = ($1) ORDER BY last_edited_at DESC;", tID).Scan(&templateObj.Name, &templateObj.Size)
		if err != nil && err != pgx.ErrNoRows {
			log.Printf("Error happened when retrieving template data from db. Err: %s", err)
			return templateset, err
		}
		templateObj.TemplateID = tID
		
		frontPage, err = RetrieveFrontPage(ctx, storeDB, tID, true) 
		log.Printf("Retrieving templates pages")
		log.Println(frontPage)
		if err != nil {
			log.Printf("Error happened when retrieving template pages from db. Err: %s", err)
			return templateset, err
		}
		if frontPage.PreviewImageLink != nil {
			templateObj.PreviewImageLink = *frontPage.PreviewImageLink
		}
		if frontPage.CreatingImageLink != nil {
			templateObj.FrontPage.CreatingImageLink = frontPage.CreatingImageLink
		}
		templateObj.FrontPage.Data = frontPage.Data
		if tsize != "" {
			if tsize == templateObj.Size {
				templateset.Templates = append(templateset.Templates, templateObj)
			}
		} else {
			templateset.Templates = append(templateset.Templates, templateObj)
		}
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error happened when retrieving templates from pgx table. Err: %s", err)
		return templateset, err
	}
	defer rows.Close()
	var countAllString string
	err = storeDB.QueryRow(ctx, "SELECT COUNT(templates_id) FROM templates WHERE status = ($1);", "PUBLISHED").Scan(&countAllString)
		if err != nil && err != pgx.ErrNoRows{
			log.Printf("Error happened when counting templates in pgx table. Err: %s", err)
			return templateset, err
		}

	if tcategory != "" {
		if tsize != "" {
			err = storeDB.QueryRow(ctx, "SELECT COUNT(templates_id) FROM templates WHERE status = ($1) AND size = ($2) AND category = ($3);", "PUBLISHED", tsize, tcategory).Scan(&countAllString)
			if err != nil && err != pgx.ErrNoRows{
				log.Printf("Error happened when counting templates in pgx table. Err: %s", err)
				return templateset, err
			}
		} else {

			err = storeDB.QueryRow(ctx, "SELECT COUNT(templates_id) FROM templates WHERE status = ($1) AND category = ($2);", "PUBLISHED", tcategory).Scan(&countAllString)
			if err != nil && err != pgx.ErrNoRows{
				log.Printf("Error happened when counting templates in pgx table. Err: %s", err)
				return templateset, err
			}
		}
	} else if tsize != "" {
		err = storeDB.QueryRow(ctx, "SELECT COUNT(templates_id) FROM templates WHERE status = ($1) AND size = ($2);", "PUBLISHED", tsize).Scan(&countAllString)
		if err != nil && err != pgx.ErrNoRows{
			log.Printf("Error happened when counting templates in pgx table. Err: %s", err)
			return templateset, err
		}
	}
	
	templateset.CountAll, _ = strconv.Atoi(countAllString)
	return templateset, nil

}

// SavePagePhotos function performs the operation of saving edited photos related to existing project.
func SavePagePhotos(ctx context.Context, storeDB *pgxpool.Pool, pageID uint, photoIDS []uint) error {

	var err error

	_, err = storeDB.Exec(ctx, "DELETE FROM page_has_photos WHERE pages_id=($1);",
		pageID,
	)
	if err != nil && err != pgx.ErrNoRows {
		log.Printf("Error happened when deleting old page photos from pgx table. Err: %s", err)
		return err
	}

	tx, err := storeDB.Begin(ctx)
	if err != nil {
		log.Printf("Error happened when initiating pgx transaction. Err: %s", err)
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Prepare(ctx, "my-query-photo","INSERT INTO page_has_photos (pages_id, photos_id, last_edited_at) VALUES ($1, $2, $3);")
	if err != nil {
		log.Printf("Error happened when preparing pgx transaction context. Err: %s", err)
		return err
	}
	t := time.Now()
	for _, v := range photoIDS {
		if _, err = tx.Exec(ctx, "my-query-photo", pageID, v, t); err != nil {
			log.Printf("Error happened when declaring transaction. Err: %s", err)
			return err
		}
	}

	return tx.Commit(ctx)

}