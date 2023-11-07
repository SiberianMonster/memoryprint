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
	"github.com/SiberianMonster/memoryprint/internal/models"
	"github.com/SiberianMonster/memoryprint/internal/projectstorage"
	"github.com/SiberianMonster/memoryprint/internal/objectsstorage"
	"github.com/SiberianMonster/memoryprint/internal/handlersfunc"
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
	photoIDBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		handlersfunc.HandleWrongBytesInput(rw, resp)
	}
	defer r.Body.Close()
	aByteToInt, _ := strconv.Atoi(string(photoIDBytes))
	photoID := uint(aByteToInt)
	log.Printf("Delete photo %d", photoID)
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	userID := handlersfunc.UserIDContextReader(r)

	userCheck := objectsstorage.CheckUserOwnsPhoto(ctx, config.DB, userID, photoID)

	if !userCheck {
		handlersfunc.HandlePermissionError(rw, resp)
	}
	_, err = objectsstorage.DeletePhoto(ctx, config.DB, photoID)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
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

func CreateProject(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]string)
	var ProjectObj models.ProjectObj

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
	_, err = projectstorage.CreateProject(ctx, config.DB, userID, ProjectObj.PageNumber, ProjectObj.Orientation, ProjectObj.CoverImage, ProjectObj.Name)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
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


func CreateTemplate(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]string)
	var ProjectObj models.ProjectObj

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
	_, err = projectstorage.CreateProject(ctx, config.DB, userID, ProjectObj.PageNumber, ProjectObj.Orientation, ProjectObj.CoverImage, ProjectObj.Name)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
		return
	}

	rw.WriteHeader(http.StatusOK)
	resp["status"] = "template added successfully"
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
	projectID, err = projectstorage.CreateProject(ctx, config.DB, userID, ProjectObj.PageNumber, ProjectObj.Orientation, ProjectObj.CoverImage, ProjectObj.Name)

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

	ms := &emailutils.SGMailService{config.YandexApiKey, config.MailVerifCodeExpiration, config.PassResetCodeExpiration, config.MailVerifTemplateID, config.PassResetTemplateID, config.DesignerOrderTemplateID}
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
	var ProjectObj models.ProjectObj

	err := json.NewDecoder(r.Body).Decode(&ProjectObj)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, resp, err)
	}
	
	defer r.Body.Close()
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	userID := handlersfunc.UserIDContextReader(r)
	log.Printf("Save project %d for user %d",ProjectObj.ProjectID, userID)
	
	err = projectstorage.SaveProject(ctx, config.DB, ProjectObj, userID)
	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
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

func LoadProject(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]string)
	var retrievedProject []models.Page
	projectIDBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		handlersfunc.HandleWrongBytesInput(rw, resp)
	}
	defer r.Body.Close()
	aByteToInt, _ := strconv.Atoi(string(projectIDBytes))
	projectID := uint(aByteToInt)
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()

	userID := handlersfunc.UserIDContextReader(r)
	log.Printf("Load project %d for user %d",projectID, userID)

	userCheck := projectstorage.CheckUserHasProject(ctx, config.DB, userID, projectID)

	if !userCheck {
		handlersfunc.HandlePermissionError(rw, resp)
	}
	
	projectPages, err := projectstorage.RetrieveProjectPages(ctx, config.DB, projectID)
	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
	}
	for _, num := range projectPages {
		var retrievedPage models.Page
		retrievedPage.ProjectID = projectID
		retrievedPage.PageID = num
		retrievedPage.Decorations, err = projectstorage.RetrievePageDecorations(ctx, config.DB, num)
		if err != nil {
			handlersfunc.HandleDatabaseServerError(rw, resp)
		}
		retrievedPage.Photos, err = projectstorage.RetrievePagePhotos(ctx, config.DB, num)
		if err != nil {
			handlersfunc.HandleDatabaseServerError(rw, resp)
		}
		retrievedPage.Background, err = projectstorage.RetrievePageBackground(ctx, config.DB, num)
		if err != nil {
			handlersfunc.HandleDatabaseServerError(rw, resp)
		}
		retrievedPage.Layout, err = projectstorage.RetrievePageLayout(ctx, config.DB, num)
		if err != nil {
			handlersfunc.HandleDatabaseServerError(rw, resp)
		}
		retrievedPage.TextObj, err = projectstorage.RetrievePageText(ctx, config.DB, num)
		if err != nil {
			handlersfunc.HandleDatabaseServerError(rw, resp)
		}
		retrievedProject = append(retrievedProject, retrievedPage)
    }

	rw.WriteHeader(http.StatusOK)
	json.NewEncoder(rw).Encode(retrievedProject)
}


func LoadTemplate(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]string)
	var retrievedProject []models.Page
	projectIDBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		handlersfunc.HandleWrongBytesInput(rw, resp)
	}
	defer r.Body.Close()
	aByteToInt, _ := strconv.Atoi(string(projectIDBytes))
	projectID := uint(aByteToInt)
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	log.Printf("Load template %d",projectID)
	
	projectPages, err := projectstorage.RetrieveProjectPages(ctx, config.DB, projectID)
	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
	}
	for _, num := range projectPages {
		var retrievedPage models.Page
		retrievedPage.ProjectID = projectID
		retrievedPage.PageID = num
		retrievedPage.Decorations, err = projectstorage.RetrievePageDecorations(ctx, config.DB, num)
		if err != nil {
			handlersfunc.HandleDatabaseServerError(rw, resp)
		}
		retrievedPage.Background, err = projectstorage.RetrievePageBackground(ctx, config.DB, num)
		if err != nil {
			handlersfunc.HandleDatabaseServerError(rw, resp)
		}
		retrievedPage.Layout, err = projectstorage.RetrievePageLayout(ctx, config.DB, num)
		if err != nil {
			handlersfunc.HandleDatabaseServerError(rw, resp)
		}
		retrievedPage.TextObj, err = projectstorage.RetrievePageText(ctx, config.DB, num)
		if err != nil {
			handlersfunc.HandleDatabaseServerError(rw, resp)
		}
		retrievedProject = append(retrievedProject, retrievedPage)
    }

	rw.WriteHeader(http.StatusOK)
	json.NewEncoder(rw).Encode(retrievedProject)
}


func SavePage(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]string)
	var PageObj models.Page

	err := json.NewDecoder(r.Body).Decode(&PageObj)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, resp, err)
	}
	log.Printf("Save project page %d", PageObj.PageID)
	
	defer r.Body.Close()
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	
	err = projectstorage.SavePagePhotos(ctx, config.DB, PageObj.PageID, PageObj.Photos)
	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
	}
	err = projectstorage.SavePageDecorations(ctx, config.DB, PageObj.PageID, PageObj.Decorations)
	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
	}
	err = projectstorage.SavePageLayout(ctx, config.DB, PageObj.PageID, PageObj.Layout)
	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
	}
	err = projectstorage.SavePageBackground(ctx, config.DB, PageObj.PageID, PageObj.Background)
	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
	}
	err = projectstorage.SavePageText(ctx, config.DB, PageObj.PageID, PageObj.TextObj)
	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
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

func DeleteProject(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]string)
	projectIDBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		handlersfunc.HandleWrongBytesInput(rw, resp)
	}
	defer r.Body.Close()
	aByteToInt, _ := strconv.Atoi(string(projectIDBytes))
	projectID := uint(aByteToInt)
	log.Printf("Delete project %d", projectID)
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	userID := handlersfunc.UserIDContextReader(r)

	userCheck := projectstorage.CheckUserHasProject(ctx, config.DB, userID, projectID)

	if !userCheck {
		handlersfunc.HandlePermissionError(rw, resp)
	}
	err = projectstorage.DeleteProject(ctx, config.DB, projectID)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
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
	projectIDBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		handlersfunc.HandleWrongBytesInput(rw, resp)
	}
	defer r.Body.Close()
	aByteToInt, _ := strconv.Atoi(string(projectIDBytes))
	projectID := uint(aByteToInt)
	log.Printf("Add new page for project %d", projectID)
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	err = projectstorage.AddProjectPage(ctx, config.DB, projectID)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
	}

	rw.WriteHeader(http.StatusOK)
	resp["status"] = "project page added successfully"
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
	}
	
	projectPhotos, err := projectstorage.RetrieveProjectPhotos(ctx, config.DB, projectID)
	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
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
	}
	

	rw.WriteHeader(http.StatusOK)
	json.NewEncoder(rw).Encode(projectSession)
}