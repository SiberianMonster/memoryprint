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
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

var err error
var host, connStr, accrualStr *string
var db *pgxpool.Pool

func init() {

	host = config.GetEnv("RUN_ADDRESS", flag.String("a", "127.0.0.1:8080", "SERVER HOST RUN_ADDRESS"))
	connStr = config.GetEnv("DATABASE_URI", flag.String("d", "", "DATA STORAGE DATABASE_URI"))

}


func main() {

	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), config.ContextDBTimeout*time.Second)
	defer cancel()

	if *connStr == "" {
		log.Fatalf("Database credentials were not passed")
	}

	config.DB, _ = initstorage.SetUpDBConnection(ctx, connStr)
	defer config.DB.Close()

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

	mailR := router.PathPrefix("/verify").Methods(http.MethodPost).Subrouter()
	mailR.HandleFunc("/mail", userhandlers.VerifyMail)
	mailR.HandleFunc("/password-reset", userhandlers.VerifyPasswordReset)
	mailR.Use(middleware.MiddlewareValidateVerificationData)

	// used the PathPrefix as workaround for scenarios where all the
	// get requests must use the ValidateAccessToken middleware except
	// the /refresh-token request which has to use ValidateRefreshToken middleware
	refToken := router.PathPrefix("/refresh-token").Subrouter()
	refToken.HandleFunc("", authhandlers.RefreshToken)
	refToken.Use(middleware.MiddlewareValidateRefreshToken)

	getR := router.Methods(http.MethodGet).Subrouter()
	getR.HandleFunc("/greet", authhandlers.Greet)
	getR.HandleFunc("/get-password-reset-code", authhandlers.GeneratePassResetCode)
	getR.Use(middleware.MiddlewareValidateAccessToken)

	authRouter.Use(middleware.MiddlewareValidateAccessToken)
	adminRouter.Use(middleware.MiddlewareValidateAccessToken)
	adminRouter.Use(middleware.AdminHandler)
	printAgentRouter.Use(middleware.MiddlewareValidateAccessToken)
	printAgentRouter.Use(middleware.PAHandler)

	adminRouter.HandleFunc("/api/admin/users", userhandlers.ViewUsers).Methods("GET")
	adminRouter.HandleFunc("/api/admin/orders", orderhandlers.ViewOrders).Methods("GET")
	adminRouter.HandleFunc("/api/admin/assign-print-agent", orderhandlers.AddOrderPrintAgency).Methods("POST")
	adminRouter.HandleFunc("/api/admin/delete-order", orderhandlers.DeleteOrder).Methods("POST")
	adminRouter.HandleFunc("/api/admin/delete-user", userhandlers.DeleteUser).Methods("POST")
	adminRouter.HandleFunc("/api/admin/update-user-category", userhandlers.UpdateUserCategory).Methods("POST")
	adminRouter.HandleFunc("/api/admin/update-user-status", userhandlers.UpdateUserStatus).Methods("POST")
	adminRouter.HandleFunc("/api/admin/update-order-status", orderhandlers.UpdateOrderStatus).Methods("POST")

	printAgentRouter.HandleFunc("/api/printagent/update-order-status", orderhandlers.UpdateOrderStatus).Methods("POST")
	printAgentRouter.HandleFunc("/api/printagent/orders", orderhandlers.PAViewOrders).Methods("GET")

	authRouter.HandleFunc("/update-username", userhandlers.UpdateUsername)
	authRouter.HandleFunc("/reset-password", userhandlers.ResetPassword)
	authRouter.HandleFunc("/api/user/create-order", orderhandlers.CreateOrder).Methods("POST")
	authRouter.HandleFunc("/api/user/orders", orderhandlers.UserLoadOrders).Methods("GET")
	authRouter.HandleFunc("/api/user/add-new-editor", projecthandlers.AddProjectEditor).Methods("POST")
	authRouter.HandleFunc("/api/user/projects", projecthandlers.UserLoadProjects).Methods("GET")
	authRouter.HandleFunc("/api/user/photos", projecthandlers.UserLoadPhotos).Methods("GET")
	authRouter.HandleFunc("/api/user/upload-photo", projecthandlers.NewPhoto).Methods("POST")
	authRouter.HandleFunc("/api/user/delete-photo", projecthandlers.DeletePhoto).Methods("POST")
	authRouter.HandleFunc("/api/user/create-project", projecthandlers.CreateProject).Methods("POST")
	authRouter.HandleFunc("/api/user/load-project", projecthandlers.LoadProject).Methods("POST")
	authRouter.HandleFunc("/api/user/save-page", projecthandlers.SavePage).Methods("POST")
	authRouter.HandleFunc("/api/user/delete-project", projecthandlers.DeleteProject).Methods("POST")
	authRouter.HandleFunc("/api/user/add-page", projecthandlers.AddProjectPage).Methods("POST")
	authRouter.HandleFunc("/api/user/project-photos", projecthandlers.LoadProjectPhotos).Methods("GET")
	authRouter.HandleFunc("/api/user/project-session", projecthandlers.LoadProjectSession).Methods("GET")

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
