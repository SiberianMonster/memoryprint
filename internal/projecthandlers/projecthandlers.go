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
	"strings"

	"github.com/SiberianMonster/memoryprint/internal/config"
	"github.com/SiberianMonster/memoryprint/internal/models"
	"github.com/SiberianMonster/memoryprint/internal/projectstorage"
	"github.com/SiberianMonster/memoryprint/internal/objectsstorage"
	"github.com/SiberianMonster/memoryprint/internal/handlersfunc"
	"github.com/go-playground/validator/v10"
	"golang.org/x/exp/slices"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

var err error
var resp map[string]string


func UserLoadPhotos(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]models.ResponsePhotos)
	var photoParams models.RequestPhotos
	var rPhotos models.ResponsePhotos
	defer r.Body.Close()
	rOffset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	rLimit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	photoParams.Offset = uint(rOffset)
	photoParams.Limit = uint(rLimit)

	
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	userID := handlersfunc.UserIDContextReader(r)
	log.Printf("Load photos of the user %d", userID)
	rPhotos, err := objectsstorage.RetrieveUserPhotos(ctx, config.DB, userID, photoParams.Offset, photoParams.Limit)

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
	pID, err = objectsstorage.AddPhoto(ctx, config.DB, photoParams.Link, userID)

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
	pID, err = projectstorage.CreateProject(ctx, config.DB, userID, ProjectObj, 0)

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

func LoadProject(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]models.SavedProjectObj)
	var retrievedProject models.SavedProjectObj
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	projectID := uint(aByteToInt)
	defer r.Body.Close()

	userID := handlersfunc.UserIDContextReader(r)
	log.Printf("Load project %d for user %d",projectID, userID)

	userCheck := projectstorage.CheckUserHasProject(ctx, config.DB, userID, projectID)

	if !userCheck {
		handlersfunc.HandlePermissionError(rw)
		return
	}
	
	retrievedProject.Project, err = projectstorage.LoadProject(ctx, config.DB, projectID)
	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}
	log.Println(retrievedProject)
	retrievedProject.Pages, err = projectstorage.RetrieveProjectPages(ctx, config.DB, projectID, false)
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
	rOffset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	rLimit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	requestB.Offset = uint(rOffset)
	requestB.Limit = uint(rLimit)
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

	rOffset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	rLimit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	requestD.Offset = uint(rOffset)
	requestD.Limit = uint(rLimit)
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

	rOffset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	rLimit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	rCountImages, _ := strconv.Atoi(r.URL.Query().Get("countimages"))
	requestL.Offset = uint(rOffset)
	requestL.Limit = uint(rLimit)
	requestL.CountImages = uint(rCountImages)
	requestL.Size = strings.ToUpper(r.URL.Query().Get("size"))
	requestL.IsFavourite, _ = strconv.ParseBool(r.URL.Query().Get("isfavourite"))

	defer r.Body.Close()
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	userID := handlersfunc.UserIDContextReader(r)
	log.Printf("Load layouts for user %d", userID)
	
	
	
	layoutSet, err := objectsstorage.LoadLayouts(ctx, config.DB, userID, requestL.Offset, requestL.Limit, requestL.Size, requestL.CountImages, requestL.IsFavourite)
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

	resp := make(map[string][]models.ResponseProject)
	defer r.Body.Close()
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	userID := handlersfunc.UserIDContextReader(r)
	log.Printf("Load projects of the user %d", userID)
	projects, err := projectstorage.RetrieveUserProjects(ctx, config.DB, userID)

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
	log.Println(savedPages)
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	projectID := uint(aByteToInt)
	defer r.Body.Close()
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
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

	tOffset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	tLimit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	requestT.Offset = uint(tOffset)
	requestT.Limit = uint(tLimit)
	
	category := r.URL.Query().Get("category")

    if category != "" {
		requestT.Category = strings.ToUpper(r.URL.Query().Get("category"))
	}
	size := r.URL.Query().Get("size")

    if size != "" {
		requestT.Size = strings.ToUpper(r.URL.Query().Get("size"))
	}
	defer r.Body.Close()
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	log.Printf("Retrieving templates")

	templates, err := projectstorage.RetrieveTemplates(ctx, config.DB, requestT.Offset, requestT.Limit, requestT.Category, requestT.Size)

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
	
	tID, err = projectstorage.CreateTemplate(ctx, config.DB, TemplateObj.Name, TemplateObj.Size, TemplateObj.Category)

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