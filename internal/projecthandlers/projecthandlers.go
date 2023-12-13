// Handlers package contains endpoints handlers for the Photo Book Editor module.
//
// https://github.com/SiberianMonster/memoryprint/tree/development/internal/projecthandlers
package projecthandlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"io/ioutil"

	"github.com/SiberianMonster/memoryprint/internal/config"
	"github.com/SiberianMonster/memoryprint/internal/emailutils"
	"github.com/SiberianMonster/memoryprint/internal/userstorage"
	"github.com/SiberianMonster/memoryprint/internal/models"
	"github.com/SiberianMonster/memoryprint/internal/projectstorage"
	"github.com/SiberianMonster/memoryprint/internal/objectsstorage"
	"github.com/SiberianMonster/memoryprint/internal/handlersfunc"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

var err error
var resp map[string]string

func AddProjectEditor(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]string)
	var ProjectObj models.ProjectEditorObj

	err := json.NewDecoder(r.Body).Decode(&ProjectObj)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, resp, err)
		return
	}

	defer r.Body.Close()
	log.Printf("Adding new editor for project %d", ProjectObj.ProjectID)

	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	userID := handlersfunc.UserIDContextReader(r)

	userCheck := projectstorage.CheckUserHasProject(ctx, config.DB, userID, ProjectObj.ProjectID)

	if !userCheck {
		handlersfunc.HandlePermissionError(rw, resp)
		return
	}

	_, err = projectstorage.AddProjectEditor(ctx, config.DB, ProjectObj.Email, ProjectObj.ProjectID, models.EditorCategory)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
		return
	}

	
	rw.WriteHeader(http.StatusOK)
	resp["status"] = "project editor added successfully"
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)

}

func AddProjectViewer(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]string)
	var ProjectObj models.ProjectEditorObj
	var dbUser models.User

	err := json.NewDecoder(r.Body).Decode(&ProjectObj)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, resp, err)
		return
	}

	defer r.Body.Close()
	log.Printf("Adding new viewer for project %d", ProjectObj.ProjectID)

	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	userID := handlersfunc.UserIDContextReader(r)

	userCheck := projectstorage.CheckUserHasProject(ctx, config.DB, userID, ProjectObj.ProjectID)

	if !userCheck {
		handlersfunc.HandlePermissionError(rw, resp)
		return
	}
	dbUser, err = userstorage.GetUserData(ctx, config.DB, userID)
	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
		return
	}

	_, err = projectstorage.AddProjectEditor(ctx, config.DB, ProjectObj.Email, ProjectObj.ProjectID, models.EditorCategory)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
		return
	}

	var user models.User

	user.Email = ProjectObj.Email

	if userstorage.CheckUser(ctx, config.DB, user) {
		// Send notification mail to existing user
		from := "support@memoryprint.ru"
		to := []string{ProjectObj.Email}
		subject := "MemoryPrint Invitation"
		mailType := emailutils.MailViewerInvitationExist
		mailData := &emailutils.MailData{
			OwnerName: dbUser.Username,
			OwnerEmail: dbUser.Email,
		}

		ms := &emailutils.SGMailService{config.YandexApiKey, config.MailVerifCodeExpiration, config.PassResetCodeExpiration, config.MailVerifTemplateID, config.PassResetTemplateID, config.DesignerOrderTemplateID, config.ViewerInvitationNewTemplateID, config.ViewerInvitationExistTemplateID}
		mailReq := emailutils.NewMail(from, to, subject, mailType, mailData)
		err = emailutils.SendMail(mailReq, ms)
		if err != nil {
			log.Printf("unable to send mail", "error", err)
			handlersfunc.HandleMailSendError(rw, resp)
			return
		}
	} else {
		user.Password = emailutils.GenerateRandomString(8)
		user.Category = models.CustomerCategory
		userID, err = userstorage.CreateUser(ctx, config.DB, user)

		// Send notification mail to new user
		from := "support@memoryprint.ru"
		to := []string{ProjectObj.Email}
		subject := "MemoryPrint Invitation"
		mailType := emailutils.MailViewerInvitationNew
		mailData := &emailutils.MailData{
			OwnerName: dbUser.Username,
			OwnerEmail: dbUser.Email,
			UserEmail: ProjectObj.Email,
			TempPass: user.Password,
		}

		ms := &emailutils.SGMailService{config.YandexApiKey, config.MailVerifCodeExpiration, config.PassResetCodeExpiration, config.MailVerifTemplateID, config.PassResetTemplateID, config.DesignerOrderTemplateID, config.ViewerInvitationNewTemplateID, config.ViewerInvitationExistTemplateID}
		mailReq := emailutils.NewMail(from, to, subject, mailType, mailData)
		err = emailutils.SendMail(mailReq, ms)
		if err != nil {
			log.Printf("unable to send mail", "error", err)
			handlersfunc.HandleMailSendError(rw, resp)
			return
		}
	}


	rw.WriteHeader(http.StatusOK)
	resp["status"] = "project viewer added successfully"
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)

}

func UserLoadProjects(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]string)
	defer r.Body.Close()
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	userID := handlersfunc.UserIDContextReader(r)
	log.Printf("Load projects of the user %d", userID)
	projects, err := projectstorage.RetrieveUserProjects(ctx, config.DB, userID)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
		return
	}

	if len(projects) == 0 {
		handlersfunc.HandleNoContent(rw, resp)
		return
	}

	rw.WriteHeader(http.StatusOK)
	json.NewEncoder(rw).Encode(projects)
}

func UserLoadPhotos(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]string)
	defer r.Body.Close()
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	userID := handlersfunc.UserIDContextReader(r)
	log.Printf("Load photos of the user %d", userID)
	photos, err := objectsstorage.RetrieveUserPhotos(ctx, config.DB, userID)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
		return
	}

	if len(photos) == 0 {
		handlersfunc.HandleNoContent(rw, resp)
		return
	}

	rw.WriteHeader(http.StatusOK)
	json.NewEncoder(rw).Encode(photos)
}

func UserLoadPersObjects(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]string)
	defer r.Body.Close()
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	userID := handlersfunc.UserIDContextReader(r)
	log.Printf("Load personalised backgrounds and stickers of the user %d", userID)
	persObjects, err := objectsstorage.RetrieveUserPersonalisedObjects(ctx, config.DB, userID)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
		return
	}

	rw.WriteHeader(http.StatusOK)
	json.NewEncoder(rw).Encode(persObjects)
}

func RetrieveTemplates(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]string)
	defer r.Body.Close()
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	log.Printf("Retrieve templates")
	templates, err := projectstorage.RetrieveTemplates(ctx, config.DB)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
		return
	}

	if len(templates) == 0 {
		handlersfunc.HandleNoContent(rw, resp)
		return
	}

	rw.WriteHeader(http.StatusOK)
	json.NewEncoder(rw).Encode(templates)
}

func NewPhoto(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]string)
	photoLinkBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		handlersfunc.HandleWrongBytesInput(rw, resp)
		return
	}
	defer r.Body.Close()
	photoLink := string(photoLinkBytes)
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	userID := handlersfunc.UserIDContextReader(r)
	log.Printf("Add new photo %s of the user %d", photoLink, userID)
	_, err = objectsstorage.AddPhoto(ctx, config.DB, photoLink, userID)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
		return
	}

	rw.WriteHeader(http.StatusOK)
	resp["status"] = "photo added successfully"
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func DeletePhoto(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]string)
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	photoID := uint(aByteToInt)
	defer r.Body.Close()
	userID := handlersfunc.UserIDContextReader(r)

	userCheck := objectsstorage.CheckUserOwnsPhoto(ctx, config.DB, userID, photoID)

	if !userCheck {
		handlersfunc.HandlePermissionError(rw, resp)
		return
	}
	_, err = objectsstorage.DeletePhoto(ctx, config.DB, photoID)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
		return
	}

	rw.WriteHeader(http.StatusOK)
	resp["status"] = "photo deleted successfully"
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func CreateBlankProject(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]string)
	var ProjectObj models.ProjectObj
	var pagesIDs []uint
	var pID uint
	var pageID uint
	var blankProject models.NewBlankProjectObj

	err := json.NewDecoder(r.Body).Decode(&ProjectObj)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, resp, err)
		return
	}

	defer r.Body.Close()
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	userID := handlersfunc.UserIDContextReader(r)
	log.Printf("Create project for user %d", userID)
	pID, err = projectstorage.CreateProject(ctx, config.DB, userID, ProjectObj.PageNumber, ProjectObj.Orientation, ProjectObj.CoverImage, ProjectObj.Name, ProjectObj.Covertype,
		ProjectObj.Bindingtype,
		ProjectObj.Papertype,
		ProjectObj.PromooffersID)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
		return
	}
	for i:=0; i < ProjectObj.PageNumber; i++ {
		pageID, err = projectstorage.AddProjectPage(ctx, config.DB, pID)
		if err != nil {
			handlersfunc.HandleDatabaseServerError(rw, resp)
			return
		}
		pagesIDs = append(pagesIDs, pageID)
	}

	rw.WriteHeader(http.StatusOK)
	resp["status"] = "project added successfully"
	blankProject.ProjectID = pID
	blankProject.PagesIDs = pagesIDs
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
	json.NewEncoder(rw).Encode(blankProject)
}

func CreateProjectFromTemplate(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]string)
	var ProjectObj models.ProjectObj
	var pagesIDs []uint
	var pID uint
	var blankProject models.NewBlankProjectObj

	err := json.NewDecoder(r.Body).Decode(&ProjectObj)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, resp, err)
		return
	}

	defer r.Body.Close()
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	userID := handlersfunc.UserIDContextReader(r)
	log.Printf("Create project for user %d from template %d", userID, ProjectObj.TemplateID)

	pID, pagesIDs, err = projectstorage.CreateProjectFromTemplate(ctx, config.DB, userID, ProjectObj.TemplateID, ProjectObj.Covertype,
		ProjectObj.Bindingtype,
		ProjectObj.Papertype,
		ProjectObj.PromooffersID)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
		return
	}


	rw.WriteHeader(http.StatusOK)
	resp["status"] = "project added successfully"
	blankProject.ProjectID = pID
	blankProject.PagesIDs = pagesIDs
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
	json.NewEncoder(rw).Encode(blankProject)
}


func CreateTemplate(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]string)
	var ProjectObj models.TemplateProjectObj

	err := json.NewDecoder(r.Body).Decode(&ProjectObj)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, resp, err)
		return
	}

	defer r.Body.Close()
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	userID := handlersfunc.UserIDContextReader(r)
	log.Printf("Create template for user %d", userID)
	_, err = projectstorage.CreateTemplate(ctx, config.DB, userID, ProjectObj.PageNumber, ProjectObj.Orientation, ProjectObj.CoverImage, ProjectObj.Category, ProjectObj.HardCopy,
		ProjectObj.Name)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
		return
	}

	rw.WriteHeader(http.StatusOK)
	resp["status"] = "template created successfully"
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func CreateDesignerProject(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]string)
	var ProjectObj models.DesignerProjectObj
	var projectID uint

	err := json.NewDecoder(r.Body).Decode(&ProjectObj)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, resp, err)
		return
	}

	defer r.Body.Close()
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	userID := handlersfunc.UserIDContextReader(r)
	log.Printf("Create designer project for user %d", userID)
	projectID, err = projectstorage.CreateProject(ctx, config.DB, userID, ProjectObj.PageNumber, ProjectObj.Orientation, ProjectObj.CoverImage, ProjectObj.Name, ProjectObj.Covertype,
		ProjectObj.Bindingtype,
		ProjectObj.Papertype,
		ProjectObj.PromooffersID)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
		return
	}

	err = projectstorage.AddProjectPhotos(ctx, config.DB, projectID, userID)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
		return
	}

	_, err = projectstorage.AddProjectEditor(ctx, config.DB, config.AdminEmail, ProjectObj.ProjectID, models.EditorCategory)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
		return
	}

	// Send notification mail
	from := "support@memoryprint.ru"
	to := []string{config.AdminEmail}
	subject := "MemoryPrint Designer Order"
	mailType := emailutils.MailDesignerOrder
	mailData := &emailutils.MailData{
		Username: "Admin",
		Code: 	emailutils.GenerateRandomString(8),
	}

	ms := &emailutils.SGMailService{config.YandexApiKey, config.MailVerifCodeExpiration, config.PassResetCodeExpiration, config.MailVerifTemplateID, config.PassResetTemplateID, config.DesignerOrderTemplateID, config.ViewerInvitationNewTemplateID, config.ViewerInvitationExistTemplateID}
	mailReq := emailutils.NewMail(from, to, subject, mailType, mailData)
	err = emailutils.SendMail(mailReq, ms)
	if err != nil {
		log.Printf("unable to send mail", "error", err)
		handlersfunc.HandleMailSendError(rw, resp)
		return
	}


	rw.WriteHeader(http.StatusOK)
	resp["status"] = "project added successfully"
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func SaveProject(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]string)
	var savedObj models.SavedProjectObj
	err := json.NewDecoder(r.Body).Decode(&savedObj)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, resp, err)
		return
	}
	
	defer r.Body.Close()
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	userID := handlersfunc.UserIDContextReader(r)
	log.Printf("Save project %d for user %d",savedObj.Project.ProjectID, userID)
	
	err = projectstorage.SaveProject(ctx, config.DB, savedObj.Project, userID)
	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
		return
	}

	for _, page := range savedObj.Pages {
		log.Printf("Save project page %d", page.PageID)

		err = projectstorage.DeletePageObjects(ctx, config.DB, page.PageID)

		if err != nil {
			handlersfunc.HandleDatabaseServerError(rw, resp)
			return
		}
			
		err = projectstorage.SavePagePhotos(ctx, config.DB, page.PageID, page.Photos)
		if err != nil {
			handlersfunc.HandleDatabaseServerError(rw, resp)
			return
		}
		err = projectstorage.SavePageDecorations(ctx, config.DB, page.PageID, page.Decorations)
		if err != nil {
			handlersfunc.HandleDatabaseServerError(rw, resp)
			return
		}
		err = projectstorage.SavePageLayout(ctx, config.DB, page.PageID, page.Layout)
		if err != nil {
			handlersfunc.HandleDatabaseServerError(rw, resp)
			return
		}
		err = projectstorage.SavePageBackground(ctx, config.DB, page.PageID, page.Background)
		if err != nil {
			handlersfunc.HandleDatabaseServerError(rw, resp)
			return
		}
		err = projectstorage.SavePageText(ctx, config.DB, page.PageID, page.TextObj)
		if err != nil {
			handlersfunc.HandleDatabaseServerError(rw, resp)
			return
		}
	}

	rw.WriteHeader(http.StatusOK)
	resp["status"] = "project saved successfully"
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func SaveTemplate(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]string)
	var savedObj models.SavedProjectObj
	err := json.NewDecoder(r.Body).Decode(&savedObj)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, resp, err)
		return
	}
	
	defer r.Body.Close()
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	userID := handlersfunc.UserIDContextReader(r)
	log.Printf("Save template %d for user %d",savedObj.Project.ProjectID, userID)
	
	err = projectstorage.SaveTemplate(ctx, config.DB, savedObj.Project, userID)
	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
		return
	}

	for _, page := range savedObj.Pages {
		log.Printf("Save project page %d", page.PageID)

		err = projectstorage.DeletePageObjects(ctx, config.DB, page.PageID)

		if err != nil {
			handlersfunc.HandleDatabaseServerError(rw, resp)
			return
		}
			
		err = projectstorage.SavePageDecorations(ctx, config.DB, page.PageID, page.Decorations)
		if err != nil {
			handlersfunc.HandleDatabaseServerError(rw, resp)
			return
		}
		err = projectstorage.SavePageLayout(ctx, config.DB, page.PageID, page.Layout)
		if err != nil {
			handlersfunc.HandleDatabaseServerError(rw, resp)
			return
		}
		err = projectstorage.SavePageBackground(ctx, config.DB, page.PageID, page.Background)
		if err != nil {
			handlersfunc.HandleDatabaseServerError(rw, resp)
			return
		}
		err = projectstorage.SavePageText(ctx, config.DB, page.PageID, page.TextObj)
		if err != nil {
			handlersfunc.HandleDatabaseServerError(rw, resp)
			return
		}
	}

	rw.WriteHeader(http.StatusOK)
	resp["status"] = "template saved successfully"
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func SavePage(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]string)
	var savedPage models.Page
	err := json.NewDecoder(r.Body).Decode(&savedPage)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, resp, err)
		return
	}
	
	defer r.Body.Close()
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	userID := handlersfunc.UserIDContextReader(r)
	log.Printf("Save page %d for user %d",savedPage.PageID, userID)
	
	err = projectstorage.DeletePageObjects(ctx, config.DB, savedPage.PageID)

	if err != nil {
			handlersfunc.HandleDatabaseServerError(rw, resp)
			return
	}
			
	err = projectstorage.SavePagePhotos(ctx, config.DB, savedPage.PageID, savedPage.Photos)
	if err != nil {
			handlersfunc.HandleDatabaseServerError(rw, resp)
			return
	}
	err = projectstorage.SavePageDecorations(ctx, config.DB, savedPage.PageID, savedPage.Decorations)
	if err != nil {
			handlersfunc.HandleDatabaseServerError(rw, resp)
			return
	}
	err = projectstorage.SavePageLayout(ctx, config.DB, savedPage.PageID, savedPage.Layout)
	if err != nil {
			handlersfunc.HandleDatabaseServerError(rw, resp)
			return
	}
	err = projectstorage.SavePageBackground(ctx, config.DB, savedPage.PageID, savedPage.Background)
	if err != nil {
			handlersfunc.HandleDatabaseServerError(rw, resp)
			return
	}
	err = projectstorage.SavePageText(ctx, config.DB, savedPage.PageID, savedPage.TextObj)
	if err != nil {
			handlersfunc.HandleDatabaseServerError(rw, resp)
			return
	}

	rw.WriteHeader(http.StatusOK)
	resp["status"] = "page saved successfully"
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func PublishTemplate(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]string)
	var projectObj models.TemplateProjectObj
	err := json.NewDecoder(r.Body).Decode(&projectObj)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, resp, err)
		return
	}
	
	defer r.Body.Close()
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	userID := handlersfunc.UserIDContextReader(r)
	log.Printf("Publish template %d for user %d",projectObj.TemplateID, userID)
	
	err = projectstorage.PublishTemplate(ctx, config.DB, projectObj)
	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
		return
	}

	rw.WriteHeader(http.StatusOK)
	resp["status"] = "template published successfully"
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func LoadProject(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]string)
	var retrievedProject []models.Page
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	projectID := uint(aByteToInt)
	defer r.Body.Close()

	userID := handlersfunc.UserIDContextReader(r)
	log.Printf("Load project %d for user %d",projectID, userID)

	userCheck := projectstorage.CheckUserHasProject(ctx, config.DB, userID, projectID)

	if !userCheck {
		handlersfunc.HandlePermissionError(rw, resp)
		return
	}
	
	projectPages, err := projectstorage.RetrieveProjectPages(ctx, config.DB, projectID, false)
	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
		return
	}
	for _, num := range projectPages {
		var retrievedPage models.Page
		retrievedPage.ProjectID = projectID
		retrievedPage.PageID = num
		retrievedPage.Decorations, err = projectstorage.RetrievePageDecorations(ctx, config.DB, num)
		if err != nil {
			handlersfunc.HandleDatabaseServerError(rw, resp)
			return
		}
		retrievedPage.Photos, err = projectstorage.RetrievePagePhotos(ctx, config.DB, num)
		if err != nil {
			handlersfunc.HandleDatabaseServerError(rw, resp)
			return
		}
		retrievedPage.Background, err = projectstorage.RetrievePageBackground(ctx, config.DB, num)
		if err != nil {
			handlersfunc.HandleDatabaseServerError(rw, resp)
			return
		}
		retrievedPage.Layout, err = projectstorage.RetrievePageLayout(ctx, config.DB, num)
		if err != nil {
			handlersfunc.HandleDatabaseServerError(rw, resp)
			return
		}
		retrievedPage.TextObj, err = projectstorage.RetrievePageText(ctx, config.DB, num)
		if err != nil {
			handlersfunc.HandleDatabaseServerError(rw, resp)
			return
		}
		retrievedProject = append(retrievedProject, retrievedPage)
    }

	rw.WriteHeader(http.StatusOK)
	json.NewEncoder(rw).Encode(retrievedProject)
}

func LoadTemplate(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]string)
	var retrievedProject []models.Page

	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	templateID := uint(aByteToInt)
	defer r.Body.Close()

	userID := handlersfunc.UserIDContextReader(r)
	log.Printf("Load template %d for user %d",templateID, userID)
	
	templatePages, err := projectstorage.RetrieveProjectPages(ctx, config.DB, templateID, true)
	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
		return
	}
	for _, num := range templatePages {
		var retrievedPage models.Page
		retrievedPage.ProjectID = templateID
		retrievedPage.PageID = num
		retrievedPage.Decorations, err = projectstorage.RetrievePageDecorations(ctx, config.DB, num)
		if err != nil {
			handlersfunc.HandleDatabaseServerError(rw, resp)
			return
		}
		retrievedPage.Photos, err = projectstorage.RetrievePagePhotos(ctx, config.DB, num)
		if err != nil {
			handlersfunc.HandleDatabaseServerError(rw, resp)
			return
		}
		retrievedPage.Background, err = projectstorage.RetrievePageBackground(ctx, config.DB, num)
		if err != nil {
			handlersfunc.HandleDatabaseServerError(rw, resp)
			return
		}
		retrievedPage.Layout, err = projectstorage.RetrievePageLayout(ctx, config.DB, num)
		if err != nil {
			handlersfunc.HandleDatabaseServerError(rw, resp)
			return
		}
		retrievedPage.TextObj, err = projectstorage.RetrievePageText(ctx, config.DB, num)
		if err != nil {
			handlersfunc.HandleDatabaseServerError(rw, resp)
			return
		}
		retrievedProject = append(retrievedProject, retrievedPage)
    }

	rw.WriteHeader(http.StatusOK)
	json.NewEncoder(rw).Encode(retrievedProject)
}


func DeleteProject(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]string)
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	projectID := uint(aByteToInt)
	defer r.Body.Close()
	userID := handlersfunc.UserIDContextReader(r)

	userCheck := projectstorage.CheckUserHasProject(ctx, config.DB, userID, projectID)

	if !userCheck {
		handlersfunc.HandlePermissionError(rw, resp)
		return
	}
	err = projectstorage.DeleteProject(ctx, config.DB, projectID)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
		return
	}

	rw.WriteHeader(http.StatusOK)
	resp["status"] = "project deleted successfully"
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func AddProjectPage(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]string)
	var pageID uint
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	projectID := uint(aByteToInt)
	defer r.Body.Close()
	log.Printf("Add new page for project %d", projectID)
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	pageID, err = projectstorage.AddProjectPage(ctx, config.DB, projectID)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
		return
	}

	rw.WriteHeader(http.StatusOK)
	resp["status"] = "project page added successfully"
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
	json.NewEncoder(rw).Encode(pageID)
}

func DuplicatePage(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]string)
	var newPageID uint
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	duplicateID := uint(aByteToInt)
	defer r.Body.Close()
	log.Printf("Duplicate page %d", duplicateID)
	
	newPageID, err = projectstorage.DuplicateProjectPage(ctx, config.DB, duplicateID)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
		return
	}

	rw.WriteHeader(http.StatusOK)
	resp["status"] = "page duplicated successfully"
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
	json.NewEncoder(rw).Encode(newPageID)
}

func DeletePage(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]string)
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	pageID := uint(aByteToInt)
	defer r.Body.Close()
	
	err = projectstorage.DeletePage(ctx, config.DB, pageID)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
		return
	}

	rw.WriteHeader(http.StatusOK)
	resp["status"] = "page deleted successfully"
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func LoadProjectPhotos(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]string)
	projectIDBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		handlersfunc.HandleWrongBytesInput(rw, resp)
		return
	}
	defer r.Body.Close()
	aByteToInt, _ := strconv.Atoi(string(projectIDBytes))
	projectID := uint(aByteToInt)
	log.Printf("Load project photos for project %d", projectID)
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	userID := handlersfunc.UserIDContextReader(r)

	userCheck := projectstorage.CheckUserHasProject(ctx, config.DB, userID, projectID)

	if !userCheck {
		handlersfunc.HandlePermissionError(rw, resp)
		return
	}
	
	projectPhotos, err := projectstorage.RetrieveProjectPhotos(ctx, config.DB, projectID)
	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
		return
	}
	

	rw.WriteHeader(http.StatusOK)
	json.NewEncoder(rw).Encode(projectPhotos)
}

func LoadProjectSession(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]string)
	
	defer r.Body.Close()
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()

	log.Printf("Load project session")
	
	projectSession, err := objectsstorage.LoadProjectSession(ctx, config.DB)
	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
		return
	}
	

	rw.WriteHeader(http.StatusOK)
	json.NewEncoder(rw).Encode(projectSession)
}

func CreateDecor(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]string)
	var DecorObj models.PersonalisedObject
	var dID uint

	err := json.NewDecoder(r.Body).Decode(&DecorObj)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, resp, err)
		return
	}

	defer r.Body.Close()
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	userID := handlersfunc.UserIDContextReader(r)
	log.Printf("Create decor for user %d", userID)
	dID, err = objectsstorage.AddDecoration(ctx, config.DB, DecorObj, userID)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
		return
	}


	rw.WriteHeader(http.StatusOK)
	resp["status"] = "decor added successfully"
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
	json.NewEncoder(rw).Encode(dID)
}

func DeleteDecor(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]string)
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	decorID := uint(aByteToInt)
	defer r.Body.Close()
	
	
	err = objectsstorage.DeleteDecoration(ctx, config.DB, decorID)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
		return
	}

	rw.WriteHeader(http.StatusOK)
	resp["status"] = "decor deleted successfully"
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func CreateBackground(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]string)
	var BackgroundObj models.PersonalisedObject
	var bID uint

	err := json.NewDecoder(r.Body).Decode(&BackgroundObj)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, resp, err)
		return
	}

	defer r.Body.Close()
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	userID := handlersfunc.UserIDContextReader(r)
	log.Printf("Create background for user %d", userID)
	bID, err = objectsstorage.AddBackground(ctx, config.DB, BackgroundObj, userID)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
		return
	}


	rw.WriteHeader(http.StatusOK)
	resp["status"] = "background added successfully"
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
	json.NewEncoder(rw).Encode(bID)
}

func DeleteBackground(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]string)
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	backgroundID := uint(aByteToInt)
	defer r.Body.Close()
	
	err = objectsstorage.DeleteBackground(ctx, config.DB, backgroundID)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
		return
	}

	rw.WriteHeader(http.StatusOK)
	resp["status"] = "background deleted successfully"
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}