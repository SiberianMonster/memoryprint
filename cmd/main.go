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

	host = config.GetEnv("RUN_ADDRESS", flag.String("a", "127.0.0.1:8080", "SERVER HOST RUN_ADDRESS"))
	connStr = config.GetEnv("DATABASE_URI", flag.String("d", "", "DATA STORAGE DATABASE_URI"))
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
	
	printAgentRouter := router.MatcherFunc(func(r *http.Request, rm *mux.RouteMatch) bool {
		return true
	}).Subrouter()

	
	noAuthRouter.HandleFunc("/api/user/register", userhandlers.Register)
	noAuthRouter.HandleFunc("/api/user/login", userhandlers.Login)
	noAuthRouter.HandleFunc("/api/prices", orderhandlers.LoadPrices)
	noAuthRouter.HandleFunc("/api/templates", projecthandlers.RetrieveTemplates)
	noAuthRouter.HandleFunc("/api/load-template", projecthandlers.LoadTemplate)
	noAuthRouter.HandleFunc("/api/project-session", projecthandlers.LoadProjectSession)
	
	noAuthRouter.HandleFunc("/greet", authhandlers.Greet)
	noAuthRouter.HandleFunc("/get-password-reset-code", authhandlers.GeneratePassResetCode)
	noAuthRouter.HandleFunc("/api/verify/mail", userhandlers.VerifyMail)
	noAuthRouter.HandleFunc("/api/verify/password-reset", userhandlers.VerifyPasswordReset)


	authRouter.Use(middleware.MiddlewareValidateAccessToken)
	adminRouter.Use(middleware.MiddlewareValidateAccessToken)
	// authRouter.Use(middleware.MiddlewareValidateRefreshToken)
	// adminRouter.Use(middleware.MiddlewareValidateRefreshToken)
	adminRouter.Use(middleware.AdminHandler)
	printAgentRouter.Use(middleware.MiddlewareValidateAccessToken)
	// printAgentRouter.Use(middleware.MiddlewareValidateRefreshToken)
	printAgentRouter.Use(middleware.PAHandler)

	adminRouter.HandleFunc("/api/admin/users", userhandlers.ViewUsers).Methods("GET")
	adminRouter.HandleFunc("/api/admin/orders", orderhandlers.ViewOrders).Methods("GET")
	adminRouter.HandleFunc("/api/admin/assign-print-agent", orderhandlers.AddOrderPrintAgency).Methods("POST")
	adminRouter.HandleFunc("/api/admin/create-template", projecthandlers.CreateTemplate).Methods("POST")
	adminRouter.HandleFunc("/api/admin/delete-order", orderhandlers.DeleteOrder).Methods("POST")
	adminRouter.HandleFunc("/api/admin/delete-user", userhandlers.DeleteUser).Methods("POST")
	adminRouter.HandleFunc("/api/admin/update-user-category", userhandlers.UpdateUserCategory).Methods("POST")
	adminRouter.HandleFunc("/api/admin/update-user-status", userhandlers.UpdateUserStatus).Methods("POST")
	adminRouter.HandleFunc("/api/admin/update-order-status", orderhandlers.UpdateOrderStatus).Methods("POST")

	printAgentRouter.HandleFunc("/api/printagent/update-order-status", orderhandlers.UpdateOrderStatus).Methods("POST")
	printAgentRouter.HandleFunc("/api/printagent/orders", orderhandlers.PAViewOrders).Methods("GET")

	authRouter.HandleFunc("/update-username", userhandlers.UpdateUsername)
	authRouter.HandleFunc("/reset-password", userhandlers.ResetPassword)
	authRouter.HandleFunc("/greet", authhandlers.Greet)
	authRouter.HandleFunc("/refresh-token", authhandlers.RefreshToken)
	authRouter.HandleFunc("/get-password-reset-code", authhandlers.GeneratePassResetCode)
	authRouter.HandleFunc("/api/user/create-order", orderhandlers.CreateOrder).Methods("POST")
	authRouter.HandleFunc("/api/user/orders", orderhandlers.UserLoadOrders).Methods("GET")
	authRouter.HandleFunc("/api/user/order-status", orderhandlers.CheckOrderStatus).Methods("GET")
	authRouter.HandleFunc("/api/user/add-new-editor", projecthandlers.AddProjectEditor).Methods("POST")
	authRouter.HandleFunc("/api/user/projects", projecthandlers.UserLoadProjects).Methods("GET")
	authRouter.HandleFunc("/api/user/photos", projecthandlers.UserLoadPhotos).Methods("GET")
	authRouter.HandleFunc("/api/user/upload-photo", projecthandlers.NewPhoto).Methods("POST")
	authRouter.HandleFunc("/api/user/delete-photo", projecthandlers.DeletePhoto).Methods("POST")
	authRouter.HandleFunc("/api/user/create-project", projecthandlers.CreateProject).Methods("POST")
	authRouter.HandleFunc("/api/user/create-designer-project", projecthandlers.CreateDesignerProject).Methods("POST")
	authRouter.HandleFunc("/api/user/save-project", projecthandlers.SaveProject).Methods("POST")
	authRouter.HandleFunc("/api/user/load-project", projecthandlers.LoadProject).Methods("POST")
	authRouter.HandleFunc("/api/user/save-page", projecthandlers.SavePage).Methods("POST")
	authRouter.HandleFunc("/api/user/delete-project", projecthandlers.DeleteProject).Methods("POST")
	authRouter.HandleFunc("/api/user/add-page", projecthandlers.AddProjectPage).Methods("POST")
	authRouter.HandleFunc("/api/user/project-photos", projecthandlers.LoadProjectPhotos).Methods("GET")

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
