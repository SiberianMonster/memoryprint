package delivery

import (
	"context"
	"encoding/json"
	"github.com/SiberianMonster/memoryprint/internal/config"
	"github.com/SiberianMonster/memoryprint/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"log"
	"net/http"
	"net/url"
	"time"
	"strconv"
	"errors"
	"bytes"
)

// LoadActiveDeliveries function performs the operation of retrieving orders in IN_DELIVERY status from pgx database with a query.
func LoadActiveDeliveries(ctx context.Context, storeDB *pgxpool.Pool) ([]uint, error) {

	var orders []uint

	rows, err := storeDB.Query(ctx, "SELECT orders_id FROM orders WHERE status = ($1);", "IN_DELIVERY")
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				log.Printf("Error happened when retrieving orders in delivery info from pgx table. Err: %s", err)
				return orders, err
	}

	
	for rows.Next() {
		var order uint
		if err = rows.Scan(&order); err != nil {
			log.Printf("Error happened when scanning projects. Err: %s", err)
			return orders, err
		}

		orders = append(orders, order)
	}
	return orders, nil

}

type FullPackage struct {
	
	Number int `json:"number" validate:"required"`
	Weight float64 `json:"weight" validate:"required"`
	Length float64 `json:"length" validate:"required"`
	Width float64 `json:"width" validate:"required"`
	Height float64 `json:"height" validate:"required"`
	Cost float64 `json:"cost" validate:"required"`
	Amount float64 `json:"amount" validate:"required"`
	Items []Item `json:"items" validate:"required"`
}

type Item struct {
	
	Name string `json:"name" validate:"required"`
	WareKey string `json:"ware_key" validate:"required"`
	Payment uint `json:"payment" validate:"required"`

}

type DeliveryRecipientCost struct {
	
	Value int `json:"value" validate:"required"`

}

type DeliveryRecipient struct {
	
	Name string `json:"name" validate:"required"`
	Email string `json:"email" validate:"required"`
	Phones []Phone `json:"phones" validate:"required"`

}

type Phone struct {
	
	Number string `json:"number" validate:"required"`

}



type ResponseAuthorization struct {

	AccessToken string `json:"access_token" validate:"required"`
	TokenType string `json:"token_type" validate:"required"`
	ExpiresIn uint `json:"expires_in" validate:"required"`
	Scope string `json:"scope" validate:"required"`
	JTI string `json:"jti" validate:"required"`
}

type RequestAuthorization struct {

	GrantType string `json:"grant_type" validate:"required"`
	ClientID string `json:"client_id" validate:"required"`
	ClientSecret string `json:"client_secret" validate:"required"`

}

type RequestDelivery struct {

	TariffCode int64 `json:"tariff_code" validate:"required"`
	DeliveryRecipientCost DeliveryRecipientCost `json:"delivery_recipient_cost" validate:"required"`
	Recipient DeliveryRecipient `json:"recipient" validate:"required"`
	FromLocation models.Location `json:"from_location" validate:"required"`
	ToLocation models.Location `json:"to_location",omitempty`
	DeliveryPoint string `json:"delivery_point",omitempty`
	Packages []FullPackage `json:"packages" validate:"required"`

}

type ResponseDelivery struct {

	Entity Entity `json:"entity" validate:"required"`
	Requests []DeliveryRequest `json:"requests" validate:"required"`

}

type Entity struct {

	UUID string `json:"uuid" validate:"required"`

}

type DeliveryRequest struct {

	RequestUUID string `json:"request_uuid" validate:"required"`
	Type string `json:"type" validate:"required"`
	State string `json:"state" validate:"required"`
	DateTime string `json:"date_time" validate:"required"`
	Errors []string `json:"errors" validate:"required"`
	Warnings []string `json:"warnings" validate:"required"`

}

type DeliveryStatus struct {

	Code string `json:"code" validate:"required"`
	Name string `json:"name" validate:"required"`
	City string `json:"city" validate:"required"`
	DateTime string `json:"date_time" validate:"required"`

} 

type DeliveryStatusCheck struct {

	Entity DeliveryEntity `json:"entity" validate:"required"`
	Requests interface{} `json:"requests" validate:"required"`

}

type DeliveryEntity struct {

	UUID string `json:"uuid" validate:"required"`
	Type int `json:"type" validate:"required"`
	IsReturn bool `json:"is_return" validate:"required"`
	IsReverse bool `json:"is_reverse" validate:"required"`
	CDEKNumber string `json:"cdek_number" validate:"required"`
	Number string `json:"number" validate:"required"`
	DeliveryMode string `json:"delivery_mode" validate:"required"`
	TariffCode int `json:"tariff_code" validate:"required"`
	Comment string `json:"comment" validate:"required"`
	DeliveryRecipientCost interface{} `json:"delivery_recipient_cost" validate:"required"`
	DeliveryRecipientCostAdv interface{} `json:"delivery_recipient_cost_adv" validate:"required"`
	Sender interface{} `json:"sender" validate:"required"`
	Seller interface{} `json:"seller" validate:"required"`
	Recipient interface{} `json:"recipient" validate:"required"`
	FromLocation interface{} `json:"from_location" validate:"required"`
	ToLocation interface{} `json:"to_location" validate:"required"`
	Services interface{} `json:"services" validate:"required"`
	Packages interface{} `json:"packages" validate:"required"`
	DeliveryProblem []string `json:"delivery_problem" validate:"required"`
	Statuses []DeliveryStatus `json:"statuses" validate:"required"`
	DeliveryDetail interface{} `json:"delivery_detail" validate:"required"`

}

func AuthorizeDelivery() (string, error) {
	var authResponse ResponseAuthorization
	ctx, cancel := context.WithTimeout(context.Background(), config.ContextDBTimeout)
	// не забываем освободить ресурс
	defer cancel()
	
	authorizationURL := &url.URL{
        Scheme: "https",
        Host:   config.DeliveryDomain,
        Path:   "/v2/oauth/token",
    }
	queryValues := url.Values{}
    queryValues.Add("grant_type", "client_credentials")
    queryValues.Add("client_id", config.DeliveryClientID)
	queryValues.Add("client_secret", config.DeliverySecret)
    authorizationURL.RawQuery = queryValues.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, authorizationURL.String(), nil)
	if err != nil {
		log.Printf("Error in authorization")
		return "", errors.New("failed request to authorize")
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := http.DefaultClient
	response, err := client.Do(request)

	if err != nil {
		log.Printf("Error in authorization")
		return "", errors.New("failed request to authorize")
	}

	if response.StatusCode == http.StatusInternalServerError {
		log.Printf("Error in authorization 500")
		return "", errors.New("failed request to authorize")
	}

	if response.StatusCode == http.StatusOK {
		err := json.NewDecoder(response.Body).Decode(&authResponse)
		defer response.Body.Close()
		if err != nil  {
			log.Printf("Unable to decode dauthorization")
			return "", errors.New("failed decoding response to authorize")
		}
		
		return authResponse.AccessToken, nil

	}
	return "", errors.New("failed request to authorize")
}

func CalculateDelivery(OrderObj models.RequestDeliveryCost) (models.ApiResponseDeliveryCost, error) {
	var delivery models.ApiResponseDeliveryCost
	ctx, cancel := context.WithTimeout(context.Background(), config.ContextDBTimeout)
	// не забываем освободить ресурс
	defer cancel()

	accessToken, _ := AuthorizeDelivery()

	
	calculateURL := "https://api.edu.cdek.ru/v2/calculator/tariff"
	bodyBytes, _ := json.Marshal(OrderObj)
	body := bytes.NewBuffer(bodyBytes)
	log.Println(body)

    
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, calculateURL, body)
	if err != nil {
		log.Printf("Error in calculating delivery cost")
		return delivery, errors.New("failed request to delivery")
	}
	fullToken := "Bearer " + accessToken
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", fullToken)

	client := http.DefaultClient
	response, err := client.Do(request)

	if err != nil {
		log.Printf("Error in calculating delivery cost")
		return delivery, errors.New("failed request to delivery")
	}
	log.Println(response.StatusCode)
	if response.StatusCode == http.StatusInternalServerError {
		log.Printf("Internal server error happened when calculating delivery cost")
		return delivery, errors.New("failed response from delivery")
	}

	if response.StatusCode == http.StatusOK {
		err := json.NewDecoder(response.Body).Decode(&delivery)
		defer response.Body.Close()
		log.Println(delivery)
		if err != nil  {
			log.Printf("Unable to decode delivery response")
			return delivery, errors.New("failed reading response from delivery")
		}
		
		return delivery, nil

	}
	return delivery, errors.New("failed response from delivery")
}


func OrderDelivery(orderID uint) (error) {
	var delivery RequestDelivery
	var contacts models.Contacts
	var dRecipient DeliveryRecipient
	var dCost DeliveryRecipientCost
	var phone Phone
	var toLoc models.Location
	var fromLoc models.Location
	var packages []FullPackage
	var respDelivery ResponseDelivery
	var respEntity Entity


	accessToken, _ := AuthorizeDelivery()
	ctx, cancel := context.WithTimeout(context.Background(), config.ContextDBTimeout)
	// не забываем освободить ресурс
	defer cancel()

	deliveryObj, err := LoadApiDelivery(ctx, config.DB, orderID)
	if err != nil {
		log.Printf("Failed to obtain delivery data for the order %s", strconv.Itoa(int(orderID)) )
		return errors.New("Failed to obtain delivery data for the order")
	}

	if deliveryObj.Method == "door_to_door" {
		delivery.TariffCode = 139
		toLoc.PostalCode = deliveryObj.PostalCode
		toLoc.Address = deliveryObj.Address
		delivery.ToLocation = toLoc
	} else if deliveryObj.Method == "door_to_office" {
		delivery.TariffCode = 138
		delivery.DeliveryPoint = deliveryObj.Code
	}

	for num := 1; num <= int(deliveryObj.Projects); num++ {
		
        var p FullPackage
		var i Item
		p.Number = num
		p.Weight = 0.2 
		p.Height = 0.2 
		p.Width = 0.2
		p.Length = 0.2 
		i.Name = "photobook"
		i.WareKey = "1"
		i.Payment = 0
		p.Items = append(p.Items, i)
		p.Cost = 10000
		p.Amount = 1
		packages = append(packages, p)
		num = num + 1

	}

	fromLoc.PostalCode = "123456"
	fromLoc.Address = "123456"
	delivery.FromLocation = fromLoc

	dCost.Value = int(deliveryObj.Amount)
	delivery.DeliveryRecipientCost = dCost

	contacts = deliveryObj.ContactData
	dRecipient.Name = contacts.FirstName + " " + contacts.LastName
	dRecipient.Email = contacts.Email
	phone.Number = contacts.Phone
	dRecipient.Phones = append(dRecipient.Phones, phone)
	delivery.Recipient = dRecipient
	delivery.Packages = packages

	
	orderURL := "https://api.edu.cdek.ru/v2/orders"
	bodyBytes, _ := json.Marshal(orderURL)
	body := bytes.NewBuffer(bodyBytes)
	log.Println(body)

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, orderURL, body)
	if err != nil {
		log.Printf("Error in placing delivery for the order %s", strconv.Itoa(int(orderID)) )
		return errors.New("failed request to order delivery")
	}
	fullToken := "Bearer " + accessToken
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Authorization", fullToken)

	client := http.DefaultClient
	response, err := client.Do(request)

	if err != nil {
		log.Printf("Error in placing delivery for the order %s", strconv.Itoa(int(orderID)) )
		return errors.New("failed request to order delivery")
	}

	if response.StatusCode == http.StatusInternalServerError {
		log.Printf("Error in placing delivery for the order %s", strconv.Itoa(int(orderID)) )
		return errors.New("failed response 500 to order delivery")
	}

	if response.StatusCode == http.StatusOK {
		err := json.NewDecoder(response.Body).Decode(&respDelivery)
		defer response.Body.Close()
		if err != nil  {
			log.Printf("Error in decoding placed delivery for the order %s", strconv.Itoa(int(orderID)) )
			return errors.New("failed decoding placed delivery")
		}
		respEntity = respDelivery.Entity
		if respEntity.UUID == ""  {
			log.Printf("API error in creating delivery for the order %s", strconv.Itoa(int(orderID)) )
			return errors.New("error placing delivery")
		} else {
			err = AddDeliveryID(ctx, config.DB, orderID, respEntity.UUID) 
			if err != nil  {
				log.Printf("Failed to update delivery api id for the order %s", strconv.Itoa(int(orderID)) )
				return errors.New("Failed to update delivery api id")
			}
		}
		
		return nil

	}
	return errors.New("failed request to order delivery")
}

func CheckDeliveryStatus(orderID uint) (error) {
	var dStatusObj DeliveryStatusCheck
	var dStatus string
	var dtrackingNumber string
	var dEntity DeliveryEntity
	accessToken, _ := AuthorizeDelivery()
	ctx, cancel := context.WithTimeout(context.Background(), config.ContextDBTimeout)
	// не забываем освободить ресурс
	defer cancel()

	deliveryID, deliveryUUID, status, trackingNumber, err := FindDeliveryUUID(ctx, config.DB, orderID) 
	if err != nil {
		log.Printf("Failed to obtain delivery uuid for the order %s", strconv.Itoa(int(orderID)) )
		return errors.New("Failed to obtain delivery uuid for the order")
	}

	
	statusURL := "https://api.edu.cdek.ru/v2/orders/" + deliveryUUID

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, statusURL, nil)
	if err != nil {
		log.Printf("Error in getting delivery data for the order %s", strconv.Itoa(int(orderID)) )
		return errors.New("failed request to get delivery data")
	}
	fullToken := "Bearer " + accessToken
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Authorization", fullToken)

	client := http.DefaultClient
	response, err := client.Do(request)

	if err != nil {
		log.Printf("Error in getting delivery data for the order %s", strconv.Itoa(int(orderID)) )
		return errors.New("failed request to get delivery data")
	}

	if response.StatusCode == http.StatusInternalServerError {
		log.Printf("Error 500 in getting delivery data for the order %s", strconv.Itoa(int(orderID)) )
		return errors.New("failed request to get delivery data 500")
	}

	if response.StatusCode == http.StatusOK {
		err := json.NewDecoder(response.Body).Decode(&dStatusObj)
		defer response.Body.Close()
		if err != nil  {
			log.Printf("Error in decoding status delivery for the order %s", strconv.Itoa(int(orderID)) )
			return errors.New("failed decoding status delivery")
		}
		dEntity = dStatusObj.Entity
		dtrackingNumber = dEntity.CDEKNumber
		lastStatus := dEntity.Statuses[0]
		dStatus = lastStatus.Code
		if trackingNumber == "" {
			err = UpdateTrackingNumber(ctx, config.DB, deliveryID, dtrackingNumber) 
			if err != nil  {
					log.Printf("Failed to update delivery tracking number for the order %s", strconv.Itoa(int(orderID)) )
					return errors.New("Failed to update delivery tracking number")
			}
		}
		if dStatus != status {
			err = UpdateDeliveryStatus(ctx, config.DB, deliveryID, dStatus) 
			if err != nil  {
				log.Printf("Failed to update delivery status for the order %s", strconv.Itoa(int(orderID)) )
				return errors.New("Failed to update delivery status")
			}
		
		}

		
		return nil

	}
	return errors.New("Failed to update delivery status")
}

func RoutineUpdateDeliveryStatus(ctx context.Context, storeDB *pgxpool.Pool) {

	ticker := time.NewTicker(config.UpdateInterval)
	var err error
	var orderList []uint

	jobCh := make(chan uint)
	for i := 0; i < config.WorkersCount; i++ {
		go func() {
			for job := range jobCh {
	
				err = CheckDeliveryStatus(job)
				if err != nil {
					log.Printf("Error happened when updating pending deliveries. Err: %s", err)
					continue
				}
			}
		}()
	}

	for range ticker.C {

		orderList, err = LoadActiveDeliveries(ctx, storeDB)
		if err != nil {
			log.Printf("Error happened when retrieving pending deliveries. Err: %s", err)
			continue
		}

		for _, order := range orderList {
			jobCh <- order

		}

	}
}

// LoadApiDelivery function performs the operation of retrieving delivery info from pgx database with a query.
func LoadApiDelivery(ctx context.Context, storeDB *pgxpool.Pool, orderID uint) (models.ResponseApiDeliveryInfo, error) {

	orderObj := models.ResponseApiDeliveryInfo{}
	var contactData models.Contacts
	var deliveryID uint

	err := storeDB.QueryRow(ctx, "SELECT delivery_id, firstname, lastname, email, phone FROM orders WHERE orders_id = ($1);", orderID).Scan(&deliveryID, &contactData.FirstName, &contactData.LastName, &contactData.Email, &contactData.Phone)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				log.Printf("Error happened when retrieving delivery id info from pgx table. Err: %s", err)
				return orderObj, err
	}
	orderObj.ContactData = contactData
	err = storeDB.QueryRow(ctx, "SELECT method, address, code, postal_code, deliverystatus, amount, deliveryid, trackingnumber FROM delivery WHERE delivery_id = ($1);", deliveryID).Scan(&orderObj.Method, &orderObj.Address, &orderObj.Code, &orderObj.PostalCode, &orderObj.DeliveryStatus, &orderObj.Amount, &orderObj.DeliveryID, &orderObj.TrackingNumber)
	if err != nil && err != pgx.ErrNoRows{
		log.Printf("Error happened when retrieving delivery info from pgx table. Err: %s", err)
		return orderObj, err
	}
	err = storeDB.QueryRow(ctx, "SELECT COUNT(projects_id) FROM orders_has_projects WHERE orders_id = ($1);", orderID).Scan(&orderObj.Projects)
	if err != nil && err != pgx.ErrNoRows{
				log.Printf("Error happened when counting projects for order in pgx table. Err: %s", err)
				return orderObj, err
	}

	return orderObj, nil

}


// AddDeliveryID function performs the operation of adding delivery uuid from pgx database with a query.
func AddDeliveryID(ctx context.Context, storeDB *pgxpool.Pool, orderID uint, uuid string) (error) {

	var deliveryID uint

	err := storeDB.QueryRow(ctx, "SELECT delivery_id FROM orders WHERE orders_id = ($1);", orderID).Scan(&deliveryID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				log.Printf("Error happened when retrieving delivery id info from pgx table. Err: %s", err)
				return err
	}
	_, err = storeDB.Exec(ctx, "UPDATE delivery SET deliveryid = ($1), status = ($2) WHERE delivery_id = ($3);",
			"IN PROGRESS",
			uuid,
			deliveryID,
	)
	if err != nil {
		log.Printf("Error happened when updating delivery uuid into pgx table. Err: %s", err)
		return err
	}


	return nil

}

// FindDeliveryUUID function performs the operation of retrieving delivery uuid from pgx database with a query.
func FindDeliveryUUID(ctx context.Context, storeDB *pgxpool.Pool, orderID uint) (uint, string, string, string, error) {

	var uuid string
	var status string
	var trackingNumber string
	var deliveryID uint

	err := storeDB.QueryRow(ctx, "SELECT delivery_id FROM orders WHERE orders_id = ($1);", orderID).Scan(&deliveryID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				log.Printf("Error happened when retrieving delivery id info from pgx table. Err: %s", err)
				return deliveryID, uuid, status, trackingNumber, err
	}
	err = storeDB.QueryRow(ctx, "SELECT deliveryid, deliverystatus, trackingnumber FROM delivery WHERE delivery_id = ($1);", deliveryID).Scan(&uuid, &status, &trackingNumber)
	if err != nil {
		log.Printf("Error happened when retrieving delivery uuid from pgx table. Err: %s", err)
		return deliveryID, uuid, status, trackingNumber, err
	}
	
	return deliveryID, uuid, status, trackingNumber, nil

}

// UpdateTrackingNumber function performs the operation of updating delivery tracking number from pgx database with a query.
func UpdateTrackingNumber(ctx context.Context, storeDB *pgxpool.Pool, deliveryID uint, dtrackingNumber string) (error) {


	_, err := storeDB.Exec(ctx, "UPDATE delivery SET trackingnumber = ($1) WHERE delivery_id = ($2);",
			dtrackingNumber,
			deliveryID,
	)
	if err != nil {
		log.Printf("Error happened when updating delivery tracking number into pgx table. Err: %s", err)
		return err
	}


	return nil

}

// UpdateDeliveryStatus function performs the operation of updating delivery status from pgx database with a query.
func UpdateDeliveryStatus(ctx context.Context, storeDB *pgxpool.Pool, deliveryID uint, dstatus string) (error) {


	_, err := storeDB.Exec(ctx, "UPDATE delivery SET deliverystatus = ($1) WHERE delivery_id = ($2);",
			dstatus,
			deliveryID,
	)
	if err != nil {
		log.Printf("Error happened when updating delivery status into pgx table. Err: %s", err)
		return err
	}
	if dstatus == "DELIVERED" {
		_, err = storeDB.Exec(ctx, "UPDATE delivery SET status = ($1) WHERE delivery_id = ($2);",
			"COMPLETED",
			deliveryID,
		)
		_, err = storeDB.Exec(ctx, "UPDATE orders SET status = ($1) WHERE delivery_id = ($2);",
			"COMPLETED",
			deliveryID,
		)
	}


	return nil

}