// Handlers package contains endpoints handlers for the Photo Book Editor module.
//
// https://github.com/SiberianMonster/memoryprint/tree/development/internal/orderhandlers
package orderhandlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"io/ioutil"
	"strconv"

	"github.com/SiberianMonster/memoryprint/internal/config"
	"github.com/SiberianMonster/memoryprint/internal/models"
	"github.com/SiberianMonster/memoryprint/internal/orderstorage"
	"github.com/SiberianMonster/memoryprint/internal/handlersfunc"
	_ "github.com/lib/pq"
)

var err error
var resp map[string]string

func ViewOrders(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]string)
	defer r.Body.Close()
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	orders, err := orderstorage.RetrieveOrders(ctx, config.DB)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
		return
	}

	if len(orders) == 0 {
		handlersfunc.HandleNoContent(rw, resp)
		return
	}

	rw.WriteHeader(http.StatusOK)
	json.NewEncoder(rw).Encode(orders)
}

func AddOrderPrintAgency(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]string)
	var PaParams models.PAAssignment

	err := json.NewDecoder(r.Body).Decode(&PaParams)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, resp, err)
		return
	}

	defer r.Body.Close()

	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	_, err = orderstorage.AddOrderResponsible(ctx, config.DB, PaParams.PaID, PaParams.OrderID)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
		return
	}

	
	rw.WriteHeader(http.StatusOK)
	resp["status"] = "printing agency assigned successfully"
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)

}

func DeleteOrder(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]string)
	orderNumBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		handlersfunc.HandleWrongBytesInput(rw, resp)
		return
	}
	defer r.Body.Close()
	aByteToInt, _ := strconv.Atoi(string(orderNumBytes))
	orderNum := uint(aByteToInt)

	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()


	_, err =  orderstorage.DeleteOrder(ctx, config.DB, orderNum)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
		return
	}

	rw.WriteHeader(http.StatusOK)
	resp["status"] = "order deleted successfully"
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)

}


func CreateOrder(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]string)
	var OrderParams models.AdminOrder

	err := json.NewDecoder(r.Body).Decode(&OrderParams)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, resp, err)
		return
	}
	
	defer r.Body.Close()

	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	userID := handlersfunc.UserIDContextReader(r)
	_, err = orderstorage.AddOrder(ctx, config.DB, OrderParams.Link, userID)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
		return
	}

	
	rw.WriteHeader(http.StatusOK)
	resp["status"] = "order created successfully"
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)

}

func UpdateOrderStatus(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]string)
	var OrderParams models.AdminOrder

	err := json.NewDecoder(r.Body).Decode(&OrderParams)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, resp, err)
		return
	}
	
	defer r.Body.Close()

	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	err = orderstorage.UpdateOrderPaymentStatus(ctx, config.DB, OrderParams.Status, OrderParams.OrderID)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
		return
	}

	
	rw.WriteHeader(http.StatusOK)
	resp["status"] = "order status updated successfully"
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)

}

func PAViewOrders(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]string)
	defer r.Body.Close()
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	userID := handlersfunc.UserIDContextReader(r)
	orders, err := orderstorage.PARetrieveOrders(ctx, config.DB, userID)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
		return
	}

	if len(orders) == 0 {
		handlersfunc.HandleNoContent(rw, resp)
		return
	}

	rw.WriteHeader(http.StatusOK)
	json.NewEncoder(rw).Encode(orders)
}


func UserLoadOrders(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]string)
	defer r.Body.Close()
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	userID := handlersfunc.UserIDContextReader(r)
	orders, err := orderstorage.RetrieveUserOrders(ctx, config.DB, userID)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw, resp)
		return
	}

	if len(orders) == 0 {
		handlersfunc.HandleNoContent(rw, resp)
		return
	}

	rw.WriteHeader(http.StatusOK)
	json.NewEncoder(rw).Encode(orders)
}
