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
	"github.com/SiberianMonster/memoryprint/internal/userhandlers"
	"github.com/SiberianMonster/memoryprint/internal/orderhandlers"
	"github.com/SiberianMonster/memoryprint/internal/projecthandlers"
	"github.com/SiberianMonster/memoryprint/internal/middleware"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

var err error
var host, connStr, accrualStr, adminEmail, yandexKey *string
var db *pgxpool.Pool

func init() {

	host = config.GetEnv("RUN_ADDRESS", flag.String("a", "109.70.24.79:8080", "SERVER HOST RUN_ADDRESS"))
	connStr = config.GetEnv("DATABASE_URI", flag.String("d", "postgres://postgres:iQ8hA2vI8p@localhost/memory_print?sslmode=disable&statement_cache_capacity=1", "DATA STORAGE DATABASE_URI"))
	adminEmail = config.GetEnv("ADMIN_EMAIL", flag.String("am", "support@memoryprint.ru", "ADMIN_EMAIL"))
	yandexKey = config.GetEnv("YANDEX_PASSWORD", flag.String("yp", "", "YANDEX_PASSWORD"))

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

	router := mux.NewRouter()
	noAuthRouter := router.MatcherFunc(func(r *http.Request, rm *mux.RouteMatch) bool {
		return r.Header.Get("Authorization") == ""
	}).Subrouter()

	authRouter := router.MatcherFunc(func(r *http.Request, rm *mux.RouteMatch) bool {
		return true
	}).Subrouter()

	adminRouter := router.MatcherFunc(func(r *http.Request, rm *mux.RouteMatch) bool {
		return true
	}).Subrouter()

	refreshRouter := router.MatcherFunc(func(r *http.Request, rm *mux.RouteMatch) bool {
		return true
	}).Subrouter()
	
	printAgentRouter := router.MatcherFunc(func(r *http.Request, rm *mux.RouteMatch) bool {
		return true
	}).Subrouter()

	
	noAuthRouter.HandleFunc("/api/v1/user/register", userhandlers.Register)
	noAuthRouter.HandleFunc("/api/v1/user/login", userhandlers.Login)
	noAuthRouter.HandleFunc("/api/v1/prices", orderhandlers.LoadPrices)
	noAuthRouter.HandleFunc("/api/v1/templates", projecthandlers.RetrieveTemplates)
	noAuthRouter.HandleFunc("/api/v1/project-session", projecthandlers.LoadProjectSession)
	
	noAuthRouter.HandleFunc("/api/v1/greet", authhandlers.Greet)
	noAuthRouter.HandleFunc("/api/v1//get-password-reset-code", authhandlers.GeneratePassResetCode)
	noAuthRouter.HandleFunc("/api/v1/verify/mail", userhandlers.VerifyMail)
	noAuthRouter.HandleFunc("/api/v1/verify/password-reset", userhandlers.VerifyPasswordReset)


	authRouter.Use(middleware.MiddlewareValidateAccessToken)
	adminRouter.Use(middleware.MiddlewareValidateAccessToken)
	// authRouter.Use(middleware.MiddlewareValidateRefreshToken)
	// adminRouter.Use(middleware.MiddlewareValidateRefreshToken)
	adminRouter.Use(middleware.AdminHandler)
	printAgentRouter.Use(middleware.MiddlewareValidateAccessToken)
	// printAgentRouter.Use(middleware.MiddlewareValidateRefreshToken)
	printAgentRouter.Use(middleware.PAHandler)

	refreshRouter.Use(middleware.MiddlewareValidateRefreshToken)
	refreshRouter.HandleFunc("/refresh-token", authhandlers.RefreshToken).Methods("GET")

	adminRouter.HandleFunc("/api/v1/admin/users", userhandlers.ViewUsers).Methods("GET")
	adminRouter.HandleFunc("/api/v1/admin/orders", orderhandlers.ViewOrders).Methods("GET")
	adminRouter.HandleFunc("/api/v1/admin/assign-print-agent", orderhandlers.AddOrderPrintAgency).Methods("POST")
	adminRouter.HandleFunc("/api/v1/admin/create-template", projecthandlers.CreateTemplate).Methods("POST")
	adminRouter.HandleFunc("/api/v1/admin/save-template", projecthandlers.SaveTemplate).Methods("POST")
	adminRouter.HandleFunc("/api/v1/admin/load-template/{id}", projecthandlers.LoadTemplate).Methods("GET")
	adminRouter.HandleFunc("/api/v1/admin/publish-template", projecthandlers.PublishTemplate).Methods("POST")
	adminRouter.HandleFunc("/api/v1/admin/delete-order/{id}", orderhandlers.DeleteOrder).Methods("GET")
	adminRouter.HandleFunc("/api/v1/admin/delete-user", userhandlers.DeleteUser).Methods("POST")
	adminRouter.HandleFunc("/api/v1/admin/update-user-category", userhandlers.UpdateUserCategory).Methods("POST")
	adminRouter.HandleFunc("/api/v1/admin/update-user-status", userhandlers.UpdateUserStatus).Methods("POST")
	adminRouter.HandleFunc("/api/v1/admin/update-order-status", orderhandlers.UpdateOrderStatus).Methods("POST")

	printAgentRouter.HandleFunc("/api/v1/printagent/update-order-status", orderhandlers.UpdateOrderStatus).Methods("POST")
	printAgentRouter.HandleFunc("/api/v1/printagent/orders", orderhandlers.PAViewOrders).Methods("GET")

	authRouter.HandleFunc("/api/v1/update-username", userhandlers.UpdateUsername)
	authRouter.HandleFunc("/api/v1/reset-password", userhandlers.ResetPassword)
	authRouter.HandleFunc("/api/v1/greet", authhandlers.Greet)
	authRouter.HandleFunc("/api/v1/refresh-token", authhandlers.RefreshToken)
	authRouter.HandleFunc("/api/v1/get-password-reset-code", authhandlers.GeneratePassResetCode)
	authRouter.HandleFunc("/api/v1/user/create-draft-order", orderhandlers.CreateDraftOrder).Methods("POST")
	authRouter.HandleFunc("/api/v1/user/orders", orderhandlers.UserLoadOrders).Methods("GET")
	authRouter.HandleFunc("/api/v1/user/order-status", orderhandlers.CheckOrderStatus).Methods("GET")
	authRouter.HandleFunc("/api/v1/user/add-new-editor", projecthandlers.AddProjectEditor).Methods("POST")
	authRouter.HandleFunc("/api/v1/user/add-new-viewer", projecthandlers.AddProjectViewer).Methods("POST")
	authRouter.HandleFunc("/api/v1/user/projects", projecthandlers.UserLoadProjects).Methods("GET")
	authRouter.HandleFunc("/api/v1/user/photos", projecthandlers.UserLoadPhotos).Methods("GET")
	authRouter.HandleFunc("/api/v1/user/upload-photo", projecthandlers.NewPhoto).Methods("POST")
	authRouter.HandleFunc("/api/v1/user/delete-photo/{id}", projecthandlers.DeletePhoto).Methods("GET")
	authRouter.HandleFunc("/api/v1/user/create-decor", projecthandlers.CreateDecor).Methods("POST")
	authRouter.HandleFunc("/api/v1/user/create-background", projecthandlers.CreateBackground).Methods("POST")
	authRouter.HandleFunc("/api/v1/user/delete-decor/{id}", projecthandlers.DeleteDecor).Methods("GET")
	authRouter.HandleFunc("/api/v1/user/delete-background/{id}", projecthandlers.DeleteBackground).Methods("GET")
	authRouter.HandleFunc("/api/v1/user/load-personalised-objects", projecthandlers.UserLoadPersObjects).Methods("GET")
	authRouter.HandleFunc("/api/v1/user/create-project", projecthandlers.CreateBlankProject).Methods("POST")
	authRouter.HandleFunc("/api/v1/user/create-project-from-template", projecthandlers.CreateProjectFromTemplate).Methods("POST")
	authRouter.HandleFunc("/api/v1/user/create-designer-project", projecthandlers.CreateDesignerProject).Methods("POST")
	authRouter.HandleFunc("/api/v1/user/save-project", projecthandlers.SaveProject).Methods("POST")
	authRouter.HandleFunc("/api/v1/user/save-page", projecthandlers.SavePage).Methods("POST")
	authRouter.HandleFunc("/api/v1/user/load-project/{id}", projecthandlers.LoadProject).Methods("GET")
	authRouter.HandleFunc("/api/v1/user/delete-project/{id}", projecthandlers.DeleteProject).Methods("GET")
	authRouter.HandleFunc("/api/v1/user/add-page", projecthandlers.AddProjectPage).Methods("POST")
	authRouter.HandleFunc("/api/v1/user/duplicate-page", projecthandlers.DuplicatePage).Methods("POST")
	authRouter.HandleFunc("/api/v1/user/delete-page/{id}", projecthandlers.DeletePage).Methods("GET")
	authRouter.HandleFunc("/api/v1/user/project-photos", projecthandlers.LoadProjectPhotos).Methods("GET")

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
