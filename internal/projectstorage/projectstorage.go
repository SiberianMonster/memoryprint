// Storage package contains functions for storing photos and projects in a pgx database.
//
// Available at https://github.com/SiberianMonster/memoryprint/tree/development/internal/projectstorage
package projectstorage

import (
	"context"
	"database/sql"
	"errors"
	"github.com/SiberianMonster/memoryprint/internal/models"
	"github.com/SiberianMonster/memoryprint/internal/config"
	"log"
	"time"
	"strconv"
	"net/http"
	"strings"
	"encoding/json"
	"golang.org/x/exp/slices"
	"github.com/tebeka/selenium"

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
func DeleteImage(filename string) error {
	if filename != "" {

		newStr := `{"source":["`+filename+`"]}`
		var data = strings.NewReader(newStr)
		req, err := http.NewRequest("DELETE", "https://api.timeweb.cloud/api/v1/storages/buckets/1051/object-manager/remove", data)
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer " + config.TimewebToken)
		client := http.DefaultClient
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != 204 {
			log.Println(resp.StatusCode)
			err = errors.New("error deleting image from bucket")
			return err
		}
	}
	return nil
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
func CheckHardCover(ctx context.Context, storeDB *pgxpool.Pool, projectID uint) bool {
	var imageLink *string
	err := storeDB.QueryRow(ctx, "SELECT creating_image_link FROM pages WHERE projects_id = ($1) AND is_template = ($2) AND type = ($3);", projectID, "false", "front").Scan(&imageLink)
	if err != nil {
			log.Printf("Error happened when retrieving front page from pgx table. Err: %s", err)
			return false
	}
	if imageLink != nil {
		strimageLink := *imageLink
		if strimageLink != ""{
			return true
		}
		
	}
	return false
}
func CheckProjectPublished(ctx context.Context, storeDB *pgxpool.Pool, projectID uint) bool {
	var statusActive bool
	
	err = storeDB.QueryRow(ctx, "SELECT CASE WHEN EXISTS (SELECT * FROM projects WHERE status = ($1) AND projects_id = ($2)) THEN TRUE ELSE FALSE END;", "PUBLISHED", projectID).Scan(&statusActive)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("Error happened when checking if project active. Err: %s", err)
		return false
	}

	return statusActive
}


func CheckLeatherID(ctx context.Context, storeDB *pgxpool.Pool, leatherID uint) bool {
	var leatherExists bool
	
	err = storeDB.QueryRow(ctx, "SELECT CASE WHEN EXISTS (SELECT * FROM leather WHERE leather_id = ($1)) THEN TRUE ELSE FALSE END;", leatherID).Scan(&leatherExists)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("Error happened when checking leather id. Err: %s", err)
		return false
	}

	return leatherExists
}

func CheckProjectNotCompleted(ctx context.Context, storeDB *pgxpool.Pool, projectID uint) bool {
	var statusActive bool
	var orderID uint
	err = storeDB.QueryRow(ctx, "SELECT orders_id FROM orders_has_projects WHERE projects_id = ($1);", projectID).Scan(&orderID)
	if err != nil {
			log.Printf("Error happened when retrieving order id data from db. Err: %s", err)
			return false
	}

	err = storeDB.QueryRow(ctx, "SELECT CASE WHEN EXISTS (SELECT * FROM orders WHERE status = ($1) AND orders_id = ($2)) THEN TRUE ELSE FALSE END;", "AWAITING_PAYMENT", orderID).Scan(&statusActive)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("Error happened when checking if order is awaiting payment. Err: %s", err)
		return false
	}
	log.Println("Status completed")
	log.Println(statusActive)

	return statusActive
}

func CheckTemplate(ctx context.Context, storeDB *pgxpool.Pool, projectID uint) bool {
	var statusExists bool
	err = storeDB.QueryRow(ctx, "SELECT CASE WHEN EXISTS (SELECT * FROM templates WHERE templates_id = ($1)) THEN TRUE ELSE FALSE END;", projectID).Scan(&statusExists)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("Error happened when checking if template exists. Err: %s", err)
		return false
	}

	return statusExists
}

func CheckTemplatePublished(ctx context.Context, storeDB *pgxpool.Pool, projectID uint) bool {
	var statusExists bool
	err = storeDB.QueryRow(ctx, "SELECT CASE WHEN EXISTS (SELECT * FROM templates WHERE templates_id = ($1) and status = ($2)) THEN TRUE ELSE FALSE END;", projectID, "PUBLISHED").Scan(&statusExists)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("Error happened when checking if template is published. Err: %s", err)
		return false
	}

	return statusExists
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
	if sort > countPage - 1 || sort == 0 {
		log.Printf("Attempt to change cover or non-existing page. Err: %s", err)
		return false
	}
	return true
}


// CreateProject function performs the operation of creating a new photobook project in pgx database with a query.
func CreateProject(ctx context.Context, storeDB *pgxpool.Pool, userID uint, projectObj models.NewBlankProjectObj) (uint, error) {

	t := time.Now()
	var pID uint
	var email string
	var templateCategory string
	var projectSpine *string
	var creatingLinkSpine *string
	log.Println(projectObj)
	if projectObj.Name == "" {
		var countAllProjects string
		err = storeDB.QueryRow(ctx, "SELECT COUNT(projects_id) FROM projects;").Scan(&countAllProjects)
		if err != nil && err != pgx.ErrNoRows{
					log.Printf("Error happened when counting projects in pgx table. Err: %s", err)
					return 0, err
		}
		CountAll, _ := strconv.Atoi(countAllProjects)
		newCount := CountAll + 1
		newCountString := strconv.Itoa(newCount)
		projectName := "Проект_" + newCountString
		projectObj.Name = projectName
	}
	if projectObj.TemplateID != 0 {

		templateCategory, projectSpine, creatingLinkSpine, err = RetrieveTemplateData(ctx, storeDB, projectObj.TemplateID)
		if err != nil {
			log.Printf("Error happened when retrieving template category from db. Err: %s", err)
			return 0, err
		}
	}
	err := storeDB.QueryRow(ctx, "INSERT INTO projects (name, created_at, last_edited_at, status, size, variant, count_pages, users_id, last_editor, category, cover, paper, preview_spine_link, creating_spine_link, leather_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15) RETURNING projects_id;",
		projectObj.Name,
		t,
		t,
		"EDITED",
		projectObj.Size,
		projectObj.Variant,
		23,
		userID,
		userID,
		templateCategory,
		projectObj.Cover,
		projectObj.Surface,
		projectSpine,
		creatingLinkSpine,
		projectObj.LeatherID,
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
		var leatherID *uint
		leatherID = &projectObj.LeatherID
		
		templatePages, err = RetrieveProjectPages(ctx, storeDB, projectObj.TemplateID, true, leatherID)
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
			if projectObj.Cover == "LEATHERETTE" && page.Type != "page" {
				_, err = storeDB.Exec(ctx, "INSERT INTO pages (last_edited_at, sort, type, is_template, creating_image_link, data, projects_id) VALUES ($1, $2, $3, $4, $5, $6, $7);",
				t,
				page.Sort,
				page.Type,
				false,
				"",
				strdata,
				pID,
				)
				if err != nil {
					log.Printf("Error happened when inserting a new page from template into pgx table. Err: %s", err)
					return pID, err
				}
			} else {
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
			if projectObj.LeatherID != 0 && ptype != "page" {
				_, err = storeDB.Exec(ctx, "INSERT INTO pages (last_edited_at, sort, type, creating_image_link, is_template, projects_id) VALUES ($1, $2, $3, $4, $5, $6);",
					t,
					num,
					ptype,
					"",
					false,
					pID,
				)
				if err != nil {
					log.Printf("Error happened when inserting page into pgx table. Err: %s", err)
					return pID, err
				}
			} else {
				_, err = storeDB.Exec(ctx, "INSERT INTO pages (last_edited_at, sort, type, is_template, projects_id) VALUES ($1, $2, $3, $4, $5);",
					t,
					num,
					ptype,
					false,
					pID,
				)
				if err != nil {
					log.Printf("Error happened when inserting page into pgx table. Err: %s", err)
					return pID, err
				}
			}
		}
	}

	
	log.Printf("added new project to DB")
	log.Println(pID)
	return pID, nil
}

// DuplicateProject function performs the operation of duplicating existing project in pgx database with a query.
func DuplicateProject(ctx context.Context, storeDB *pgxpool.Pool, projectID uint, userID uint) (uint, error) {

	t := time.Now()
	var pID uint
	var projectObj models.ProjectObj
	var countPages uint
	var email string
	var leatherID *uint
	var strleatherID uint
	err := storeDB.QueryRow(ctx, "SELECT name, size, variant, count_pages, cover, paper, creating_spine_link, preview_spine_link, leather_id FROM projects WHERE projects_id = ($1);", projectID).Scan(&projectObj.Name, &projectObj.Size, &projectObj.Variant, &countPages, &projectObj.Cover, &projectObj.Surface, &projectObj.CreatingSpineLink, &projectObj.PreviewSpineLink, &leatherID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("Error happened when retrieving project from pgx table. Err: %s", err)
		return pID, err
	}
	projectObj.Name = "Копия_" + projectObj.Name 
	//_, err = storeDB.Exec(ctx, "SELECT setval('projects_id', MAX(projects_id)) FROM projects;")
	//if err != nil {
	//		log.Printf("Error happened when nulling id sequence. Err: %s", err)
	//		return pID, err
	//}
	if leatherID != nil {
		strleatherID = *leatherID
	}

	err = storeDB.QueryRow(ctx, "INSERT INTO projects (name, created_at, last_edited_at, status, size, variant, count_pages, cover, paper, creating_spine_link, preview_spine_link, users_id, leather_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13) RETURNING projects_id;",
		projectObj.Name,
		t,
		t,
		"EDITED",
		projectObj.Size,
		projectObj.Variant,
		countPages,
		projectObj.Cover,
		projectObj.Surface,
		projectObj.CreatingSpineLink,
		projectObj.PreviewSpineLink,
		userID,
		strleatherID,
	).Scan(&pID)
	if err != nil {
		log.Printf("Error happened when inserting a new project into pgx table. Err: %s", err)
		return pID, err
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
		log.Printf("Error happened when inserting project into users_edit_projects . Err: %s", err)
		return 0, err
	}


	var projectPages []models.Page

	projectPages, err = RetrieveProjectPages(ctx, storeDB, projectID, false, leatherID)
	if err != nil {
			log.Printf("Error happened when retrieving project pages from db. Err: %s", err)
			return 0, err
	}
	for _, page := range projectPages {
			var strdata *string
			var pageID uint
			err = storeDB.QueryRow(ctx, "SELECT data FROM pages WHERE pages_id = ($1);", page.PageID).Scan(&strdata)
			if err != nil {
					log.Printf("Error happened when retrieving project page data from db. Err: %s", err)
					return 0, err
			}
			err = storeDB.QueryRow(ctx, "INSERT INTO pages (last_edited_at, sort, type, is_template, creating_image_link, data, projects_id) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING pages_id;",
			t,
			page.Sort,
			page.Type,
			false,
			page.CreatingImageLink,
			strdata,
			pID,
			).Scan(&pageID)
			if err != nil {
				log.Printf("Error happened when inserting a new page from project into pgx table. Err: %s", err)
				return pID, err
			}
			for _, photoID := range page.UsedPhotoIDs {
				_, err = storeDB.Exec(ctx, "INSERT INTO page_has_photos (pages_id, photos_id, last_edited_at) VALUES ($1, $2, $3);",
					pageID,
					photoID,
					t,
				)
				if err != nil {
					log.Printf("Error happened when inserting a new entry into page_has_photos table. Err: %s", err)
					return pID, err
				}
			}
	}
		
	log.Printf("duplicated project to DB")
	return pID, nil
}

// CreateTemplate function performs the operation of creating a new photobook template in pgx database with a query.
func CreateTemplate(ctx context.Context, storeDB *pgxpool.Pool, name string, size string, category string, variant string) (uint, error) {

	t := time.Now()
	var tID uint
	err := storeDB.QueryRow(ctx, "INSERT INTO templates (name, created_at, last_edited_at, status, size, category, variant) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING templates_id;",
		name,
		t,
		t,
		"EDITED",
		size,
		category,
		variant,
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

// Duplicate Template function performs the operation of duplicating existing template in pgx database with a query.
func DuplicateTemplate(ctx context.Context, storeDB *pgxpool.Pool, templateID uint) (uint, error) {

	t := time.Now()
	var tID uint
	var projectObj models.SavedTemplateObj
	var tCategory string
	err := storeDB.QueryRow(ctx, "SELECT name, size, category, variant, creating_spine_link, preview_spine_link FROM templates WHERE templates_id = ($1);", templateID).Scan(&projectObj.Name, &projectObj.Size, &tCategory, &projectObj.Variant, &projectObj.CreatingSpineLink, &projectObj.PreviewSpineLink)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("Error happened when retrieving template from pgx table. Err: %s", err)
		return tID, err
	}
	projectObj.Name = "Копия_" + projectObj.Name 

	err = storeDB.QueryRow(ctx, "INSERT INTO templates (name, created_at, last_edited_at, status, size, category, variant, creating_spine_link, preview_spine_link) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING templates_id;",
		projectObj.Name,
		t,
		t,
		"EDITED",
		projectObj.Size,
		tCategory,
		projectObj.Variant,
		projectObj.CreatingSpineLink,
		projectObj.PreviewSpineLink,
	).Scan(&tID)
	if err != nil {
		log.Printf("Error happened when inserting a new template into pgx table. Err: %s", err)
		return tID, err
	}

	var templatePages []models.Page

	var leatherID *uint
	templatePages, err = RetrieveProjectPages(ctx, storeDB, templateID, true, leatherID)
	if err != nil {
			log.Printf("Error happened when retrieving template pages from db. Err: %s", err)
			return 0, err
	}
	for _, page := range templatePages {
			var strdata *string
			var pageID uint
			err = storeDB.QueryRow(ctx, "SELECT data FROM pages WHERE pages_id = ($1);", page.PageID).Scan(&strdata)
			if err != nil {
					log.Printf("Error happened when retrieving template page data from db. Err: %s", err)
					return 0, err
			}
			err = storeDB.QueryRow(ctx, "INSERT INTO pages (last_edited_at, sort, type, is_template, creating_image_link, data, projects_id) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING pages_id;",
			t,
			page.Sort,
			page.Type,
			true,
			page.CreatingImageLink,
			strdata,
			tID,
			).Scan(&pageID)
			if err != nil {
				log.Printf("Error happened when inserting a new page from project into pgx table. Err: %s", err)
				return tID, err
			}
			for _, photoID := range page.UsedPhotoIDs {
				_, err = storeDB.Exec(ctx, "INSERT INTO page_has_photos (pages_id, photos_id, last_edited_at) VALUES ($1, $2, $3);",
					pageID,
					photoID,
					t,
				)
				if err != nil {
					log.Printf("Error happened when inserting a new entry into page_has_photos table. Err: %s", err)
					return tID, err
				}
			}
			if err != nil {
				log.Printf("Error happened when inserting a new page from template into pgx table. Err: %s", err)
				return tID, err
			}
	}
		
	log.Printf("duplicated template to DB")
	return tID, nil
}

// UpdateTemplate function performs the operation of updating photobook template parameters in pgx database with a query.
func UpdateTemplate(ctx context.Context, storeDB *pgxpool.Pool, templateID uint, name string, category string) (uint, error) {

	_, err = storeDB.Exec(ctx, "UPDATE templates SET name = ($1), category = ($2) WHERE templates_id = ($3);",
		name,
		category,
		templateID,
	)
	if err != nil {
		log.Printf("Error happened when unpdating template into pgx table. Err: %s", err)
		return templateID, err
	}

	return templateID, nil
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

// UNPublishTemplate function performs the operation of rolling back template status.
func UnpublishTemplate(ctx context.Context, storeDB *pgxpool.Pool, templateID uint) error {

	var err error

	_, err = storeDB.Exec(ctx, "UPDATE templates SET status = ($1) WHERE templates_id = ($2);",
		"EDITED",
		templateID,
	)
	if err != nil {
		log.Printf("Error happened when unpublishing template. Err: %s", err)
		return err
	}
	return nil

}

// UNPublishProject function performs the operation of rolling back project status.
func UnpublishProject(ctx context.Context, storeDB *pgxpool.Pool, projectID uint) error {

	var err error

	_, err = storeDB.Exec(ctx, "UPDATE projects SET status = ($1) WHERE projects_id = ($2);",
		"EDITED",
		projectID,
	)
	if err != nil {
		log.Printf("Error happened when unpublishing project. Err: %s", err)
		return err
	}
	_, err = storeDB.Exec(ctx, "DELETE FROM orders_has_projects WHERE projects_id = ($1);", projectID)
	if err != nil {
				log.Printf("Error happened when unpublishing project. Err: %s", err)
				return err
	}
	return nil

}


// DeleteTemplate function performs the operation of deleting template from the db.
func DeleteTemplate(ctx context.Context, storeDB *pgxpool.Pool, tID uint) (error) {

	rows, err := storeDB.Query(ctx, "SELECT pages_id FROM pages WHERE projects_id = ($1) AND is_template = ($2) ORDER BY sort;", tID, true)
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
		err = DeletePage(ctx, storeDB, pageID, tID, true)

		if err != nil {
			log.Printf("Error happened when deleting template pages from pgx table. Err: %s", err)
			return err
		}
	}

	_, err = storeDB.Exec(ctx, "DELETE FROM templates WHERE templates_id=($1);",
		tID,
	)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("Error happened when deleting template from templates pgx table. Err: %s", err)
		return err
	}


	return nil

}

// RetrieveUserProjects function performs the operation of retrieving user photobook projects from pgx database with a query.
func RetrieveUserProjects(ctx context.Context, storeDB *pgxpool.Pool, userID uint, offset uint, limit uint) (models.ResponseProjects, error) {

	projectslice := []models.ResponseProject{}
	projectset := models.ResponseProjects{}
	var email string
	err = storeDB.QueryRow(ctx, "SELECT email FROM users WHERE users_id = ($1);", userID).Scan(&email)
	if err != nil {
			log.Printf("Error happened when retrieving user email data from db. Err: %s", err)
			return projectset, err
	}
	rows, err := storeDB.Query(ctx, "SELECT projects_id FROM users_edit_projects WHERE email = ($1) AND category = ($2) ORDER BY projects_id DESC LIMIT ($3) OFFSET ($4);",  email, "OWNER", limit, offset)

	if err != nil {
		log.Printf("Error happened when retrieving projects from pgx table. Err: %s", err)
		return projectset, err
	}
	defer rows.Close()

	for rows.Next() {

		var projectObj models.ResponseProject
		var pID uint
		var leatherID *uint
		if err = rows.Scan(&pID); err != nil {
			log.Printf("Error happened when scanning projects. Err: %s", err)
			return projectset, err
		}

		err = storeDB.QueryRow(ctx, "SELECT name, size, status, cover, leather_id FROM projects WHERE projects_id = ($1) ORDER BY last_edited_at DESC;", pID).Scan(&projectObj.Name, &projectObj.Size, &projectObj.Status, &projectObj.Cover, &leatherID)
		if err != nil && err != pgx.ErrNoRows {
			log.Printf("Error happened when retrieving project data from db. Err: %s", err)
			return projectset, err
		}
		projectObj.ProjectID = pID
		if projectObj.Status == "EDITED" || projectObj.Status == "PUBLISHED" {
			projectObj.Pages, err = RetrieveProjectPages(ctx, storeDB, pID, false, leatherID)
			if err != nil {
				log.Printf("Error happened when retrieving project pages from db. Err: %s", err)
				return projectset, err
			}
			projectslice = append(projectslice, projectObj)
		}
		
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error happened when retrieving projects from pgx table. Err: %s", err)
		return projectset, err
	}
	projectset.Projects = projectslice
	var countAllString string
	err = storeDB.QueryRow(ctx, "SELECT COUNT(projects.projects_id) FROM projects LEFT JOIN users_edit_projects ON projects.projects_id = users_edit_projects.projects_id WHERE users_edit_projects.email = ($1) AND users_edit_projects.category = ($2) AND projects.status <> ($3);",  email, "OWNER", "PRINTED").Scan(&countAllString)
	if err != nil && err != pgx.ErrNoRows{
				log.Printf("Error happened when counting projects in pgx table. Err: %s", err)
				return projectset, err
	}
		
	projectset.CountAll, _ = strconv.Atoi(countAllString)
	return projectset, nil

}

// RetrieveAdminProjects function performs the operation of retrieving photobook projects from pgx database with a query.
func RetrieveAdminProjects(ctx context.Context, storeDB *pgxpool.Pool, userID uint, offset uint, limit uint) (models.ResponseProjects, error) {

	projectslice := []models.ResponseProject{}
	projectset := models.ResponseProjects{}

	rows, err := storeDB.Query(ctx, "SELECT projects_id FROM projects WHERE status = ($1) ORDER BY projects_id DESC LIMIT ($2) OFFSET ($3);", "PUBLISHED", limit, offset)
	if err != nil {
		log.Printf("Error happened when retrieving projects from pgx table. Err: %s", err)
		return projectset, err
	}
	defer rows.Close()

	for rows.Next() {

		var projectObj models.ResponseProject
		var pID uint
		if err = rows.Scan(&pID); err != nil {
			log.Printf("Error happened when scanning projects. Err: %s", err)
			return projectset, err
		}

		err = storeDB.QueryRow(ctx, "SELECT name FROM projects WHERE projects_id = ($1) ORDER BY last_edited_at DESC;", pID).Scan(&projectObj.Name)
		if err != nil && err != pgx.ErrNoRows {
			log.Printf("Error happened when retrieving project data from db. Err: %s", err)
			return projectset, err
		}
		projectObj.ProjectID = pID
		projectslice = append(projectslice, projectObj)
		
		
	}
	projectset.Projects = projectslice

	if err = rows.Err(); err != nil {
		log.Printf("Error happened when retrieving projects from pgx table. Err: %s", err)
		return projectset, err
	}
	var countAllString string
	err = storeDB.QueryRow(ctx, "SELECT COUNT(projects_id) FROM projects WHERE status = ($1);", "PUBLISHED").Scan(&countAllString)
			if err != nil && err != pgx.ErrNoRows{
				log.Printf("Error happened when counting templates in pgx table. Err: %s", err)
				return projectset, err
		}

	
	projectset.CountAll, _ = strconv.Atoi(countAllString)
	return projectset, nil

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
func LoadProject(ctx context.Context, storeDB *pgxpool.Pool, pID uint) (models.ResponseProjectObj, error) {

	var projectObj models.ResponseProjectObj
	var updateTimeStorage time.Time
	var createTimeStorage time.Time
	err := storeDB.QueryRow(ctx, "SELECT name, size, variant, created_at, cover, last_edited_at, creating_spine_link, preview_spine_link FROM projects WHERE projects_id = ($1);", pID).Scan(&projectObj.Name, &projectObj.Size, &projectObj.Variant, &createTimeStorage, &projectObj.Cover, &updateTimeStorage, &projectObj.CreatingSpineLink, &projectObj.PreviewSpineLink)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("Error happened when retrieving project from pgx table. Err: %s", err)
		return projectObj, err
	}

	projectObj.LastEditedAt = updateTimeStorage.Unix()
	projectObj.CreatedAt = createTimeStorage.Unix()

	return projectObj, nil

}

// RetrieveTemplateData function performs the operation of retrieving template category by id from pgx database with a query.
func RetrieveTemplateData(ctx context.Context, storeDB *pgxpool.Pool, pID uint) (string, *string, *string, error) {

	var category string
	var spinelink *string
	var creatingspinelink *string
	err := storeDB.QueryRow(ctx, "SELECT category, preview_spine_link, creating_spine_link FROM templates WHERE templates_id = ($1);", pID).Scan(&category, &spinelink, &creatingspinelink)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("Error happened when retrieving category from pgx table. Err: %s", err)
		return category, spinelink, creatingspinelink, err
	}
	return category, spinelink, creatingspinelink, nil

}

// DeleteProject function performs the operation of deleting project from the db.
func DeleteProject(ctx context.Context, storeDB *pgxpool.Pool, pID uint) (error) {

	rows, err := storeDB.Query(ctx, "SELECT pages_id FROM pages WHERE projects_id = ($1) AND is_template = ($2) ORDER BY sort;", pID, false)
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
		err = DeletePage(ctx, storeDB, pageID, pID, true)

		if err != nil {
			log.Printf("Error happened when deleting project pages from pgx table. Err: %s", err)
			return err
		}
	}

	_, err = storeDB.Exec(ctx, "DELETE FROM users_edit_projects WHERE projects_id=($1);",
	pID,
	)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("Error happened when deleting project from users_edit_projects pgx table. Err: %s", err)
		return err
	}

	_, err = storeDB.Exec(ctx, "DELETE FROM projects WHERE projects_id=($1);",
	pID,
	)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("Error happened when deleting project from projects pgx table. Err: %s", err)
		return err
	}


	return nil

}

// LoadTemplate function performs the operation of retrieving template by id from pgx database with a query.
func LoadTemplate(ctx context.Context, storeDB *pgxpool.Pool, pID uint) (models.SavedTemplateObj, error) {

	var projectObj models.SavedTemplateObj
	var updateTimeStorage time.Time
	var createTimeStorage time.Time
	err := storeDB.QueryRow(ctx, "SELECT name, size, created_at, last_edited_at, creating_spine_link, preview_spine_link FROM templates WHERE templates_id = ($1) AND status = ($2);", pID, "PUBLISHED").Scan(&projectObj.Name, &projectObj.Size, &createTimeStorage, &updateTimeStorage, &projectObj.CreatingSpineLink, &projectObj.PreviewSpineLink)
	if err != nil {
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

// AdminLoadTemplate function performs the operation of retrieving template by id from pgx database with a query.
func AdminLoadTemplate(ctx context.Context, storeDB *pgxpool.Pool, pID uint) (models.SavedTemplateObj, error) {

	var projectObj models.SavedTemplateObj
	var updateTimeStorage time.Time
	var createTimeStorage time.Time
	err := storeDB.QueryRow(ctx, "SELECT name, size, variant, created_at, last_edited_at, creating_spine_link, preview_spine_link FROM templates WHERE templates_id = ($1);", pID).Scan(&projectObj.Name, &projectObj.Size, &projectObj.Variant, &createTimeStorage, &updateTimeStorage, &projectObj.CreatingSpineLink, &projectObj.PreviewSpineLink)
	if err != nil {
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
func RetrieveProjectPages(ctx context.Context, storeDB *pgxpool.Pool, projectID uint, isTemplate bool, leatherID *uint) ([]models.Page, error) {

	var pageslice []models.Page
	rows, err := storeDB.Query(ctx, "SELECT pages_id, type, sort, creating_image_link, preview_link, data FROM pages WHERE projects_id = ($1) AND is_template = ($2) ORDER BY sort;", projectID, isTemplate)
	if err != nil {
		log.Printf("Error happened when retrieving pages from pgx table. Err: %s", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var page models.Page
		var strdata *string
		
		if err = rows.Scan(&page.PageID, &page.Type, &page.Sort, &page.CreatingImageLink, &page.PreviewImageLink, &strdata); err != nil {
			log.Printf("Error happened when scanning pages. Err: %s", err)
			return nil, err
		}
		if leatherID != nil && page.Type != "page" {
			if *leatherID != 0 {

				err := storeDB.QueryRow(ctx, "SELECT colourlink FROM leather WHERE leather_id = ($1);", leatherID).Scan(&page.CreatingImageLink)
				page.PreviewImageLink = page.CreatingImageLink
				if err != nil {
					log.Printf("Error happened when retrieving colour image for leather cover from pgx table. Err: %s", err)
					return nil, err
				}
				
			}
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
	rows, err := storeDB.Query(ctx, "SELECT pages_id, type, sort, creating_image_link, preview_link, data FROM pages WHERE projects_id = ($1) AND is_template = ($2) ORDER BY sort;", projectID, true)
	if err != nil {
		log.Printf("Error happened when retrieving pages from pgx table. Err: %s", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var page models.TemplatePage
		var strdata *string
	
		
		if err = rows.Scan(&page.PageID, &page.Type, &page.Sort, &page.CreatingImageLink, &page.PreviewImageLink, &strdata); err != nil {
			log.Printf("Error happened when scanning pages. Err: %s", err)
			return nil, err
		}

		if err != nil {
			log.Printf("Error happened when setting empty value for creating_image_link. Err: %s", err)
			return nil, err
		}
		if strdata != nil{
			page.Data = json.RawMessage(*strdata)
		} else {
			page.Data = nil
		}
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
		pageslice = append(pageslice, page)
	}
	
	return pageslice, nil

}


// RetrieveFrontPage function performs the operation of retrieving a photobook project front page from pgx database with a query.
func RetrieveFrontPage(ctx context.Context, storeDB *pgxpool.Pool, projectID uint, isTemplate bool) (models.FrontPage, error) {

	var page models.FrontPage
	var cover string
	var coverImage *string
	var leatherID *uint
	err := storeDB.QueryRow(ctx, "SELECT creating_image_link FROM pages WHERE projects_id = ($1) AND is_template = ($2) AND type = ($3);", projectID, isTemplate, "front").Scan(&page.CreatingImageLink)
	if err != nil && err != pgx.ErrNoRows{
		log.Printf("Error happened when retrieving pages from pgx table. Err: %s", err)
		return page, err
	}
	if isTemplate == false {
		err := storeDB.QueryRow(ctx, "SELECT cover, leather_id FROM projects WHERE projects_id = ($1);", projectID).Scan(&cover, &leatherID)
		if err != nil && err != pgx.ErrNoRows{
			log.Printf("Error happened when retrieving cover from pgx table. Err: %s", err)
			return page, err
		}
		if cover == "LEATHERETTE" {
			if leatherID != nil {
				err := storeDB.QueryRow(ctx, "SELECT colourlink FROM leather WHERE leather_id = ($1);", leatherID).Scan(&coverImage)
				if err != nil && err != pgx.ErrNoRows{
					log.Printf("Error happened when retrieving leather image from pgx table. Err: %s", err)
					return page, err
				}
				page.CreatingImageLink = coverImage
			} else {
				err := storeDB.QueryRow(ctx, "SELECT colourlink FROM leather WHERE leather_id = ($1);", 0).Scan(&coverImage)
				if err != nil && err != pgx.ErrNoRows{
					log.Printf("Error happened when retrieving leather image from pgx table. Err: %s", err)
					return page, err
				}
				page.CreatingImageLink = coverImage
			}
			
		}
	}
	
	return page, nil

}

// RetrieveTemplateFrontPage function performs the operation of retrieving a photobook project front page from pgx database with a query.
func RetrieveTemplateFrontPage(ctx context.Context, storeDB *pgxpool.Pool, projectID uint, isTemplate bool) (models.TemplateFrontPage, error) {

	var page models.TemplateFrontPage
	var strdata *string
		
	err := storeDB.QueryRow(ctx, "SELECT creating_image_link, data FROM pages WHERE projects_id = ($1) AND is_template = ($2) AND type = ($3);", projectID, isTemplate, "front").Scan(&page.CreatingImageLink, &strdata)
	if err != nil && err != pgx.ErrNoRows{
		log.Printf("Error happened when retrieving pages from pgx table. Err: %s", err)
		return page, err
	}
	if strdata != nil{
		page.Data = json.RawMessage(*strdata)
	} else {
		page.Data = nil
	}
	
	
	return page, nil

}

// SavePage function performs the operation of updating page data in pgx database with a query.
func SavePage(ctx context.Context, storeDB *pgxpool.Pool, page models.SavePage) (error) {

	strdata := string(page.Data)
	var oldImage string
	var imageHolder *string
	err := storeDB.QueryRow(ctx, "SELECT creating_image_link FROM pages WHERE pages_id = ($1);", page.PageID).Scan(&imageHolder)
	if err != nil && err != pgx.ErrNoRows{
		log.Printf("Error happened when retrieving old image from pgx table. Err: %s", err)
		return err
	}
	if imageHolder != nil {
		oldImage = *imageHolder
		err = DeleteImage(oldImage) 
		if err != nil {
			log.Printf("Error happened when deleting image from bucket. Err: %s", err)
		}
	}
	
	
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
	var oldImage string
	var imageHolder *string
	err := storeDB.QueryRow(ctx, "SELECT sort, creating_image_link FROM pages WHERE pages_id = ($1) and projects_id = ($2);", pageID, projectID).Scan(&oldsort, &imageHolder)
	if err != nil {
		log.Printf("Error happened when retrieving sort number for the page to be removed from pgx table. Err: %s", err)
		return err
	}
	if imageHolder != nil {
		oldImage = *imageHolder
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
	if !isTemplate {
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
	}
	err = DeleteImage(oldImage) 
	if err != nil {
		log.Printf("Error happened when deleting image from bucket. Err: %s", err)
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
func RetrieveTemplates(ctx context.Context, storeDB *pgxpool.Pool, offset uint, limit uint, tcategory string, tsize string, tvariant string, tstatus string) (models.ResponseTemplates, error) {

	templateset := models.ResponseTemplates{}
	templateset.Templates = []models.Template{}

	if tstatus == "PUBLISHED" || tstatus == "EDITED" {

		log.Println(tvariant)
		rows, err := storeDB.Query(ctx, "SELECT templates_id FROM templates WHERE status = ($1) AND variant = ($2) ORDER BY templates_id DESC LIMIT ($3) OFFSET ($4);", tstatus, tvariant, limit, offset)
		if err != nil && !errors.Is(err, pgx.ErrNoRows){
			log.Printf("Error happened when retrieving templates from pgx table. Err: %s", err)
			return templateset, err
		}
		defer rows.Close()

		if tcategory != "" {
			if tsize != "" {
				rows, err = storeDB.Query(ctx, "SELECT * FROM (SELECT templates_id FROM templates WHERE status = ($1) AND category = ($2) AND size =($3) AND variant = ($4)) AS selectedT ORDER BY selectedT.templates_id DESC LIMIT ($5) OFFSET ($6);", tstatus, tcategory, tsize, tvariant, limit, offset)
				if err != nil && !errors.Is(err, pgx.ErrNoRows){
					log.Printf("Error happened when retrieving templates from pgx table. Err: %s", err)
					return templateset, err
				}
				defer rows.Close()
			} else {
				rows, err = storeDB.Query(ctx, "SELECT * FROM (SELECT templates_id FROM templates WHERE status = ($1) AND category = ($2) AND variant = ($3)) AS selectedT ORDER BY selectedT.templates_id DESC LIMIT ($4) OFFSET ($5);", tstatus, tcategory, tvariant, limit, offset)
				if err != nil && !errors.Is(err, pgx.ErrNoRows){
					log.Printf("Error happened when retrieving templates from pgx table. Err: %s", err)
					return templateset, err
				}
				defer rows.Close()
			}
		} else if tsize != "" {
			rows, err = storeDB.Query(ctx, "SELECT * FROM (SELECT templates_id FROM templates WHERE status = ($1) AND size =($2) AND variant = ($3)) AS selectedT ORDER BY selectedT.templates_id DESC LIMIT ($4) OFFSET ($5);", tstatus, tsize, tvariant, limit, offset)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				log.Printf("Error happened when retrieving templates from pgx table. Err: %s", err)
				return templateset, err
			}
			defer rows.Close()
		}

		for rows.Next() {

			var templateObj models.Template
			var frontPage models.TemplateFrontPage
			var tID uint
			if err = rows.Scan(&tID); err != nil {
				log.Printf("Error happened when scanning projects. Err: %s", err)
				return templateset, err
			}

			err = storeDB.QueryRow(ctx, "SELECT name, size, variant FROM templates WHERE templates_id = ($1) ORDER BY last_edited_at DESC;", tID).Scan(&templateObj.Name, &templateObj.Size,  &templateObj.Variant)
			if err != nil && err != pgx.ErrNoRows {
				log.Printf("Error happened when retrieving template data from db. Err: %s", err)
				return templateset, err
			}
			templateObj.TemplateID = tID
			
			frontPage, err = RetrieveTemplateFrontPage(ctx, storeDB, tID, true) 
			log.Printf("Retrieving templates pages")
			log.Println(frontPage)
			if err != nil {
				log.Printf("Error happened when retrieving template pages from db. Err: %s", err)
				return templateset, err
			}
			
			templateObj.FrontPage = frontPage
			templateObj.Status = tstatus
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
		err = storeDB.QueryRow(ctx, "SELECT COUNT(templates_id) FROM templates WHERE status = ($1);", tstatus).Scan(&countAllString)
			if err != nil && err != pgx.ErrNoRows{
				log.Printf("Error happened when counting templates in pgx table. Err: %s", err)
				return templateset, err
			}

		if tcategory != "" {
			if tsize != "" {
				err = storeDB.QueryRow(ctx, "SELECT COUNT(templates_id) FROM templates WHERE status = ($1) AND size = ($2) AND category = ($3) AND variant = ($4);", tstatus, tsize, tcategory, tvariant).Scan(&countAllString)
				if err != nil && err != pgx.ErrNoRows{
					log.Printf("Error happened when counting templates in pgx table. Err: %s", err)
					return templateset, err
				}
			} else {

				err = storeDB.QueryRow(ctx, "SELECT COUNT(templates_id) FROM templates WHERE status = ($1) AND category = ($2) AND variant = ($3);", tstatus, tcategory, tvariant).Scan(&countAllString)
				if err != nil && err != pgx.ErrNoRows{
					log.Printf("Error happened when counting templates in pgx table. Err: %s", err)
					return templateset, err
				}
			}
		} else if tsize != "" {
			err = storeDB.QueryRow(ctx, "SELECT COUNT(templates_id) FROM templates WHERE status = ($1) AND size = ($2) AND variant = ($3);", tstatus, tsize, tvariant).Scan(&countAllString)
			if err != nil && err != pgx.ErrNoRows{
				log.Printf("Error happened when counting templates in pgx table. Err: %s", err)
				return templateset, err
			}
		}
		
		templateset.CountAll, _ = strconv.Atoi(countAllString)
	
	}
	return templateset, nil

}

// RetrieveAdminTemplates function performs the operation of retrieving templates from pgx database with a query.
func RetrieveAdminTemplates(ctx context.Context, storeDB *pgxpool.Pool, offset uint, limit uint, tcategory string, tsize string, tstatus string) (models.ResponseTemplates, error) {

	templateset := models.ResponseTemplates{}
	templateset.Templates = []models.Template{}

	if tstatus == "PUBLISHED" || tstatus == "EDITED" {

		rows, err := storeDB.Query(ctx, "SELECT templates_id FROM templates WHERE status = ($1) ORDER BY templates_id DESC LIMIT ($2) OFFSET ($3);", tstatus, limit, offset)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			log.Printf("Error happened when retrieving templates from pgx table. Err: %s", err)
			return templateset, err
		}
		defer rows.Close()

		if tcategory != "" {
			if tsize != "" {
				rows, err = storeDB.Query(ctx, "SELECT * FROM (SELECT templates_id FROM templates WHERE status = ($1) AND category = ($2) AND size =($3)) AS selectedT ORDER BY selectedT.templates_id DESC LIMIT ($4) OFFSET ($5);", tstatus, tcategory, tsize, limit, offset)
				if err != nil && !errors.Is(err, pgx.ErrNoRows) {
					log.Printf("Error happened when retrieving templates from pgx table. Err: %s", err)
					return templateset, err
				}
				defer rows.Close()
			} else {
				rows, err = storeDB.Query(ctx, "SELECT * FROM (SELECT templates_id FROM templates WHERE status = ($1) AND category = ($2)) AS selectedT ORDER BY selectedT.templates_id DESC LIMIT ($3) OFFSET ($4);", tstatus, tcategory, limit, offset)
				if err != nil && !errors.Is(err, pgx.ErrNoRows) {
					log.Printf("Error happened when retrieving templates from pgx table. Err: %s", err)
					return templateset, err
				}
				defer rows.Close()
			}
		} else if tsize != "" {
			rows, err = storeDB.Query(ctx, "SELECT * FROM (SELECT templates_id FROM templates WHERE status = ($1) AND size =($2)) AS selectedT ORDER BY selectedT.templates_id DESC LIMIT ($3) OFFSET ($4);", tstatus, tsize, limit, offset)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				log.Printf("Error happened when retrieving templates from pgx table. Err: %s", err)
				return templateset, err
			}
			defer rows.Close()
		}
	
		for rows.Next() {

			var templateObj models.Template
			var frontPage models.TemplateFrontPage
			var tID uint
			if err = rows.Scan(&tID); err != nil {
				log.Printf("Error happened when scanning projects. Err: %s", err)
				return templateset, err
			}

			err = storeDB.QueryRow(ctx, "SELECT name, size, status, variant FROM templates WHERE templates_id = ($1) ORDER BY last_edited_at DESC;", tID).Scan(&templateObj.Name, &templateObj.Size, &templateObj.Status, &templateObj.Variant)
			if err != nil && err != pgx.ErrNoRows {
				log.Printf("Error happened when retrieving template data from db. Err: %s", err)
				return templateset, err
			}
			templateObj.TemplateID = tID
			
			frontPage, err = RetrieveTemplateFrontPage(ctx, storeDB, tID, true) 
			log.Printf("Retrieving templates pages")
			log.Println(frontPage)
			if err != nil {
				log.Printf("Error happened when retrieving template pages from db. Err: %s", err)
				return templateset, err
			}
			
			templateObj.FrontPage = frontPage
			templateset.Templates = append(templateset.Templates, templateObj)
			
		}

		if err = rows.Err(); err != nil {
			log.Printf("Error happened when retrieving templates from pgx table. Err: %s", err)
			return templateset, err
		}
		defer rows.Close()
		
		
	} else {
		rows, err := storeDB.Query(ctx, "SELECT templates_id FROM templates ORDER BY templates_id DESC LIMIT ($1) OFFSET ($2);", limit, offset)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			log.Printf("Error happened when retrieving templates from pgx table. Err: %s", err)
			return templateset, err
		}
		defer rows.Close()

		if tcategory != "" {
			if tsize != "" {
				rows, err = storeDB.Query(ctx, "SELECT * FROM (SELECT templates_id FROM templates WHERE category = ($1) AND size =($2)) AS selectedT ORDER BY selectedT.templates_id DESC LIMIT ($3) OFFSET ($4);", tcategory, tsize, limit, offset)
				if err != nil && !errors.Is(err, pgx.ErrNoRows) {
					log.Printf("Error happened when retrieving templates from pgx table. Err: %s", err)
					return templateset, err
				}
				defer rows.Close()
			} else {
				rows, err = storeDB.Query(ctx, "SELECT * FROM (SELECT templates_id FROM templates WHERE category = ($1)) AS selectedT ORDER BY selectedT.templates_id DESC LIMIT ($2) OFFSET ($3);", tcategory, limit, offset)
				if err != nil && !errors.Is(err, pgx.ErrNoRows) {
					log.Printf("Error happened when retrieving templates from pgx table. Err: %s", err)
					return templateset, err
				}
				defer rows.Close()
			}
		} else if tsize != "" {
			rows, err = storeDB.Query(ctx, "SELECT * FROM (SELECT templates_id FROM templates WHERE size =($1)) AS selectedT ORDER BY selectedT.templates_id DESC LIMIT ($2) OFFSET ($3);", tsize, limit, offset)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				log.Printf("Error happened when retrieving templates from pgx table. Err: %s", err)
				return templateset, err
			}
			defer rows.Close()
		}
		
		for rows.Next() {

			var templateObj models.Template
			var frontPage models.TemplateFrontPage
			var tID uint
			if err = rows.Scan(&tID); err != nil {
				log.Printf("Error happened when scanning projects. Err: %s", err)
				return templateset, err
			}

			err = storeDB.QueryRow(ctx, "SELECT name, size, status, variant FROM templates WHERE templates_id = ($1) ORDER BY last_edited_at DESC;", tID).Scan(&templateObj.Name, &templateObj.Size, &templateObj.Status, &templateObj.Variant)
			if err != nil && err != pgx.ErrNoRows {
				log.Printf("Error happened when retrieving template data from db. Err: %s", err)
				return templateset, err
			}
			templateObj.TemplateID = tID
			
			frontPage, err = RetrieveTemplateFrontPage(ctx, storeDB, tID, true) 
			log.Printf("Retrieving templates pages")
			log.Println(frontPage)
			if err != nil {
				log.Printf("Error happened when retrieving template pages from db. Err: %s", err)
				return templateset, err
			}

			templateObj.FrontPage = frontPage
			templateset.Templates = append(templateset.Templates, templateObj)
			
		}

		if err = rows.Err(); err != nil {
			log.Printf("Error happened when retrieving templates from pgx table. Err: %s", err)
			return templateset, err
		}
		defer rows.Close()
		
		
	}
	var countAllString string
	err = storeDB.QueryRow(ctx, "SELECT COUNT(templates_id) FROM templates;").Scan(&countAllString)
			if err != nil && err != pgx.ErrNoRows{
				log.Printf("Error happened when counting templates in pgx table. Err: %s", err)
				return templateset, err
		}
	if tstatus != "" {
		err = storeDB.QueryRow(ctx, "SELECT COUNT(templates_id) FROM templates WHERE status = ($1);", tstatus).Scan(&countAllString)
		if err != nil && err != pgx.ErrNoRows{
				log.Printf("Error happened when counting templates in pgx table. Err: %s", err)
				return templateset, err
		}

		if tcategory != "" {
				if tsize != "" {
					err = storeDB.QueryRow(ctx, "SELECT COUNT(templates_id) FROM templates WHERE status = ($1) AND size = ($2) AND category = ($3);", tstatus, tsize, tcategory).Scan(&countAllString)
					if err != nil && err != pgx.ErrNoRows{
						log.Printf("Error happened when counting templates in pgx table. Err: %s", err)
						return templateset, err
					}
		} else {

					err = storeDB.QueryRow(ctx, "SELECT COUNT(templates_id) FROM templates WHERE status = ($1) AND category = ($2);", tstatus, tcategory).Scan(&countAllString)
					if err != nil && err != pgx.ErrNoRows{
						log.Printf("Error happened when counting templates in pgx table. Err: %s", err)
						return templateset, err
					}
				}
		} else if tsize != "" {
				err = storeDB.QueryRow(ctx, "SELECT COUNT(templates_id) FROM templates WHERE status = ($1) AND size = ($2);", tstatus, tsize).Scan(&countAllString)
				if err != nil && err != pgx.ErrNoRows{
					log.Printf("Error happened when counting templates in pgx table. Err: %s", err)
					return templateset, err
				}
			}
	} else {
		if tcategory != "" {
			if tsize != "" {
				err = storeDB.QueryRow(ctx, "SELECT COUNT(templates_id) FROM templates WHERE size = ($1) AND category = ($2);", tsize, tcategory).Scan(&countAllString)
				if err != nil && err != pgx.ErrNoRows{
					log.Printf("Error happened when counting templates in pgx table. Err: %s", err)
					return templateset, err
				}
		} else {

					err = storeDB.QueryRow(ctx, "SELECT COUNT(templates_id) FROM templates WHERE category = ($1);", tcategory).Scan(&countAllString)
					if err != nil && err != pgx.ErrNoRows{
						log.Printf("Error happened when counting templates in pgx table. Err: %s", err)
						return templateset, err
					}
				}
		} else if tsize != "" {
				err = storeDB.QueryRow(ctx, "SELECT COUNT(templates_id) FROM templates WHERE size = ($1);", tsize).Scan(&countAllString)
				if err != nil && err != pgx.ErrNoRows{
					log.Printf("Error happened when counting templates in pgx table. Err: %s", err)
					return templateset, err
				}
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

// AddViewer function performs the operation of adding project viewer in pgx database with a query.
func AddViewer(ctx context.Context, storeDB *pgxpool.Pool, projectID uint, email string) (error) {

	_, err = storeDB.Exec(ctx, "INSERT INTO users_edit_projects (projects_id, email, category) VALUES ($1, $2, $3);",
	projectID,
		email,
		"VIEWER",
	)
	if err != nil {
		log.Printf("Error happened when inserting project into users_edit_projects . Err: %s", err)
		return err
	}

	log.Printf("added project viewer")
	return nil
}

// UpdateCover function performs the operation of updating project cover to the db.
func UpdateCover(ctx context.Context, storeDB *pgxpool.Pool, pID uint, newC models.UpdateCover) (error) {

	_, err = storeDB.Exec(ctx, "UPDATE projects SET cover = ($1), leather_id = ($2) WHERE projects_id = ($3);",
		newC.Cover,
		newC.LeatherID,
		pID,
	)
	if err != nil {
		log.Printf("Error happened when updating project cover into pgx table. Err: %s", err)
		return err
	}
	
	return nil

}

// UpdateSurface function performs the operation of updating project surface to the db.
func UpdateSurface(ctx context.Context, storeDB *pgxpool.Pool, pID uint, newS models.UpdateSurface) (error) {

	_, err = storeDB.Exec(ctx, "UPDATE projects SET paper = ($1) WHERE projects_id = ($2);",
		newS.Surface,
		pID,
	)
	if err != nil {
		log.Printf("Error happened when updating project surface into pgx table. Err: %s", err)
		return err
	}
	
	return nil

}

// SaveSpine function performs the operation of updating project spine to the db.
func SaveSpine(ctx context.Context, storeDB *pgxpool.Pool, newS models.SavedSpine, pID uint) (error) {

	_, err = storeDB.Exec(ctx, "UPDATE projects SET creating_spine_link = ($1), preview_spine_link = ($2) WHERE projects_id = ($3);",
		newS.CreatingSpineLink,
		newS.PreviewSpineLink,
		pID,
	)
	if err != nil {
		log.Printf("Error happened when updating project spine into pgx table. Err: %s", err)
		return err
	}
	
	return nil

}

// SaveTemplateSpine function performs the operation of updating template spine to the db.
func SaveTemplateSpine(ctx context.Context, storeDB *pgxpool.Pool, newS models.SavedSpine, pID uint) (error) {

	_, err = storeDB.Exec(ctx, "UPDATE templates SET creating_spine_link = ($1), preview_spine_link = ($2) WHERE templates_id = ($3);",
		newS.CreatingSpineLink,
		newS.PreviewSpineLink,
		pID,
	)
	if err != nil {
		log.Printf("Error happened when updating project spine into pgx table. Err: %s", err)
		return err
	}
	
	return nil

}

// LoadPromocodeTemplates function performs the operation of retrieving 3 first templates from pgx database with a query.
func LoadPromocodeTemplates(ctx context.Context, storeDB *pgxpool.Pool, tcategory string) (models.ResponseTemplates, error) {

	templates := []models.Template{}
	templateset := models.ResponseTemplates{}
	tstatus := "PUBLISHED"
	rows, err := storeDB.Query(ctx, "SELECT * FROM (SELECT templates_id FROM templates WHERE status = ($1)) AS selectedT ORDER BY selectedT.templates_id DESC LIMIT ($2);", tstatus, 3)
	if err != nil && err != pgx.ErrNoRows {
					log.Printf("Error happened when retrieving templates from pgx table. Err: %s", err)
					return templateset, err
	}
	defer rows.Close()

	if tcategory != "" {
				rows, err = storeDB.Query(ctx, "SELECT * FROM (SELECT templates_id FROM templates WHERE status = ($1) AND category = ($2)) AS selectedT ORDER BY selectedT.templates_id DESC LIMIT ($3) ;", tstatus, tcategory, 3)
				if err != nil && err != pgx.ErrNoRows {
					log.Printf("Error happened when retrieving templates from pgx table. Err: %s", err)
					return templateset, err
				}
				defer rows.Close()
	} 

	for rows.Next() {

			var templateObj models.Template
			var frontPage models.TemplateFrontPage
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
			
			frontPage, err = RetrieveTemplateFrontPage(ctx, storeDB, tID, true) 
			log.Printf("Retrieving templates pages")
			log.Println(frontPage)
			if err != nil {
				log.Printf("Error happened when retrieving template pages from db. Err: %s", err)
				return templateset, err
			}
			
			templateObj.FrontPage = frontPage
			templateObj.Status = tstatus
			templates = append(templates, templateObj)
			
		}

		if err = rows.Err(); err != nil {
			log.Printf("Error happened when retrieving templates from pgx table. Err: %s", err)
			return templateset, err
		}
	defer rows.Close()
	templateset.Templates = templates
	templateset.CountAll = 3
		
	
	return templateset, nil

}

// RetrieveProjectImages function performs the operation of retrieving images of a published photobook project from pgx database with a query.
func RetrieveProjectImages(ctx context.Context, storeDB *pgxpool.Pool, projectID uint) ([]string, error) {

	var images []string
	rows, err := storeDB.Query(ctx, "SELECT creating_image_link FROM pages WHERE projects_id = ($1) AND is_template = ($2) ORDER BY sort;", projectID, false)
	if err != nil {
		log.Printf("Error happened when retrieving pages from pgx table. Err: %s", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var image string
		
		if err = rows.Scan(&image); err != nil {
			log.Printf("Error happened when scanning pages. Err: %s", err)
			return nil, err
		}
		
		images = append(images, image)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error happened when retrieving pages from pgx table. Err: %s", err)
		return nil, err
	}
	return images, nil

}

// GenerateImages function checks if a paid project has images for all pages, and if no, triggers their generation
func GenerateImages(ctx context.Context, storeDB *pgxpool.Pool, orderObj models.PaidOrderObj, driver selenium.WebDriver) ([]string, error) {

	images, err := RetrieveProjectImages(ctx, storeDB, orderObj.OrdersID)
	if err != nil {
		log.Printf("Error happened when retrieving images from pgx table. Err: %s", err)
		return images, err
	}
	if slices.Contains(images, "") {
		log.Println("Need to generate images")
		log.Println(orderObj.OrdersID)
		url :=  "https://front.memoryprint.dev.startup-it.ru/preview/generate/" + strconv.Itoa(int(orderObj.OrdersID))
		err = driver.Get(url)
		if err != nil {
			log.Printf("Error happened when generating images for paid project. Err: %s", err)
		}
	}
	
	return images, nil

}