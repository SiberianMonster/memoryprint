// Handlers package contains endpoints handlers for the Photo Book Editor module.
//
// https://github.com/SiberianMonster/memoryprint/tree/development/internal/projecthandlers
package orderhandlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"
	"github.com/SiberianMonster/memoryprint/internal/config"
	"github.com/SiberianMonster/memoryprint/internal/delivery"
	"github.com/SiberianMonster/memoryprint/internal/emailutils"
	"github.com/SiberianMonster/memoryprint/internal/models"
	"github.com/SiberianMonster/memoryprint/internal/orderstorage"
	"github.com/SiberianMonster/memoryprint/internal/handlersfunc"
	"github.com/SiberianMonster/memoryprint/internal/transactions"
	"github.com/go-playground/validator/v10"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/jackc/pgx/v5/pgxpool"
)

var err error
var resp map[string]string

func AddWorkdays(date time.Time, days int) time.Time {
	for {
		 if (days == 0) {
			 return date
		 }
 
		 date = date.AddDate(0, 0, 1)
		
		 if date.Weekday() == time.Saturday {
			 date = date.AddDate(0, 0, 2)
			 return AddWorkdays(date, days-1)
		 } else if date.Weekday() == time.Sunday {
			 date = date.AddDate(0, 0, 1)
			 return AddWorkdays(date, days-1)
		 }
 
		 days--
	}
 }


func LoadCart(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]models.ResponseCart)
	var CartObjs models.ResponseCart 
	
	defer r.Body.Close()
	// Create a new validator instance
    
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	userID := handlersfunc.UserIDContextReader(r)
	log.Printf("Loading cart for user %d", userID)
	CartObjs, err = orderstorage.LoadCart(ctx, config.DB, userID)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}

	rw.WriteHeader(http.StatusOK)
	resp["response"] = CartObjs
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func CreateOrder(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]uint)
	var OrderObj models.NewOrder
	var oID uint
	
	err := json.NewDecoder(r.Body).Decode(&OrderObj)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, err)
		return
	}

	defer r.Body.Close()
	// Create a new validator instance
    validate := validator.New()

    // Validate the User struct
    err = validate.Struct(OrderObj)
    if err != nil {
        // Validation failed, handle the error
		handlersfunc.HandleValidationError(rw, err)
        return
    }
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	userID := handlersfunc.UserIDContextReader(r)
	log.Printf("Create order for user %d", userID)
	oID, err = orderstorage.CreateOrder(ctx, config.DB, userID, OrderObj)
	// set project status to published, add links
	// create order awaiting payment
	//calculate base price

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}

	rw.WriteHeader(http.StatusOK)
	resp["response"] = oID
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}




func GeneratePersonalPromooffer(rw http.ResponseWriter, r *http.Request) {


	// after registration
	
}



func OrderPayment(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]models.TransactionLink)
	var OrderObj models.RequestOrderPayment
	var transaction models.TransactionLink
	var oID uint
	var link string
	var priceforlink float64
	
	err := json.NewDecoder(r.Body).Decode(&OrderObj)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, err)
		return
	}

	defer r.Body.Close()
	// Create a new validator instance
    validate := validator.New()

    // Validate the User struct
    err = validate.Struct(OrderObj)
    if err != nil {
        // Validation failed, handle the error
		handlersfunc.HandleValidationError(rw, err)
        return
    }
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	userID := handlersfunc.UserIDContextReader(r)
	log.Printf("Payment for order for user %d", userID)
	log.Println(OrderObj)
	priceforlink, oID, err = orderstorage.OrderPayment(ctx, config.DB, OrderObj, userID)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}

	link, err = transactions.CreateTransaction(oID, priceforlink, "PHOTOBOOK") 
	if err != nil {
		handlersfunc.HandleFailedPaymentURL(rw)
		return
	}
	transaction.PaymentLink = link

	rw.WriteHeader(http.StatusOK)
	resp["response"] = transaction
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
	
}

func CancelPayment(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]uint)
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	orderID := uint(aByteToInt)

	defer r.Body.Close()
	userID := handlersfunc.UserIDContextReader(r)
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()

	err := transactions.CancelTransaction(orderID)
	if err != nil {
		log.Printf("Failed to cancel transaction for order %d", orderID)
		handlersfunc.HandleFailedCancellationError(rw)
		return
	}
	
	err = orderstorage.CancelPayment(ctx, config.DB, orderID, userID)
	
	if err != nil {
		log.Printf("Failed to update cancelled transaction in db for order %d", orderID)
		handlersfunc.HandleFailedCancellationError(rw)
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
	
	// email to admin

	
}

//after order status changes to IN PRINT create delivery order with api call
// data = {"number":"123abc","tariff_code":184,"recipient":{"name":"Иванов Иван Иванович","phones":[{"number":"+79162153969"}]},"from_location":{"address":"Снежная 2/1"},"to_location":{"address":"Лазоревый проезд, 1"},"packages":[{"number":"123abc","weight":200.2,"length":20,"width":20,"height":20,"items":[{"name":"book","ware_key":"1a","payment":{"value":0.0},"cost":10000,"weight":200.2,"amount":1}]}]}

// goroutinre requesting delivery status
// send api to bank: username, token, transactionID, 
// wait until status 2, set order to "PAID"
// set transaction values

func LoadOrders(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]models.ResponseOrders)
	var respOrders models.ResponseOrders
	defer r.Body.Close()
		
	isactive, _ := strconv.ParseBool(r.URL.Query().Get("isactive"))
	rOffset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	rLimit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset := uint(rOffset)
	limit := uint(rLimit)
	
		
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	userID := handlersfunc.UserIDContextReader(r)
	log.Printf("Load orders of the user %d", userID)
	respOrders, err := orderstorage.RetrieveOrders(ctx, config.DB, userID, isactive, offset, limit)
	
	if err != nil {
			handlersfunc.HandleDatabaseServerError(rw)
			return
	}
	
	rw.WriteHeader(http.StatusOK)
	resp["response"] = respOrders
	jsonResp, err := json.Marshal(resp)
	if err != nil {
			log.Printf("Error happened in JSON marshal. Err: %s", err)
			return
	}
	rw.Write(jsonResp)
	
}
	

	// output
	// models.Order awaiting_payment true, created_at, status, baseprice, finalprice, tracking number, last_updated_at
	// Project pages, cover, surface, variant, name, image, previewlink
	


func LoadAdminOrders(rw http.ResponseWriter, r *http.Request) {


	// input active / notactive email orderID, userID, created_at after, created_at before, status
	resp := make(map[string]models.ResponseAdminOrders)
	var respOrders models.ResponseAdminOrders
	defer r.Body.Close()
		
	isactive, _ := strconv.ParseBool(r.URL.Query().Get("isactive"))
	orderID, _ := strconv.Atoi(r.URL.Query().Get("orderid"))
	userID, _ := strconv.Atoi(r.URL.Query().Get("userid"))
	orderID = orderID
	userID = userID
	createdAfter, _ := strconv.Atoi(r.URL.Query().Get("createdafter"))
	createdBefore, _ := strconv.Atoi(r.URL.Query().Get("createdbefore"))
	createdAfter = createdAfter
	createdBefore = createdBefore
	email := r.URL.Query().Get("email")
	status := r.URL.Query().Get("status")
	rOffset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	rLimit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset := uint(rOffset)
	limit := uint(rLimit)
	
		
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	log.Printf("Load orders of the admin")
	respOrders, err := orderstorage.RetrieveAdminOrders(ctx, config.DB, uint(userID), uint(orderID), isactive, uint(createdAfter), uint(createdBefore), email, status, offset, limit)
	
	if err != nil {
			handlersfunc.HandleDatabaseServerError(rw)
			return
	}
	
	rw.WriteHeader(http.StatusOK)
	resp["response"] = respOrders
	jsonResp, err := json.Marshal(resp)
	if err != nil {
			log.Printf("Error happened in JSON marshal. Err: %s", err)
			return
	}
	rw.Write(jsonResp)
	


	// output
	// models.Order awaiting_payment true, created_at, status, baseprice, finalprice, tracking number, last_updated_at, userID, user email, commentary
	// Project pages, cover, surface, variant, name, image, previewlink
	
}

func LoadOrder(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]models.ResponseOrderInfo)
	var retrievedOrder models.ResponseOrderInfo
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	orderID := uint(aByteToInt)
	defer r.Body.Close()
	
	retrievedOrder, err = orderstorage.LoadOrder(ctx, config.DB, orderID)
	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}
	log.Println(retrievedOrder)
	
	rw.WriteHeader(http.StatusOK)
	resp["response"] = retrievedOrder
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func AdminLoadOrder(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]models.ResponseOrderInfo)
	var retrievedOrder models.ResponseOrderInfo
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	orderID := uint(aByteToInt)
	defer r.Body.Close()
	
	retrievedOrder, err = orderstorage.AdminLoadOrder(ctx, config.DB, orderID)
	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}
	log.Println(retrievedOrder)
	
	rw.WriteHeader(http.StatusOK)
	resp["response"] = retrievedOrder
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func LoadDelivery(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]models.ResponseDeliveryInfo)
	var retrievedOrder models.ResponseDeliveryInfo
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	orderID := uint(aByteToInt)
	defer r.Body.Close()
	
	retrievedOrder, err = orderstorage.LoadDelivery(ctx, config.DB, orderID)
	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}
	log.Println(retrievedOrder)
	
	rw.WriteHeader(http.StatusOK)
	resp["response"] = retrievedOrder
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}

func UpdateOrderStatus(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]uint)
	var StatusObj models.RequestUpdateOrderStatus
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	orderID := uint(aByteToInt)
	
	err := json.NewDecoder(r.Body).Decode(&StatusObj)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, err)
		return
	}

	defer r.Body.Close()
   
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	
	err = orderstorage.UpdateOrderStatus(ctx, config.DB, orderID, StatusObj)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}

	if StatusObj.Status == "IN_DELIVERY" {
		//orderObj, err := orderstorage.RetrieveSingleOrder(ctx , storeDB, order.OrdersID) 
		var deliveryObj models.ResponseDeliveryInfo
		var contactData models.Contacts
		var retrievedOrder models.ResponseOrderInfo
		deliveryObj, err := orderstorage.LoadDelivery(ctx , config.DB, orderID)
		if err != nil {
			handlersfunc.HandleDatabaseServerError(rw)
			return
		}
		retrievedOrder, err = orderstorage.LoadOrder(ctx, config.DB, orderID)
		if err != nil {
			handlersfunc.HandleDatabaseServerError(rw)
			return
		}
		log.Println(retrievedOrder)
		contactData = retrievedOrder.ContactData

		// Send paid order email
		from := "support@memoryprint.ru"
		to := []string{contactData.Email}
		subject := "Ваш заказ передан в службу доставки!"
		mailType := emailutils.MailOrderInDelivery
		mailData := &emailutils.MailData{
			Username: contactData.FirstName,
			Ordernum: orderID,
			Trackingnum: *deliveryObj.TrackingNumber,
			//Order: orderObj,
		}

		ms := &emailutils.SGMailService{config.YandexApiKey}
		mailReq := emailutils.NewMail(from, to, subject, mailType, mailData)
		err = emailutils.SendMail(mailReq, ms)
		if err != nil {
			handlersfunc.HandleMailSendError(rw)
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

func UpdateOrderCommentary(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]uint)
	var CommentaryObj models.RequestUpdateOrderCommentary
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	orderID := uint(aByteToInt)
	
	err := json.NewDecoder(r.Body).Decode(&CommentaryObj)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, err)
		return
	}

	defer r.Body.Close()
   
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	
	err = orderstorage.UpdateOrderCommentary(ctx, config.DB, orderID, CommentaryObj)

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


func UploadOrderVideo(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]uint)
	var VideoObj models.OrderVideo
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	orderID := uint(aByteToInt)
	
	err := json.NewDecoder(r.Body).Decode(&VideoObj)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, err)
		return
	}

	defer r.Body.Close()
   
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	
	err = orderstorage.UploadOrderVideo(ctx, config.DB, orderID, VideoObj)

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

func DownloadOrderVideo(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]models.OrderVideo)
	aByteToInt, _ := strconv.Atoi(mux.Vars(r)["id"])
	orderID := uint(aByteToInt)

	defer r.Body.Close()
   
	ctx, cancel := context.WithTimeout(r.Context(), config.ContextDBTimeout)
	defer cancel()
	
	VideoObj, err := orderstorage.DownloadOrderVideo(ctx, config.DB, orderID)

	if err != nil {
		handlersfunc.HandleDatabaseServerError(rw)
		return
	}

	rw.WriteHeader(http.StatusOK)
	resp["response"] = VideoObj
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}


func CalculateDelivery(rw http.ResponseWriter, r *http.Request) {

	resp := make(map[string]models.ResponseDeliveryCost)
	var PaymentObj models.ResponseDeliveryCost
	var ApiPaymentObj models.ApiResponseDeliveryCost
	var rCost models.UserRequestDeliveryCost
	var rApiCost models.RequestDeliveryCost
	var fromLoc models.Location
	var toLoc models.Location
	var p models.Package
	var s models.Service
	
	err := json.NewDecoder(r.Body).Decode(&rCost)
	if err != nil {
		handlersfunc.HandleDecodeError(rw, err)
		return
	}

	defer r.Body.Close()
	if rCost.Method == "door_to_door" {
		rApiCost.TariffCode = 139
	} else if rCost.Method == "door_to_office" {
		rApiCost.TariffCode = 138
	}
	toLoc.Address = rCost.Address
	toLoc.PostalCode = rCost.PostalCode
	toLoc.City = rCost.City
	toLoc.Code = rCost.Code
	rApiCost.ToLocation = toLoc

	p.Weight = (300 * rCost.CountProjects)
	p.Height = 3 * rCost.CountProjects
	p.Width = 30 
	p.Length = 30 
	s.Code = "CARTON_BOX_500GR"
	s.Parameter = strconv.Itoa(rCost.CountProjects)
	rApiCost.Packages = append(rApiCost.Packages, p)
	rApiCost.Services = append(rApiCost.Services, s)
	fromLoc.PostalCode = "141016"
	fromLoc.City = "Мытищи"
	fromLoc.Address = "Тенистый бульвар, 11"
	rApiCost.FromLocation = fromLoc

	ApiPaymentObj, err = delivery.CalculateDelivery(rApiCost)
	
	if err != nil {
		handlersfunc.HandleDeliveryCalculationError(rw)
		return
	}
	PaymentObj.Amount = ApiPaymentObj.TotalSum
	log.Println(ApiPaymentObj.PeriodMin)
	log.Println(ApiPaymentObj.PeriodMax)
	daysFrom := 4 + ApiPaymentObj.PeriodMin
	dateFrom := AddWorkdays(time.Now(), daysFrom)
	PaymentObj.ExpectedDeliveryFrom = dateFrom.Format("02-01-2006")
	daysTo := 4 + ApiPaymentObj.PeriodMax
	dateTo := AddWorkdays(time.Now(), daysTo)
	PaymentObj.ExpectedDeliveryTo = dateTo.Format("02-01-2006")
	rw.WriteHeader(http.StatusOK)
	resp["response"] = PaymentObj
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	rw.Write(jsonResp)
}


func SentOrdersToPrint(ctx context.Context, storeDB *pgxpool.Pool) {

	ticker := time.NewTicker(config.UpdateInterval)
	var err error
	var orderList []models.PaidOrderObj

	jobCh := make(chan models.PaidOrderObj)
	for i := 0; i < config.WorkersCount; i++ {
		go func() {
			for job := range jobCh {
	
				err = orderstorage.OrdersToPrint(ctx, storeDB, job)
				if err != nil {
					log.Printf("Error happened when updating pending orders. Err: %s", err)
					continue
				}
			}
		}()
	}

	for range ticker.C {

		orderList, err = orderstorage.LoadPaidOrders(ctx, storeDB)
		if err != nil {
			log.Printf("Error happened when retrieving pending orders. Err: %s", err)
			continue
		}

		for _, order := range orderList {
			jobCh <- order

		}

	}
}