//MemoryPrint Api: 
// version: 0.1 
// title: MemoryPrint Api 
// Schemes: http, https 
// Host: 
// BasePath: /api/v1 
// Consumes: 
//  - application/json 
// Produces: 
//  - application/json 
//   SecurityDefinitions: 
//    Bearer: 
//     type: apiKey 
//     name: Authorization 
//     in: header 
//   swagger:meta 
package main

import (
	"context"
	"errors"
	"flag"
	"github.com/SiberianMonster/memoryprint/internal/authhandlers"
	"github.com/SiberianMonster/memoryprint/internal/config"
	"github.com/SiberianMonster/memoryprint/internal/initstorage"
	"github.com/SiberianMonster/memoryprint/internal/imagehandlers"
	"github.com/SiberianMonster/memoryprint/internal/userhandlers"
	"github.com/SiberianMonster/memoryprint/internal/projecthandlers"
	"github.com/SiberianMonster/memoryprint/internal/middleware"
	"github.com/gorilla/mux"
	// "github.com/rs/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"gopkg.in/ini.v1"
	"time"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

var err error
var host, connStr, accrualStr, adminEmail, yandexKey, timewebToken, balaToken, imageHost *string
var db *pgxpool.Pool

func init() {

	inidata, err := ini.Load("config.ini")
	if err != nil {
		log.Printf("Fail to read ini file: %v", err)
		os.Exit(1)
	}
	section := inidata.Section("section")

	host = config.GetEnv("RUN_ADDRESS", flag.String("a", section.Key("host").String(), "SERVER HOST RUN_ADDRESS"))
	connStr = config.GetEnv("DATABASE_URI", flag.String("d", section.Key("connstr").String(), "DATA STORAGE DATABASE_URI"))
	adminEmail = config.GetEnv("ADMIN_EMAIL", flag.String("am", "support@memoryprint.ru", "ADMIN_EMAIL"))
	yandexKey = config.GetEnv("YANDEX_PASSWORD", flag.String("yp", section.Key("yandexkey").String(), "YANDEX_PASSWORD"))
	timewebToken = config.GetEnv("TW_TOKEN", flag.String("tw", section.Key("timeweb").String(), "TW_TOKEN"))
	balaToken = config.GetEnv("BL_TOKEN", flag.String("bl", section.Key("bala").String(), "BL_TOKEN"))
	imageHost = config.GetEnv("IMAGE_HOST", flag.String("img", section.Key("imagehost").String(), "IMAGE_HOST"))

}


func main() {

	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), config.ContextDBTimeout*time.Second)
	defer cancel()

	// log to custom file
    LOG_FILE := "memoryprint_log.log"
    // open log file
    logFile, err := os.OpenFile(LOG_FILE, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
    if err != nil {
		log.Println("log panic")
        log.Panic(err)
    }
    defer logFile.Close()

    // Set log out put and enjoy :)
    log.SetOutput(logFile)

    // optional: log date-time, filename, and line number
    log.SetFlags(log.Lshortfile | log.LstdFlags)

	if *connStr == "" {
		log.Fatalf("Database credentials were not passed")
	}

	config.DB, _ = initstorage.SetUpDBConnection(ctx, connStr)
	defer config.DB.Close()

	config.AdminEmail = *adminEmail
	config.YandexApiKey = *yandexKey
	config.TimewebToken = *timewebToken
	config.BalaToken = *balaToken
	config.ImageHost = *imageHost

	router := mux.NewRouter()
	router.Use(middleware.MiddlewareCORSHeaders)
	noAuthRouter := router.MatcherFunc(func(r *http.Request, rm *mux.RouteMatch) bool {
		return true
	}).Subrouter()

	authRouter := router.MatcherFunc(func(r *http.Request, rm *mux.RouteMatch) bool {
		return true
	}).Subrouter()

	adminRouter := router.MatcherFunc(func(r *http.Request, rm *mux.RouteMatch) bool {
		return true
	}).Subrouter()
	
	noAuthRouter.HandleFunc("/api/v1/auth/signup", userhandlers.Register).Methods("POST","OPTIONS")
	noAuthRouter.HandleFunc("/api/v1/auth/login", userhandlers.Login).Methods("POST","OPTIONS")
	noAuthRouter.HandleFunc("/api/v1/load-templates", projecthandlers.LoadTemplates).Methods("GET","OPTIONS")
	noAuthRouter.HandleFunc("/api/v1/load-template/{id}", projecthandlers.LoadTemplate).Methods("GET","OPTIONS")
	
	noAuthRouter.HandleFunc("/api/v1/greet", authhandlers.Greet).Methods("GET","OPTIONS")
	noAuthRouter.HandleFunc("/api/v1/auth/restore", authhandlers.GenerateTempPass).Methods("POST","OPTIONS")
	noAuthRouter.HandleFunc("/api/v1/change-user-status/{id}", userhandlers.MakeUserAdmin).Methods("GET","OPTIONS")
	//noAuthRouter.HandleFunc("/api/v1/verify/password-reset", userhandlers.VerifyPasswordReset)


	authRouter.Use(middleware.MiddlewareValidateAccessToken)
	adminRouter.Use(middleware.MiddlewareValidateAccessToken)
	// authRouter.Use(middleware.MiddlewareValidateRefreshToken)
	// adminRouter.Use(middleware.MiddlewareValidateRefreshToken)
	adminRouter.Use(middleware.AdminHandler)


	adminRouter.HandleFunc("/api/v1/admin/create-template", projecthandlers.CreateTemplate).Methods("POST","OPTIONS")
	adminRouter.HandleFunc("/api/v1/admin/save-template-pages/{id}", projecthandlers.SavePage).Methods("POST","OPTIONS")
	// do I need to retrun page_id here?
	adminRouter.HandleFunc("/api/v1/admin/add-template-pages/{id}", projecthandlers.AddTemplatePages).Methods("POST","OPTIONS")
	adminRouter.HandleFunc("/api/v1/admin/delete-template-pages/{id}", projecthandlers.DeleteTemplatePages).Methods("POST","OPTIONS")
	adminRouter.HandleFunc("/api/v1/admin/reorder-template-pages/{id}", projecthandlers.ReorderTemplatePages).Methods("POST","OPTIONS")
	adminRouter.HandleFunc("/api/v1/admin/publish-template/{id}", projecthandlers.PublishTemplate).Methods("POST","OPTIONS")

	
	adminRouter.HandleFunc("/api/v1/admin/create-background", projecthandlers.AdminCreateBackground).Methods("POST","OPTIONS")
	adminRouter.HandleFunc("/api/v1/admin/create-decoration", projecthandlers.AdminCreateDecoration).Methods("POST","OPTIONS")
	adminRouter.HandleFunc("/api/v1/admin/create-layout", projecthandlers.AdminCreateLayout).Methods("POST","OPTIONS")
	adminRouter.HandleFunc("/api/v1/admin/delete-background/{id}", projecthandlers.AdminDeleteBackground).Methods("GET","OPTIONS")
	adminRouter.HandleFunc("/api/v1/admin/delete-decoration/{id}", projecthandlers.AdminDeleteDecoration).Methods("GET","OPTIONS")
	adminRouter.HandleFunc("/api/v1/admin/delete-layout/{id}", projecthandlers.AdminDeleteLayout).Methods("POST","OPTIONS")
	adminRouter.HandleFunc("/api/v1/admin/update-background/{id}", projecthandlers.AdminUpdateBackground).Methods("POST","OPTIONS")
	adminRouter.HandleFunc("/api/v1/admin/update-decoration/{id}", projecthandlers.AdminUpdateDecoration).Methods("POST","OPTIONS")

	authRouter.HandleFunc("/api/v1/auth/get-user", userhandlers.CheckUserCategory).Methods("GET","OPTIONS")
	authRouter.HandleFunc("/api/v1/image/save", imagehandlers.LoadImage).Methods("POST","OPTIONS")
	authRouter.HandleFunc("/api/v1/load-photos", projecthandlers.UserLoadPhotos).Methods("GET","OPTIONS")
	authRouter.HandleFunc("/api/v1/upload-photo", projecthandlers.NewPhoto).Methods("POST","OPTIONS")
	authRouter.HandleFunc("/api/v1/delete-photo/{id}", projecthandlers.DeletePhoto).Methods("POST","OPTIONS")
	authRouter.HandleFunc("/api/v1/create-decoration", projecthandlers.CreateDecor).Methods("POST","OPTIONS")
	authRouter.HandleFunc("/api/v1/create-background", projecthandlers.CreateBackground).Methods("POST","OPTIONS")
	authRouter.HandleFunc("/api/v1/delete-decoration/{id}", projecthandlers.DeleteDecor).Methods("GET","OPTIONS")
	authRouter.HandleFunc("/api/v1/delete-background/{id}", projecthandlers.DeleteBackground).Methods("GET","OPTIONS")
	authRouter.HandleFunc("/api/v1/create-project", projecthandlers.CreateBlankProject).Methods("POST","OPTIONS")
	authRouter.HandleFunc("/api/v1/create-pdf-link/{id}", imagehandlers.CreatePDFVisualization).Methods("POST","OPTIONS")
	authRouter.HandleFunc("/api/v1/save-project-pages/{id}", projecthandlers.SavePage).Methods("POST","OPTIONS")
	authRouter.HandleFunc("/api/v1/load-projects", projecthandlers.LoadProjects).Methods("GET","OPTIONS")
	authRouter.HandleFunc("/api/v1/load-project/{id}", projecthandlers.LoadProject).Methods("GET","OPTIONS")
	authRouter.HandleFunc("/api/v1/add-pages/{id}", projecthandlers.AddProjectPages).Methods("POST","OPTIONS")
	authRouter.HandleFunc("/api/v1/delete-pages/{id}", projecthandlers.DeletePages).Methods("POST","OPTIONS")
	authRouter.HandleFunc("/api/v1/reorder-pages/{id}", projecthandlers.ReorderPages).Methods("POST","OPTIONS")
	authRouter.HandleFunc("/api/v1/load-backgrounds", projecthandlers.LoadBackground).Methods("GET","OPTIONS")
	authRouter.HandleFunc("/api/v1/load-decorations", projecthandlers.LoadDecoration).Methods("GET","OPTIONS")
	authRouter.HandleFunc("/api/v1/load-layouts", projecthandlers.LoadLayouts).Methods("GET","OPTIONS")
	authRouter.HandleFunc("/api/v1/change-favourite-background/{id}", projecthandlers.FavourBackground).Methods("POST","OPTIONS")
	authRouter.HandleFunc("/api/v1/change-favourite-decoration/{id}", projecthandlers.FavourDecoration).Methods("POST","OPTIONS")
	authRouter.HandleFunc("/api/v1/change-favourite-layout/{id}", projecthandlers.FavourLayout).Methods("POST","OPTIONS")

	srv := &http.Server{
		Handler: router,
		Addr:    *host,
	}

	go func() {
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server error: %v", err)
		}
		log.Println("Stopped serving new connections.")
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)
	<-sigChan

	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), config.ContextSrvTimeout)
	defer shutdownRelease()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("HTTP shutdown error: %v", err)
	}
	log.Println("Graceful shutdown complete.")

}
