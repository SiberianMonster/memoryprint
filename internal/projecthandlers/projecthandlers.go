// Handlers package contains endpoints handlers for the Photo Book Editor module.
//
// https://github.com/SiberianMonster/memoryprint/tree/development/internal/projecthandlers
package projecthandlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"os"
	"net"
	"os/exec"
	"fmt"

	"github.com/SiberianMonster/memoryprint/internal/config"
	"github.com/SiberianMonster/memoryprint/internal/models"
	"github.com/SiberianMonster/memoryprint/internal/projectstorage"
	"github.com/SiberianMonster/memoryprint/internal/userstorage"
	"github.com/SiberianMonster/memoryprint/internal/objectsstorage"
	"github.com/SiberianMonster/memoryprint/internal/orderstorage"
	"github.com/SiberianMonster/memoryprint/internal/emailutils"
	"github.com/SiberianMonster/memoryprint/internal/handlersfunc"
	"github.com/SiberianMonster/memoryprint/internal/imagehandlers"
	"github.com/go-playground/validator/v10"
	"golang.org/x/exp/slices"
	"github.com/tebeka/selenium"
  	"github.com/tebeka/selenium/chrome"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
	_ "github.com/lib/pq"
)

var err error
var resp map[string]string


func UserLoadPhotos(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]models.ResponsePhotos)
	var rPhotos models.ResponsePhotos
	defer r.Body.Close()
	sorting := r.URL.Query().Get("sorting")
	myUrl, _ := url.Parse(r.URL.String())	
	params, _ := url.ParseQuery(myUrl.RawQuery)
	rOffset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	rLimit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset := uint(rOffset)
	limit := uint(rLimit)
	var lo models.LimitOffset
	if _, ok := params["offset"]; ok {
		lo.Offset = &offset
	}
	if limit != 0 {
		lo.Limit = &limit
	}
	validate := validator.New()

    // Validate the User struct
    err = validate.Struct(lo)
    if err != nil {
        // Validation failed, handle the error
		handlersfunc.HandleValidationError(rw, err)
        return
    }

	
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	userID := handlersfunc.UserIDContextReader(r)
	log.Printf("Load photos of the user %d", userID)
	rPhotos, err := objectsstorage.RetrieveUserPhotos(ctx, config.DB, userID, sorting, uint(rOffset), uint(rLimit))

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}

	rw.WriteHeader(http.StatusOK)
	resp["response"] = rPhotos
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)

}


func NewPhoto(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]models.ResponseUploadedPhoto)
	var photoParams models.Photo
	var rPhoto models.ResponseUploadedPhoto
	var pID uint

	err := json.NewDecoder(r.Body).Decode(&photoParams)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, err)
		return
	}
	
	defer r.Body.Close()
	// Create a new validator instance
    validate := validator.New()

    // Validate the User struct
    err = validate.Struct(photoParams)
    if err != nil {
        // Validation failed, handle the error
		handlersfunc.HandleValidationError(rw, err)
        return
    }
	
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	userID := handlersfunc.UserIDContextReader(r)
	log.Printf("Add new photo of the user %d", userID)
	pID, err = objectsstorage.AddPhoto(ctx, config.DB, photoParams.Link, *photoParams.SmallImage, userID)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}

	rw.WriteHeader(http.StatusOK)
	rPhoto.PhotoID = pID
	resp["response"] = rPhoto
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func DeletePhoto(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]uint)
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	photoID := uint(aByteToInt)
	defer r.Body.Close()
	userID := handlersfunc.UserIDContextReader(r)

	userCheck := objectsstorage.CheckUserOwnsPhoto(ctx, config.DB, userID, photoID)

	if !userCheck {
		handlersfunc.HandlePermissionError(rw)
		return
	}
	_, err = objectsstorage.DeletePhoto(ctx, config.DB, photoID)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}

	rw.WriteHeader(http.StatusOK)
	resp["response"] = 1
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}


func CreateBlankProject(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]models.ResponseCreatedProject)
	var ProjectObj models.NewBlankProjectObj
	var pID uint
	var rProject models.ResponseCreatedProject
	
	err := json.NewDecoder(r.Body).Decode(&ProjectObj)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, err)
		return
	}

	defer r.Body.Close()
	// Create a new validator instance
    validate := validator.New()

    // Validate the User struct
    err = validate.Struct(ProjectObj)
    if err != nil {
        // Validation failed, handle the error
		handlersfunc.HandleValidationError(rw, err)
        return
    }
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	userID := handlersfunc.UserIDContextReader(r)
	log.Printf("Create project for user %d", userID)
	pID, err = projectstorage.CreateProject(ctx, config.DB, userID, ProjectObj)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}

	rw.WriteHeader(http.StatusOK)
	rProject.ProjectID = pID
	resp["response"] = rProject
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func DuplicateProject(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]uint)	
	var pID uint
	var rProject models.ResponseCreatedProject
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	projectID := uint(aByteToInt)
	defer r.Body.Close()
	userID := handlersfunc.UserIDContextReader(r)
	checkExists := orderstorage.CheckProject(ctx, config.DB, projectID)
	if !checkExists {
			handlersfunc.HandleMissingProjectError(rw)
			return
	}

	userCheck := userstorage.CheckUserHasProject(ctx, config.DB, userID, projectID)

	if userCheck == false {
		handlersfunc.HandlePermissionError(rw)
		return
	}
	log.Printf("Duplicate project for user %d", userID)
	pID, err = projectstorage.DuplicateProject(ctx, config.DB, projectID, userID)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}

	rw.WriteHeader(http.StatusOK)
	rProject.ProjectID = pID
	resp["response"] = 1
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func DeleteProject(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]uint)
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	projectID := uint(aByteToInt)
	defer r.Body.Close()
	
	userID := handlersfunc.UserIDContextReader(r)
	log.Printf("Delete project %d for user %d",projectID, userID)
	checkExists := orderstorage.CheckProject(ctx, config.DB, projectID)
	if !checkExists {
			handlersfunc.HandleMissingProjectError(rw)
			return
	}

	userCheck := userstorage.CheckUserHasProject(ctx, config.DB, userID, projectID)

	if !userCheck {
		handlersfunc.HandlePermissionError(rw)
		return
	}

	err = projectstorage.DeleteProject(ctx, config.DB, projectID)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}

	rw.WriteHeader(http.StatusOK)
	resp["response"] = 1
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func PublishTemplate(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]uint)
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	templateID := uint(aByteToInt)
	
	defer r.Body.Close()
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()

	
	err = projectstorage.PublishTemplate(ctx, config.DB, templateID)
	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}

	rw.WriteHeader(http.StatusOK)
	resp["response"] = 1
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func UnpublishTemplate(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]uint)
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	templateID := uint(aByteToInt)
	
	defer r.Body.Close()
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	checkExists := projectstorage.CheckTemplate(ctx, config.DB, templateID)
	if !checkExists {
		handlersfunc.HandleMissingTemplateError(rw)
		return
	}

	
	err = projectstorage.UnpublishTemplate(ctx, config.DB, templateID)
	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}

	rw.WriteHeader(http.StatusOK)
	resp["response"] = 1
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func UnpublishProject(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]uint)
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	projectID := uint(aByteToInt)
	
	defer r.Body.Close()
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()

	userID := handlersfunc.UserIDContextReader(r)
	log.Printf("Unpublish project %d for user %d",projectID, userID)
	checkExists := orderstorage.CheckProject(ctx, config.DB, projectID)
	if !checkExists {
			handlersfunc.HandleMissingProjectError(rw)
			return
	}

	userCheck := userstorage.CheckUserHasProject(ctx, config.DB, userID, projectID)

	if !userCheck {
		handlersfunc.HandlePermissionError(rw)
		return
	}

	
	err = projectstorage.UnpublishProject(ctx, config.DB, projectID)
	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}

	rw.WriteHeader(http.StatusOK)
	resp["response"] = 1
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func LoadProject(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]models.ResponseProjectObj)
	var retrievedProject models.ResponseProjectObj
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	projectID := uint(aByteToInt)
	defer r.Body.Close()

	userID := handlersfunc.UserIDContextReader(r)
	log.Printf("Load project %d for user %d",projectID, userID)
	checkExists := orderstorage.CheckProject(ctx, config.DB, projectID)
	if !checkExists {
			handlersfunc.HandleMissingProjectError(rw)
			return
	}


	userCheck := userstorage.CheckUserHasProject(ctx, config.DB, userID, projectID)

	if !userCheck {
		handlersfunc.HandlePermissionError(rw)
		return
	}
	
	retrievedProject, err = projectstorage.LoadProject(ctx, config.DB, projectID)
	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}
	log.Println(retrievedProject)
	var leatherID *uint
	retrievedProject.Pages, err = projectstorage.RetrieveProjectPages(ctx, config.DB, projectID, false, leatherID, retrievedProject.Size)
	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}

	log.Println(retrievedProject)
	rw.WriteHeader(http.StatusOK)
	resp["response"] = retrievedProject
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func LoadTemplate(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]models.SavedTemplateObj)
	var retrievedProject models.SavedTemplateObj
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	projectID := uint(aByteToInt)
	defer r.Body.Close()
	checkExists := projectstorage.CheckTemplatePublished(ctx, config.DB, projectID)
	if !checkExists {
			rw.WriteHeader(http.StatusForbidden)
			jsonResp, err := json.Marshal(resp)
			if err != nil {
				log.Printf("Error happened in JSON marshal. Err: %s", err)
				return
			}
			rw.Write(jsonResp)
			return
	}

	retrievedProject, err = projectstorage.LoadTemplate(ctx, config.DB, projectID)
	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}
	rw.WriteHeader(http.StatusOK)
	resp["response"] = retrievedProject
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func AdminLoadTemplate(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]models.SavedTemplateObj)
	var retrievedProject models.SavedTemplateObj
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	projectID := uint(aByteToInt)
	defer r.Body.Close()
	checkExists := projectstorage.CheckTemplate(ctx, config.DB, projectID)
	if !checkExists {
		handlersfunc.HandleMissingTemplateError(rw)
		return
	}

	retrievedProject, err = projectstorage.AdminLoadTemplate(ctx, config.DB, projectID)
	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}
	rw.WriteHeader(http.StatusOK)
	resp["response"] = retrievedProject
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}




func CreateDecor(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]models.ResponseCreatedDecoration)
	var DecorObj models.PersonalisedObject
	var dID uint
	var rDecoration models.ResponseCreatedDecoration

	err := json.NewDecoder(r.Body).Decode(&DecorObj)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, err)
		return
	}

	defer r.Body.Close()
	// Create a new validator instance
    validate := validator.New()

    // Validate the User struct
    err = validate.Struct(DecorObj)
    if err != nil {
        // Validation failed, handle the error
		handlersfunc.HandleValidationError(rw, err)
        return
    }
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	userID := handlersfunc.UserIDContextReader(r)
	log.Printf("Create decor for user %d", userID)
	dID, err = objectsstorage.AddDecoration(ctx, config.DB, DecorObj, userID)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}


	rw.WriteHeader(http.StatusOK)
	rDecoration.DecorationID = dID
	resp["response"] = rDecoration
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func DeleteDecor(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]uint)
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	decorID := uint(aByteToInt)
	defer r.Body.Close()
	userID := handlersfunc.UserIDContextReader(r)
	
	
	err = objectsstorage.DeleteDecoration(ctx, config.DB, userID, decorID)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}

	rw.WriteHeader(http.StatusOK)
	resp["response"] = 1
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func DeleteTemplate(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]uint)
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	templateID := uint(aByteToInt)
	defer r.Body.Close()
	
	
	err = projectstorage.DeleteTemplate(ctx, config.DB, templateID)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}

	rw.WriteHeader(http.StatusOK)
	resp["response"] = 1
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func CreateBackground(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]models.ResponseCreatedBackground)
	var BackgroundObj models.PersonalisedObject
	var bID uint
	var rBackground models.ResponseCreatedBackground

	err := json.NewDecoder(r.Body).Decode(&BackgroundObj)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, err)
		return
	}

	defer r.Body.Close()
	// Create a new validator instance
    validate := validator.New()

    // Validate the User struct
    err = validate.Struct(BackgroundObj)
    if err != nil {
        // Validation failed, handle the error
		handlersfunc.HandleValidationError(rw, err)
        return
    }
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	userID := handlersfunc.UserIDContextReader(r)
	log.Printf("Create background for user %d", userID)
	bID, err = objectsstorage.AddBackground(ctx, config.DB, BackgroundObj, userID)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}


	rw.WriteHeader(http.StatusOK)
	rBackground.BackgroundID = bID
	resp["response"] = rBackground
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func DeleteBackground(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]uint)
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	backgroundID := uint(aByteToInt)
	defer r.Body.Close()
	userID := handlersfunc.UserIDContextReader(r)
	
	err = objectsstorage.DeleteBackground(ctx, config.DB, userID, backgroundID)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}

	rw.WriteHeader(http.StatusOK)
	resp["response"] = 1
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func LoadBackground(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]models.ResponseBackground)
	var requestB models.RequestBackground

	defer r.Body.Close()
	myUrl, _ := url.Parse(r.URL.String())	
	params, _ := url.ParseQuery(myUrl.RawQuery)
	rOffset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	rLimit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	requestB.Offset = uint(rOffset)
	requestB.Limit = uint(rLimit)
	var lo models.LimitOffset
	if _, ok := params["offset"]; ok {
		lo.Offset = &requestB.Offset
	}
	if requestB.Limit != 0 {
		lo.Limit = &requestB.Limit
	}
	validate := validator.New()

    // Validate the User struct
    err = validate.Struct(lo)
    if err != nil {
        // Validation failed, handle the error
		handlersfunc.HandleValidationError(rw, err)
        return
    }

	qtype := r.URL.Query().Get("type")

    if qtype != "" {
		requestB.Type = strings.ToUpper(r.URL.Query().Get("type"))
	}
	requestB.IsFavourite, _ = strconv.ParseBool(r.URL.Query().Get("isfavourite"))
	requestB.IsPersonal, _ = strconv.ParseBool(r.URL.Query().Get("ispersonal"))
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	userID := handlersfunc.UserIDContextReader(r)
	log.Printf("Load backgrounds for user %d", userID)
	
	
	log.Println(requestB)
	backgroundSet, err := objectsstorage.LoadBackgrounds(ctx, config.DB, userID, requestB.Offset, requestB.Limit, requestB.Type, requestB.IsFavourite, requestB.IsPersonal)
	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}
	

	rw.WriteHeader(http.StatusOK)
	resp["response"] = backgroundSet
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func AdminCreateBackground(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]models.ResponseCreatedBackground)
	var BackgroundObj models.Background
	var bID uint
	var rBackground models.ResponseCreatedBackground

	err := json.NewDecoder(r.Body).Decode(&BackgroundObj)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, err)
		return
	}

	defer r.Body.Close()
	// Create a new validator instance
    validate := validator.New()

    // Validate the User struct
    err = validate.Struct(BackgroundObj)
    if err != nil {
        // Validation failed, handle the error
		handlersfunc.HandleValidationError(rw, err)
        return
    }
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	log.Printf("Create admin background for user")
	bID, err = objectsstorage.AddAdminBackground(ctx, config.DB, BackgroundObj)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}


	rw.WriteHeader(http.StatusOK)
	rBackground.BackgroundID = bID
	resp["response"] = rBackground
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func AdminUpdateBackground(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]uint)
	var BackgroundObj models.Background

	err := json.NewDecoder(r.Body).Decode(&BackgroundObj)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, err)
		return
	}
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	backgroundID := uint(aByteToInt)

	defer r.Body.Close()
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	log.Printf("Update admin background for user")
	err = objectsstorage.UpdateBackground(ctx, config.DB, backgroundID, BackgroundObj)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}


	rw.WriteHeader(http.StatusOK)
	resp["response"] = 1
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func AdminDeleteBackground(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]uint)
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	backgroundID := uint(aByteToInt)
	defer r.Body.Close()
	
	err = objectsstorage.AdminDeleteBackground(ctx, config.DB, backgroundID)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}

	rw.WriteHeader(http.StatusOK)
	resp["response"] = 1
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func FavourBackground(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]uint)
	var BackgroundObj models.PersonalisedObject

	err := json.NewDecoder(r.Body).Decode(&BackgroundObj)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, err)
		return
	}
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	BackgroundObj.ObjectID = uint(aByteToInt)
	defer r.Body.Close()
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	userID := handlersfunc.UserIDContextReader(r)
	log.Printf("Favour background for user %d", userID)
	err = objectsstorage.FavourBackground(ctx, config.DB, BackgroundObj, userID)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}


	rw.WriteHeader(http.StatusOK)
	resp["response"] = 1
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func LoadDecoration(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]models.ResponseDecoration)
	var requestD models.RequestDecoration
	myUrl, _ := url.Parse(r.URL.String())	
	params, _ := url.ParseQuery(myUrl.RawQuery)

	rOffset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	rLimit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	requestD.Offset = uint(rOffset)
	requestD.Limit = uint(rLimit)
	var lo models.LimitOffset
	if _, ok := params["offset"]; ok {
		lo.Offset = &requestD.Offset
	}
	if requestD.Limit != 0 {
		lo.Limit = &requestD.Limit
	}
	validate := validator.New()

    // Validate the User struct
    err = validate.Struct(lo)
    if err != nil {
        // Validation failed, handle the error
		handlersfunc.HandleValidationError(rw, err)
        return
    }
	
	category := r.URL.Query().Get("category")

    if category != "" {
		requestD.Category = strings.ToUpper(r.URL.Query().Get("category"))
	}
	qtype := r.URL.Query().Get("type")

    if qtype != "" {
		requestD.Type = strings.ToUpper(r.URL.Query().Get("type"))
	}

	requestD.IsFavourite, _ = strconv.ParseBool(r.URL.Query().Get("isfavourite"))
	requestD.IsPersonal, _ = strconv.ParseBool(r.URL.Query().Get("ispersonal"))

	defer r.Body.Close()
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	userID := handlersfunc.UserIDContextReader(r)
	log.Printf("Load decorations for user %d", userID)
	
	
	
	decorationSet, err := objectsstorage.LoadDecorations(ctx, config.DB, userID, requestD.Offset, requestD.Limit, requestD.Type, requestD.Category, requestD.IsFavourite, requestD.IsPersonal)
	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}
	
	rw.WriteHeader(http.StatusOK)
	resp["response"] = decorationSet
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func AdminCreateDecoration(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]models.ResponseCreatedDecoration)
	var DecorationObj models.Decoration
	var dID uint
	var rDecoration models.ResponseCreatedDecoration

	err := json.NewDecoder(r.Body).Decode(&DecorationObj)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, err)
		return
	}

	defer r.Body.Close()
	// Create a new validator instance
    validate := validator.New()

    // Validate the User struct
    err = validate.Struct(DecorationObj)
    if err != nil {
        // Validation failed, handle the error
		handlersfunc.HandleValidationError(rw, err)
        return
    }
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	log.Printf("Create admin decoration for user")
	dID, err = objectsstorage.AddAdminDecoration(ctx, config.DB, DecorationObj)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}


	rw.WriteHeader(http.StatusOK)
	rDecoration.DecorationID = dID
	resp["response"] = rDecoration
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)

}

func AdminUpdateDecoration(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]uint)
	var DecorationObj models.Decoration

	err := json.NewDecoder(r.Body).Decode(&DecorationObj)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, err)
		return
	}
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	decorationID := uint(aByteToInt)

	defer r.Body.Close()
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	log.Printf("Update admin decoration for user")
	err = objectsstorage.UpdateDecoration(ctx, config.DB, decorationID, DecorationObj)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}


	rw.WriteHeader(http.StatusOK)
	resp["response"] = 1
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func AdminDeleteDecoration(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]uint)
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	decorationID := uint(aByteToInt)
	defer r.Body.Close()
	
	err = objectsstorage.AdminDeleteDecoration(ctx, config.DB, decorationID)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}

	rw.WriteHeader(http.StatusOK)
	resp["response"] = 1
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func FavourDecoration(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]uint)
	var DecorationObj models.PersonalisedObject

	err := json.NewDecoder(r.Body).Decode(&DecorationObj)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, err)
		return
	}
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	DecorationObj.ObjectID = uint(aByteToInt)
	defer r.Body.Close()
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	userID := handlersfunc.UserIDContextReader(r)
	log.Printf("Favour decoration for user %d", userID)
	err = objectsstorage.FavourDecoration(ctx, config.DB, DecorationObj, userID)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}


	rw.WriteHeader(http.StatusOK)
	resp["response"] = 1
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func LoadLayouts(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]models.ResponseLayout)
	var requestL models.RequestLayout
	myUrl, _ := url.Parse(r.URL.String())	
	params, _ := url.ParseQuery(myUrl.RawQuery)

	rOffset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	rLimit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	rCountImages, _ := strconv.Atoi(r.URL.Query().Get("count_images"))
	requestL.Offset = uint(rOffset)
	requestL.Limit = uint(rLimit)
	
	var lo models.LimitOffsetVariant
	if _, ok := params["offset"]; ok {
		lo.Offset = &requestL.Offset
	}
	if requestL.Limit != 0 {
		lo.Limit = &requestL.Limit
	}
	requestL.Size = strings.ToUpper(r.URL.Query().Get("size"))
	if requestL.Size != "" {
		lo.Size = &requestL.Size
	}
	var variant string
	variant = strings.ToUpper(r.URL.Query().Get("variant"))
	if variant != "" {
		lo.Variant = &variant
	}
	var isCover bool
	isCover, _ = strconv.ParseBool(r.URL.Query().Get("is_cover"))
	
	lo.IsCover = &isCover

	validate := validator.New()

    // Validate the User struct
    err = validate.Struct(lo)
    if err != nil {
        // Validation failed, handle the error
		handlersfunc.HandleValidationError(rw, err)
        return
    }
	requestL.CountImages = uint(rCountImages)
	requestL.IsFavourite, _ = strconv.ParseBool(r.URL.Query().Get("isfavourite"))

	defer r.Body.Close()
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	userID := handlersfunc.UserIDContextReader(r)
	log.Printf("Load layouts for user %d", userID)
	
	
	
	layoutSet, err := objectsstorage.LoadLayouts(ctx, config.DB, userID, requestL.Offset, requestL.Limit, requestL.Size, variant, requestL.CountImages, requestL.IsFavourite, isCover)
	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}
	

	rw.WriteHeader(http.StatusOK)
	resp["response"] = layoutSet
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func AdminCreateLayout(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]models.ResponseCreatedLayout)
	var LayoutObj models.Layout
	var lID uint
	var rLayout models.ResponseCreatedLayout

	err := json.NewDecoder(r.Body).Decode(&LayoutObj)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, err)
		return
	}

	defer r.Body.Close()
	// Create a new validator instance
    validate := validator.New()

    // Validate the User struct
    err = validate.Struct(LayoutObj)
    if err != nil {
        // Validation failed, handle the error
		handlersfunc.HandleValidationError(rw, err)
        return
    }
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	log.Printf("Create admin layout for user")
	lID, err = objectsstorage.AddAdminLayout(ctx, config.DB, LayoutObj)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}


	rw.WriteHeader(http.StatusOK)
	rLayout.LayoutID = lID
	resp["response"] = rLayout
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func AdminDeleteLayout(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]uint)
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	layoutID := uint(aByteToInt)
	defer r.Body.Close()
	
	err = objectsstorage.AdminDeleteLayout(ctx, config.DB, layoutID)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}

	rw.WriteHeader(http.StatusOK)
	resp["response"] = 1
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func FavourLayout(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]uint)
	var LayoutObj models.PersonalisedObject

	err := json.NewDecoder(r.Body).Decode(&LayoutObj)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, err)
		return
	}
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	LayoutObj.ObjectID = uint(aByteToInt)
	defer r.Body.Close()
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	userID := handlersfunc.UserIDContextReader(r)
	log.Printf("Favour layout for user %d", userID)
	err = objectsstorage.FavourLayout(ctx, config.DB, LayoutObj, userID)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}


	rw.WriteHeader(http.StatusOK)
	resp["response"] = 1
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func LoadProjects(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]models.ResponseProjects)
	defer r.Body.Close()
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	userID := handlersfunc.UserIDContextReader(r)
	log.Printf("Load projects of the user %d", userID)
	var requestT models.RequestTemplate
	myUrl, _ := url.Parse(r.URL.String())	
	params, _ := url.ParseQuery(myUrl.RawQuery)

	tOffset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	tLimit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	requestT.Offset = uint(tOffset)
	requestT.Limit = uint(tLimit)
	var lo models.LimitOffset
	if _, ok := params["offset"]; ok {
		lo.Offset = &requestT.Offset
	}
	if requestT.Limit != 0 {
		lo.Limit = &requestT.Limit
	}
	validate := validator.New()

    // Validate the User struct
    err = validate.Struct(lo)
    if err != nil {
        // Validation failed, handle the error
		handlersfunc.HandleValidationError(rw, err)
        return
    }

	projects, err := projectstorage.RetrieveUserProjects(ctx, config.DB, userID, requestT.Offset, requestT.Limit)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}
	log.Println(projects)

	rw.WriteHeader(http.StatusOK)
	resp["response"] = projects
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func AdminLoadProjects(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]models.ResponseProjects)
	defer r.Body.Close()
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	userID := handlersfunc.UserIDContextReader(r)
	log.Printf("Load projects of the admin %d", userID)
	var requestT models.RequestTemplate
	myUrl, _ := url.Parse(r.URL.String())	
	params, _ := url.ParseQuery(myUrl.RawQuery)

	tOffset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	tLimit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	requestT.Offset = uint(tOffset)
	requestT.Limit = uint(tLimit)
	
	var lo models.LimitOffset
	if _, ok := params["offset"]; ok {
		lo.Offset = &requestT.Offset
	}
	if requestT.Limit != 0 {
		lo.Limit = &requestT.Limit
	}
	validate := validator.New()

    // Validate the User struct
    err = validate.Struct(lo)
    if err != nil {
        // Validation failed, handle the error
		handlersfunc.HandleValidationError(rw, err)
        return
    }
	projects, err := projectstorage.RetrieveAdminProjects(ctx, config.DB, userID, requestT.Offset, requestT.Limit)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}

	rw.WriteHeader(http.StatusOK)
	resp["response"] = projects
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func SavePage(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]uint)
	var savedPages models.RequestSavePages
	err := json.NewDecoder(r.Body).Decode(&savedPages)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, err)
		return
	}
	
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	projectID := uint(aByteToInt)
	defer r.Body.Close()
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	log.Printf("Saving page for project user %d",projectID)
	for _, page := range savedPages.Pages {
		// add check for missing page
		if !projectstorage.CheckPage(ctx, config.DB, page.PageID, projectID) {
				handlersfunc.HandleMissingPageError(rw)
				return
		}

		err = projectstorage.SavePage(ctx, config.DB, page)
		if err != nil {
				handlersfunc.HandleDatabaseServerError(rw)
				return
		}
		
		err = projectstorage.SavePagePhotos(ctx, config.DB, page.PageID, page.UsedPhotoIDs)
		if err != nil {
				handlersfunc.HandleDatabaseServerError(rw)
				return
		}
	}
	

	rw.WriteHeader(http.StatusOK)
	resp["response"] = 1
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func UpdateProjectSpine(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]uint)
	var savedSpine models.SavedSpine
	err := json.NewDecoder(r.Body).Decode(&savedSpine)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, err)
		return
	}
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	projectID := uint(aByteToInt)
	defer r.Body.Close()
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	userID := handlersfunc.UserIDContextReader(r)
	log.Printf("Load project %d for user %d",projectID, userID)
	checkExists := orderstorage.CheckProject(ctx, config.DB, projectID)
	if !checkExists {
			handlersfunc.HandleMissingProjectError(rw)
			return
	}


	userCheck := userstorage.CheckUserHasProject(ctx, config.DB, userID, projectID)

	if !userCheck {
		handlersfunc.HandlePermissionError(rw)
		return
	}
	log.Printf("Saving spine for project %d",projectID)
	err = projectstorage.SaveSpine(ctx, config.DB, savedSpine, projectID)
	if err != nil {
				handlersfunc.HandleDatabaseServerError(rw)
				return
	}
	
	rw.WriteHeader(http.StatusOK)
	resp["response"] = 1
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func UpdateTemplateSpine(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]uint)
	var savedSpine models.SavedSpine
	err := json.NewDecoder(r.Body).Decode(&savedSpine)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, err)
		return
	}
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	projectID := uint(aByteToInt)
	defer r.Body.Close()
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	checkExists := projectstorage.CheckTemplate(ctx, config.DB, projectID)
	if !checkExists {
		handlersfunc.HandleMissingTemplateError(rw)
		return
	}
	err = projectstorage.SaveTemplateSpine(ctx, config.DB, savedSpine, projectID)
	if err != nil {
				handlersfunc.HandleDatabaseServerError(rw)
				return
	}
	
	rw.WriteHeader(http.StatusOK)
	resp["response"] = 1
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func AddProjectPages(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string][]models.OrderPage)
	var newPages models.RequestAddPage
	var addedPages []models.OrderPage
	err := json.NewDecoder(r.Body).Decode(&newPages)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, err)
		return
	}
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	projectID := uint(aByteToInt)
	defer r.Body.Close()
	log.Printf("Add new pages for project %d", projectID)
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()

	for _, page := range newPages.Pages {

		var addedPage models.OrderPage
		if !projectstorage.CheckPagesRange(ctx, config.DB, page.Sort, projectID, false) {
				handlersfunc.HandleMissingPageError(rw)
				return
		}
		
		
		addedPage, err = projectstorage.AddProjectPage(ctx, config.DB, projectID, page.Sort, false)
		if err != nil {
			handlersfunc.HandleDatabaseServerError(rw)
			return
		}
		if page.CloneID != 0 {
			// add check for missing page
			if !projectstorage.CheckPage(ctx, config.DB, page.CloneID, projectID) {
					handlersfunc.HandleMissingPageError(rw)
					return
			}
			if projectstorage.CheckCoverPage(ctx, config.DB, page.CloneID){
				handlersfunc.HandleCoverPageError(rw)
				return
			}
			err = projectstorage.DuplicatePage(ctx, config.DB, page.CloneID, addedPage.PageID)
			if err != nil {
				handlersfunc.HandleDatabaseServerError(rw)
				return
			}
		}
		
		addedPages = append(addedPages, addedPage)
	}
	rw.WriteHeader(http.StatusOK)
	resp["response"] = addedPages
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func AddTemplatePages(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string][]models.OrderPage)
	var newPages models.RequestAddPage
	var addedPages []models.OrderPage
	err := json.NewDecoder(r.Body).Decode(&newPages)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, err)
		return
	}
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	projectID := uint(aByteToInt)
	defer r.Body.Close()
	log.Printf("Add new pages for project %d", projectID)
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()

	for _, page := range newPages.Pages {

		var addedPage models.OrderPage
		if page.CloneID != 0{
			if !projectstorage.CheckPagesRange(ctx, config.DB, page.Sort, projectID, true) {
				handlersfunc.HandleMissingPageError(rw)
				return
			}
		}
		addedPage, err = projectstorage.AddProjectPage(ctx, config.DB, projectID, page.Sort, true)
		if err != nil {
			handlersfunc.HandleDatabaseServerError(rw)
			return
		}
		if page.CloneID != 0 {
			// add check for missing page
			if !projectstorage.CheckPage(ctx, config.DB, page.CloneID, projectID) {
				handlersfunc.HandleMissingPageError(rw)
				return
			}
			err = projectstorage.DuplicatePage(ctx, config.DB, page.CloneID, addedPage.PageID)
			if err != nil {
				handlersfunc.HandleDatabaseServerError(rw)
				return
			}
		}
		addedPages = append(addedPages, addedPage)
	}

	rw.WriteHeader(http.StatusOK)
	resp["response"] = addedPages
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func DeletePages(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]uint)
	var deletePages models.RequestDeletePage
	err := json.NewDecoder(r.Body).Decode(&deletePages)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	projectID := uint(aByteToInt)
	defer r.Body.Close()

	for _, pageID := range deletePages.PageIDs {
		err = projectstorage.DeletePage(ctx, config.DB, pageID, projectID, false)

		if err != nil {
			handlersfunc.HandleMissingPageError(rw)
			return
		}
	}
	rw.WriteHeader(http.StatusOK)
	resp["response"] = 1
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func DeleteTemplatePages(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]uint)
	var deletePages models.RequestDeletePage
	err := json.NewDecoder(r.Body).Decode(&deletePages)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	projectID := uint(aByteToInt)
	defer r.Body.Close()

	for _, pageID := range deletePages.PageIDs {
		err = projectstorage.DeletePage(ctx, config.DB, pageID, projectID, true)

		if err != nil {
			handlersfunc.HandleDatabaseServerError(rw)
			return
		}
	}
	rw.WriteHeader(http.StatusOK)
	resp["response"] = 1
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func ReorderPages(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]uint)
	var reorderPages models.RequestReorderPage
	err := json.NewDecoder(r.Body).Decode(&reorderPages)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	projectID := uint(aByteToInt)
	defer r.Body.Close()
	var sortNumbers []uint
	log.Printf("Reorder pages for project %d",projectID)

	for _, page := range reorderPages.Pages {
		sortNumbers = append(sortNumbers, page.Sort)
	}

	lenSlice := make([]int, len(sortNumbers))
	for _, v := range lenSlice {
		if !slices.Contains(sortNumbers, uint(v)) && v != 0 && v != len(lenSlice) {
			handlersfunc.HandleWrongOrderError(rw)
			return
		}
		
	}
	if !projectstorage.CheckAllPagesPassed(ctx, config.DB, uint(len(sortNumbers)), projectID, false){
		handlersfunc.HandleNotAllPagesPassedError(rw)
		return
	}

	for _, page := range reorderPages.Pages {

		if !projectstorage.CheckPage(ctx, config.DB, page.PageID, projectID){
			handlersfunc.HandleMissingPageError(rw)
			return
		}
		if projectstorage.CheckCoverPage(ctx, config.DB, page.PageID){
			handlersfunc.HandleCoverPageError(rw)
			return
		}
		err = projectstorage.ReorderPage(ctx, config.DB, page.PageID, projectID, page.Sort)

		if err != nil {
			handlersfunc.HandleDatabaseServerError(rw)
			return
		}
	}
	rw.WriteHeader(http.StatusOK)
	resp["response"] = 1
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func ReorderTemplatePages(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]uint)
	var reorderPages models.RequestReorderPage
	err := json.NewDecoder(r.Body).Decode(&reorderPages)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	projectID := uint(aByteToInt)
	defer r.Body.Close()
	var sortNumbers []uint

	for _, page := range reorderPages.Pages {
		sortNumbers = append(sortNumbers, page.Sort)
	}

	lenSlice := make([]int, len(sortNumbers))
	for _, v := range lenSlice {
		if !slices.Contains(sortNumbers, uint(v)) && v != 0 && v != len(lenSlice) {
			handlersfunc.HandleWrongOrderError(rw)
			return
		}
		
	}

	if !projectstorage.CheckAllPagesPassed(ctx, config.DB, uint(len(sortNumbers)), projectID, true){
		handlersfunc.HandleNotAllPagesPassedError(rw)
		return
	}

	for _, page := range reorderPages.Pages {

		if !projectstorage.CheckPage(ctx, config.DB, page.PageID, projectID){
			handlersfunc.HandleMissingPageError(rw)
			return
		}
		if projectstorage.CheckCoverPage(ctx, config.DB, page.PageID){
			handlersfunc.HandleCoverPageError(rw)
			return
		}
		err = projectstorage.ReorderPage(ctx, config.DB, page.PageID, projectID, page.Sort)

		if err != nil {
			handlersfunc.HandleDatabaseServerError(rw)
			return
		}
	}
	rw.WriteHeader(http.StatusOK)
	resp["response"] = 1
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func LoadTemplates(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]models.ResponseTemplates)
	
	var requestT models.RequestTemplate
	myUrl, _ := url.Parse(r.URL.String())	
	params, _ := url.ParseQuery(myUrl.RawQuery)

	tOffset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	tLimit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	size := r.URL.Query().Get("size")
	var lo models.LimitOffsetSize

    if size != "" {
		requestT.Size = strings.ToUpper(r.URL.Query().Get("size"))
		lo.Size = &requestT.Size
	}
	category := r.URL.Query().Get("category")

    if category != "" {
		requestT.Category = strings.ToUpper(r.URL.Query().Get("category"))
		lo.Category = &requestT.Category
	}
	variant := r.URL.Query().Get("variant")
	if variant != "" {
		lo.Variant = &variant
	}

	requestT.Offset = uint(tOffset)
	requestT.Limit = uint(tLimit)
	
	if _, ok := params["offset"]; ok {
		lo.Offset = &requestT.Offset
	}
	if requestT.Limit != 0 {
		lo.Limit = &requestT.Limit
	}
	validate := validator.New()

    // Validate the User struct
    err = validate.Struct(lo)
    if err != nil {
        // Validation failed, handle the error
		handlersfunc.HandleValidationError(rw, err)
        return
    }
	
	
	
	defer r.Body.Close()
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	log.Printf("Retrieving templates")

	templates, err := projectstorage.RetrieveTemplates(ctx, config.DB, requestT.Offset, requestT.Limit, requestT.Category, requestT.Size, variant, "PUBLISHED")

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}

	rw.WriteHeader(http.StatusOK)
	resp["response"] = templates
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func AdminLoadTemplates(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]models.ResponseTemplates)
	
	var requestT models.RequestTemplate
	
	category := r.URL.Query().Get("category")
	size := r.URL.Query().Get("size")
	status := r.URL.Query().Get("status")
	var lo models.LimitOffsetSizeAdmin
	if status != "" {
		requestT.Status = strings.ToUpper(r.URL.Query().Get("status"))
	}

    if category != "" {
		requestT.Category = strings.ToUpper(r.URL.Query().Get("category"))
		lo.Category = &requestT.Category
	}
	if size != "" {
		requestT.Size = strings.ToUpper(r.URL.Query().Get("size"))
		lo.Size = &requestT.Size
	}
	
	myUrl, _ := url.Parse(r.URL.String())	
	params, _ := url.ParseQuery(myUrl.RawQuery)

	tOffset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	tLimit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	requestT.Offset = uint(tOffset)
	requestT.Limit = uint(tLimit)
	if _, ok := params["offset"]; ok {
		lo.Offset = &requestT.Offset
	}
	if requestT.Limit != 0 {
		lo.Limit = &requestT.Limit
	}

	validate := validator.New()

    // Validate the User struct
    err = validate.Struct(lo)
    if err != nil {
        // Validation failed, handle the error
		handlersfunc.HandleValidationError(rw, err)
        return
    }

	defer r.Body.Close()
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	log.Printf("Retrieving templates")

	templates, err := projectstorage.RetrieveAdminTemplates(ctx, config.DB, requestT.Offset, requestT.Limit, requestT.Category, requestT.Size, requestT.Status)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}

	rw.WriteHeader(http.StatusOK)
	resp["response"] = templates
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func CreateTemplate(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]models.ResponseCreatedTemplate)
	var TemplateObj models.NewTemplateObj
	var tID uint
	var rTemplate models.ResponseCreatedTemplate
	
	err := json.NewDecoder(r.Body).Decode(&TemplateObj)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, err)
		return
	}

	defer r.Body.Close()
	// Create a new validator instance
    validate := validator.New()

    // Validate the User struct
    err = validate.Struct(TemplateObj)
    if err != nil {
        // Validation failed, handle the error
		handlersfunc.HandleValidationError(rw, err)
        return
    }
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	
	tID, err = projectstorage.CreateTemplate(ctx, config.DB, TemplateObj.Name, TemplateObj.Size, TemplateObj.Category, TemplateObj.Variant)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}

	rw.WriteHeader(http.StatusOK)
	rTemplate.TemplateID = tID
	resp["response"] = rTemplate
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func DuplicateTemplate(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]uint)
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	templateID := uint(aByteToInt)
	defer r.Body.Close()
	checkExists := projectstorage.CheckTemplate(ctx, config.DB, templateID)
	if !checkExists {
		handlersfunc.HandleMissingTemplateError(rw)
		return
	}
	
	_, err = projectstorage.DuplicateTemplate(ctx, config.DB, templateID)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}

	rw.WriteHeader(http.StatusOK)
	resp["response"] = 1
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func UpdateTemplate(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]uint)
	var TemplateObj models.NewTemplateObj
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	templateID := uint(aByteToInt)
	
	err := json.NewDecoder(r.Body).Decode(&TemplateObj)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, err)
		return
	}

	defer r.Body.Close()
	// Create a new validator instance
    validate := validator.New()

    // Validate the User struct
    err = validate.Struct(TemplateObj)
    if err != nil {
        // Validation failed, handle the error
		handlersfunc.HandleValidationError(rw, err)
        return
    }
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	checkExists := projectstorage.CheckTemplate(ctx, config.DB, templateID)
	if !checkExists {
		handlersfunc.HandleMissingTemplateError(rw)
		return
	}
	
	_, err = projectstorage.UpdateTemplate(ctx, config.DB, templateID, TemplateObj.Name, TemplateObj.Category)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}

	rw.WriteHeader(http.StatusOK)
	resp["response"] = 1
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func ShareLink(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]uint)
	var ViewerObj models.Viewer
	var OwnerObj models.UserInfo

	err := json.NewDecoder(r.Body).Decode(&ViewerObj)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, err)
		return
	}
	// Create a new validator instance
    validate := validator.New()

    // Validate the User struct
    err = validate.Struct(ViewerObj)
    if err != nil {
        // Validation failed, handle the error
		handlersfunc.HandleValidationError(rw, err)
        return
    }
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	pID := uint(aByteToInt)
	defer r.Body.Close()
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	checkExists := orderstorage.CheckProject(ctx, config.DB, pID)
	log.Println(checkExists)
	if checkExists == false {
			handlersfunc.HandleMissingProjectError(rw)
			return
	}

	userID := handlersfunc.UserIDContextReader(r)
	userCheck := userstorage.CheckUserHasProject(ctx, config.DB, userID, pID)

	if userCheck == false {
		handlersfunc.HandlePermissionError(rw)
		return
	}
	OwnerObj, err = userstorage.GetUserData(ctx, config.DB, userID)
	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}
	log.Printf("Sharing preview link for project %d",pID)

	previewLink := "https://front.memoryprint.dev.startup-it.ru/preview/" + mux.Vars(r)["id"]
	// Send email with the link
	from := "support@memoryprint.ru"
	to := []string{ViewerObj.Email}
	subject := "=?koi8-r?B?8yDXwc3JINDPxMXMyczJ09gg09PZzMvPyiDOwSDGz9TPy87Jx9Uh==?="
	mailType := emailutils.MailViewerInvitation
	mailData := &emailutils.MailData{
		Username: ViewerObj.Name,
		OwnerName: OwnerObj.Name,
		OwnerEmail: OwnerObj.Email,
		ShareLink: previewLink,
	}

	ms := &emailutils.SGMailService{config.YandexApiKey}
	mailReq := emailutils.NewMail(from, to, subject, mailType, mailData)
	err = emailutils.SendMail(mailReq, ms)
	if err != nil {
		log.Printf("unable to send shared project mail", "error", err)
		handlersfunc.HandleMailSendError(rw)
		return
	}

	err = projectstorage.AddViewer(ctx, config.DB, pID, ViewerObj.Email)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}

	rw.WriteHeader(http.StatusOK)
	resp["response"] = 1
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func AdminCreatePrices(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]uint)
	var PricesObj []models.Price
	


	err := json.NewDecoder(r.Body).Decode(&PricesObj)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, err)
		return
	}

	defer r.Body.Close()
	
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	log.Printf("Create new prices")
	err = objectsstorage.AddPrices(ctx, config.DB, PricesObj)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}


	rw.WriteHeader(http.StatusOK)
	resp["response"] = 1
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func AdminDeletePrices(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]uint)

	defer r.Body.Close()
	
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	log.Printf("Delete prices")
	err = objectsstorage.DeletePrices(ctx, config.DB)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}


	rw.WriteHeader(http.StatusOK)
	resp["response"] = 1
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func LoadPrices(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]models.ResponsePrice)
	var priceObj models.ResponsePrice
	
	defer r.Body.Close()
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	log.Printf("Retrieving prices")

	prices, err := objectsstorage.RetrievePrices(ctx, config.DB)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}

	rw.WriteHeader(http.StatusOK)
	priceObj.Prices = prices
	resp["response"] = priceObj
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func AdminCreateCover(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]uint)
	var CoverObj models.Colour


	err := json.NewDecoder(r.Body).Decode(&CoverObj)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, err)
		return
	}

	defer r.Body.Close()
	
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	log.Printf("Create new cover")
	err = objectsstorage.AddCover(ctx, config.DB, CoverObj)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}


	rw.WriteHeader(http.StatusOK)
	resp["response"] = 1
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func AdminDeleteCover(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]uint)
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	coverID := uint(aByteToInt)
	defer r.Body.Close()
	
	err = objectsstorage.AdminDeleteCover(ctx, config.DB, coverID)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}

	rw.WriteHeader(http.StatusOK)
	resp["response"] = 1
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func LoadColours(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]models.ResponseColour)
	var coverObj models.ResponseColour
	
	defer r.Body.Close()
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	log.Printf("Retrieving covers")

	covers, err := objectsstorage.RetrieveCovers(ctx, config.DB)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}

	rw.WriteHeader(http.StatusOK)
	coverObj.Colors = covers
	resp["response"] = coverObj
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func UpdateCover(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]uint)
	var CoverObj models.UpdateCover

	err := json.NewDecoder(r.Body).Decode(&CoverObj)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, err)
		return
	}
	defer r.Body.Close()
	// Create a new validator instance
    validate := validator.New()

    // Validate the User struct
    err = validate.Struct(CoverObj)
    if err != nil {
        // Validation failed, handle the error
		handlersfunc.HandleValidationError(rw, err)
        return
    }
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	projectID := uint(aByteToInt)
	userID := handlersfunc.UserIDContextReader(r)
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	checkExists := orderstorage.CheckProject(ctx, config.DB, projectID)
	if !checkExists {
			handlersfunc.HandleMissingProjectError(rw)
			return
	}

	userCheck := userstorage.CheckUserHasProject(ctx, config.DB, userID, projectID)

	if userCheck == false {
		handlersfunc.HandlePermissionError(rw)
		return
	}
	
	projectBool := projectstorage.CheckProjectPublished(ctx, config.DB, projectID)
	if projectBool == false {
		handlersfunc.HandleProjectNotPublished(rw)
		return
	}
	projectBool = projectstorage.CheckProjectNotCompleted(ctx, config.DB, projectID)
	if projectBool == false {
		handlersfunc.HandleOrderCompleted(rw)
		return
	}
    if CoverObj.Cover == "LEATHERETTE" {
		leatherBool := projectstorage.CheckLeatherID(ctx, config.DB, CoverObj.LeatherID)
		if leatherBool == false {
				handlersfunc.HandleMissingLeatherID(rw)
				return
		}
	} else {
		coverBool := projectstorage.CheckHardCover(ctx, config.DB, projectID)
		if coverBool == false {
				handlersfunc.HandleCoverBoolError(rw)
				return
		}
	}

	log.Printf("Update project cover for project %d",projectID)
	err = projectstorage.UpdateCover(ctx, config.DB, projectID, CoverObj)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}


	rw.WriteHeader(http.StatusOK)
	resp["response"] = 1
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func UpdateSurface(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]uint)
	var SurfaceObj models.UpdateSurface

	err := json.NewDecoder(r.Body).Decode(&SurfaceObj)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, err)
		return
	}
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	projectID := uint(aByteToInt)

	defer r.Body.Close()
	// Create a new validator instance
    validate := validator.New()

    // Validate the User struct
    err = validate.Struct(SurfaceObj)
    if err != nil {
        // Validation failed, handle the error
		handlersfunc.HandleValidationError(rw, err)
        return
    }
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()

	userID := handlersfunc.UserIDContextReader(r)
	checkExists := orderstorage.CheckProject(ctx, config.DB, projectID)
	if !checkExists {
			handlersfunc.HandleMissingProjectError(rw)
			return
	}
	userCheck := userstorage.CheckUserHasProject(ctx, config.DB, userID, projectID)

	if userCheck == false {
		handlersfunc.HandlePermissionError(rw)
		return
	}
	projectBool := projectstorage.CheckProjectPublished(ctx, config.DB, projectID)
	if projectBool == false {
		handlersfunc.HandleProjectNotPublished(rw)
		return
	}
	projectBool = projectstorage.CheckProjectNotCompleted(ctx, config.DB, projectID)
	if projectBool == false {
		handlersfunc.HandleOrderCompleted(rw)
		return
	}
	log.Printf("Update project surface for project %d",projectID)
	err = projectstorage.UpdateSurface(ctx, config.DB, projectID, SurfaceObj)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}


	rw.WriteHeader(http.StatusOK)
	resp["response"] = 1
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func pickUnusedPort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	port := l.Addr().(*net.TCPAddr).Port
	if err := l.Close(); err != nil {
		return 0, err
	}
	return port, nil
}

func GetBrowserPath(browser string) string {
	if _, err := os.Stat(browser); err != nil {
		path, err := exec.LookPath(browser)
		if err != nil {
			panic("Browser binary path not found")
		}
		return path
	}
	return browser
}

func GenerateCreatingImageLinks(ctx context.Context, storeDB *pgxpool.Pool) {

	ticker := time.NewTicker(config.UpdateInterval*2)
	//var err error
	

	jobCh := make(chan uint)
	for i := 0; i < config.WorkersCount; i++ {
		go func() {
			for job := range jobCh {
				var images []models.ExportPage
				checkBool := imagehandlers.CheckProjectFolder(job) 
				if checkBool {
					log.Println("Photobook already printed")
				} else {
					log.Println("Trying to print images..")
					port, err := pickUnusedPort()

					service, err := selenium.NewChromeDriverService("/usr/local/bin/chromedriver", port)
					if err != nil {
						log.Printf("Error happened when creating browser. Err: %s", err)
						continue
					}
					defer service.Stop()
					caps := selenium.Capabilities{}
					caps.AddChrome(chrome.Capabilities{Args: []string{
					"--headless=new", 
					"--no-sandbox",
					"--enable-automation",
					"--enable-javascript", 
					"--disable-dev-shm-usage", // comment out this line for testing
					}})

					// create a new remote client with the specified options
					driver, err := selenium.NewRemote(caps, fmt.Sprintf("http://localhost:%d/wd/hub", port))
					if err != nil {
						log.Printf("Error happened when creating driver. Err: %s", err)
						continue
					}
					driver.SetPageLoadTimeout(210*time.Second)

				
					images, err = projectstorage.GenerateImages(ctx, storeDB, job, driver)
					if err != nil {
						log.Printf("Error happened when updating paid orders. Err: %s", err)
						continue
					}
					var stringImages []string
					for _, v := range images {
						stringImages = append(stringImages, v.PreviewImageLink)
					}
					if !slices.Contains(stringImages, "") {
						var variant string
						log.Println("Trying to create folder")
						log.Println(job)
						variant, err = projectstorage.LoadProjectVariant(ctx, storeDB, job) 

						imagehandlers.CreateProjectFolder(images, job)
						err = imagehandlers.CreatePrintVersion(job, images , variant)
						if err != nil {
							log.Printf("Error happened when creating print version. Err: %s", err)
							continue
						}
					}
				}
				
			}
		}()
	}

	for range ticker.C {
		var projectIDs []uint
		projectIDs = append(projectIDs,452)
		cmd := exec.Command("bash", "-c", "pkill chrome")
		stdout, err := cmd.Output()
		
		if err != nil {
			log.Printf("Error happened when killing chrome processes. Err: %s", err)
		}

		log.Println(string(stdout))
		cmd = exec.Command("bash", "-c", "pkill chromedriver")
		stdout, err = cmd.Output()
		
		if err != nil {
			log.Printf("Error happened when killing chromedriver processes. Err: %s", err)
			
		}

		log.Println(string(stdout))
		// projectIDs, err = projectstorage.LoadPublishedProjects(ctx, storeDB)
		//if err != nil {
		//	log.Printf("Error happened when retrieving published projects. Err: %s", err)
		//	continue
		//}
		
		log.Println(projectIDs)
		
		for _, project := range projectIDs {
			jobCh <- project

		}

	}
}


func DuplicateLayout(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]uint)
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	
	err = objectsstorage.DuplicateLayout(ctx, config.DB)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}

	rw.WriteHeader(http.StatusOK)
	resp["response"] = 1
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func AddExtraPageTempFix(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]uint)
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	
	err = projectstorage.AddExtraPageTempFix(ctx, config.DB)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}

	rw.WriteHeader(http.StatusOK)
	resp["response"] = 1
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}
