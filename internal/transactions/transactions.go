package transactions

import (
	"context"
	"encoding/json"
	"github.com/SiberianMonster/memoryprint/internal/config"
	"github.com/SiberianMonster/memoryprint/internal/models"
	"github.com/SiberianMonster/memoryprint/internal/orderstorage"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"errors"
    "math/rand"
    "time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func randomString(l int) string {
    bytes := make([]byte, l)
    for i := 0; i < l; i++ {
        bytes[i] = byte(randInt(65, 90))
    }
    return string(bytes)
}

func randInt(min int, max int) int {
    return min + rand.Intn(max-min)
}

func CreateTransaction(orderID uint, finalPrice float64, goodType string) (string, error) {
	var paymentLink string
	ctx, cancel := context.WithTimeout(context.Background(), config.ContextDBTimeout)
	// не забываем освободить ресурс
	defer cancel()
	
	registrationURL := &url.URL{
        Scheme: "https",
        Host:   config.BankDomain,
		Path:   "/payment/rest/registerPreAuth.do",
    }
	rand.Seed(time.Now().UnixNano())
	uniqueNumber := goodType + strconv.Itoa(int(orderID)) + "attempt" + randomString(8)
	successURL := "https://front.memoryprint.dev.startup-it.ru/payment/success?id=" + strconv.Itoa(int(orderID))
	failureURL := "https://front.memoryprint.dev.startup-it.ru/payment/fail?id=" + strconv.Itoa(int(orderID))
    queryValues := url.Values{}
    queryValues.Add("userName", config.BankUsername)
    queryValues.Add("password", config.BankPassword)
	queryValues.Add("returnUrl", successURL)
	queryValues.Add("failUrl", failureURL)
	queryValues.Add("orderNumber", uniqueNumber)
	priceInCents := finalPrice * 100.0
	queryValues.Add("amount", strconv.Itoa(int(priceInCents)))
    registrationURL.RawQuery = queryValues.Encode()
	log.Printf("Creating payment link for the order %s", strconv.Itoa(int(orderID)) )
	log.Println(registrationURL.String())
    
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, registrationURL.String(), nil)
	if err != nil {
		log.Println(err)
		log.Printf("Error in creating request for the order %s", strconv.Itoa(int(orderID)) )
		return paymentLink, errors.New("failed request to bank")
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := http.DefaultClient
	response, err := client.Do(request)

	if err != nil {
		log.Printf("Error in getting payment url for the order %s", strconv.Itoa(int(orderID)) )
		return paymentLink, errors.New("failed response from bank")
	}

	if response.StatusCode == http.StatusInternalServerError {
		log.Printf("Internal server error happened when getting payment url %s order ", strconv.Itoa(int(orderID)))
		return paymentLink, errors.New("failed response from bank")
	}

	if response.StatusCode == http.StatusOK {
		var transaction models.ResponseTransaction
		err := json.NewDecoder(response.Body).Decode(&transaction)
		log.Println(transaction)
		defer response.Body.Close()
		if err != nil {
			log.Printf("Error in getting payment url for the order %s", strconv.Itoa(int(orderID)) )
			return paymentLink, errors.New("failed response from bank")
		}
		err = orderstorage.UpdateTransaction(ctx, config.DB, orderID, transaction, finalPrice, goodType)
		if err != nil {
			log.Printf("Unable to update transaction entry for the order %s", strconv.Itoa(int(orderID)))
		}
		if transaction.FormURL == "" {
			log.Printf("Internal server error happened when getting payment url %s order ", strconv.Itoa(int(orderID)))
			return paymentLink, errors.New("failed response from bank")
		}
		return transaction.FormURL, nil

	}
	return paymentLink, errors.New("failed request to bank")
}

func FindTransactionStatus(order models.PaidOrderObj) (string, error) {

	
	var tID uint
	var transactionNumber string
	statusTransaction := "PENDING"
	log.Printf("Finding transaction status for the order %s", strconv.Itoa(int(order.OrdersID)) )

	ctx, cancel := context.WithTimeout(context.Background(), config.ContextDBTimeout)
	// не забываем освободить ресурс
	defer cancel()
	terr := config.DB.QueryRow(ctx, "SELECT transactions_id FROM orders_has_transactions WHERE orders_id = ($1) ORDER BY transactions_id DESC LIMIT 1;", order.OrdersID).Scan(&tID)
	if terr != nil {
		log.Printf("Error happened when retrieving transaction info from pgx table. Err: %s", terr)
		return statusTransaction, terr
	}
	terr = config.DB.QueryRow(ctx, "SELECT bankorderid FROM transactions WHERE transactions_id = ($1);",
			tID).Scan(&transactionNumber)
	if terr != nil {
		log.Printf("Error happened when updating transaction status into pgx table. Err: %s", terr)
		return statusTransaction, terr
	}
	if transactionNumber == "" {
		log.Println(order.OrdersID)
		return statusTransaction, errors.New("empty transaction")
	}
	statusURL := &url.URL{
        Scheme: "https",
        Host:   config.BankDomain,
		Path:   "/payment/rest/getOrderStatusExtended.do",
    }
    queryValues := url.Values{}
    queryValues.Add("userName", config.BankUsername)
    queryValues.Add("password", config.BankPassword)
	queryValues.Add("orderId", transactionNumber)
    statusURL.RawQuery = queryValues.Encode()

	
	
	log.Println(statusURL.String())


	request, err := http.NewRequestWithContext(ctx, http.MethodPost, statusURL.String(), nil)
	if err != nil {
		log.Printf("Error in checking transaction status for the order %s", strconv.Itoa(int(order.OrdersID)))
		return statusTransaction, errors.New("failed response from bank")
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := http.DefaultClient
	response, err := client.Do(request)

	if err != nil {
		log.Printf("Error in getting payment data for the order %s", strconv.Itoa(int(order.OrdersID)))
		return statusTransaction, errors.New("failed response from bank")
	}

	if response.StatusCode == http.StatusInternalServerError {
		log.Printf("Internal server error happened when getting payment data %s order ", strconv.Itoa(int(order.OrdersID)))
		return statusTransaction, errors.New("failed response from bank")
	}

	if response.StatusCode == http.StatusOK {
		var transaction models.ResponseTransactionStatus
		err := json.NewDecoder(response.Body).Decode(&transaction)
		defer response.Body.Close()
		if err != nil {
			log.Printf("Unable to decode bank response for the order %s",  strconv.Itoa(int(order.OrdersID)))
			return statusTransaction, errors.New("failed reading response from bank")
		}
		if transaction.OrderStatus == 1 {
			err = orderstorage.UpdateSuccessfulTransaction(ctx, config.DB, order.OrdersID)
			if err != nil {
				log.Printf("Unable to update transaction entry for the order %s", strconv.Itoa(int(order.OrdersID)))
			}
			log.Printf("Successful transaction for the order %s",  strconv.Itoa(int(order.OrdersID)))
			return "SUCCESSFUL", nil
		}
		if transaction.OrderStatus != 0 {
			err = orderstorage.UpdateUnSuccessfulTransaction(ctx, config.DB, order.OrdersID)
			if err != nil {
				log.Printf("Unable to update unsuccessfull transaction entry for the order %s", strconv.Itoa(int(order.OrdersID)))
			}
			return "UNSUCCESSFUL", nil
		}
		return "PENDING", nil

	}
	return statusTransaction, errors.New("failed response from bank")
}
func CancelTransaction(orderID uint) error {

	ctx, cancel := context.WithTimeout(context.Background(), config.ContextDBTimeout)
	// не забываем освободить ресурс
	defer cancel()
	log.Printf("Cancel transaction for the order %s", strconv.Itoa(int(orderID)) )

	banktransactionID, _ := orderstorage.GetBankTransactionID(ctx, config.DB, orderID)
	cancellationURL := &url.URL{
        Scheme: "https",
        Host:   config.BankDomain,
        Path:   "/payment/rest/reverse.do",
    }
	queryValues := url.Values{}
    queryValues.Add("userName", config.BankUsername)
    queryValues.Add("password", config.BankPassword)
	queryValues.Add("orderId", banktransactionID)
    cancellationURL.RawQuery = queryValues.Encode()

	


	request, err := http.NewRequestWithContext(ctx, http.MethodPost, cancellationURL.String(), nil)
	if err != nil {
		log.Printf("Error in creating request for the order %s", strconv.Itoa(int(orderID)))
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := http.DefaultClient
	response, err := client.Do(request)

	if err != nil {
		log.Printf("Error in getting cancellatiom data for the order %s", strconv.Itoa(int(orderID)))
		return errors.New("failed response from bank")
	}

	if response.StatusCode == http.StatusInternalServerError {
		log.Printf("Internal server error happened when getting cancellatiom data %s order ", strconv.Itoa(int(orderID)))
		return errors.New("failed response from bank")
	}

	if response.StatusCode == http.StatusOK {
		var transaction models.ResponseTransactionCancel
		log.Println(response.Body)
		err := json.NewDecoder(response.Body).Decode(&transaction)
		defer response.Body.Close()
		log.Println(transaction)
		if err != nil {
			log.Printf("Unable to decode bank response for the order %s",  strconv.Itoa(int(orderID)))
			return errors.New("failed reading response from bank")
		}
		// need to be fixed
		if transaction.ErrorCode != "0"{

			return errors.New("failed reading response from bank")
		}

		return nil

	}
	return errors.New("failed response from bank")
}

func RoutineUpdateTransactionsStatus(ctx context.Context, storeDB *pgxpool.Pool) {

	ticker := time.NewTicker(config.UpdateInterval)
	var err error
	var orderList []models.PaidOrderObj

	jobCh := make(chan models.PaidOrderObj)
	for i := 0; i < config.WorkersCount; i++ {
		go func() {
			for job := range jobCh {
	
				_, err = FindTransactionStatus(job)
				if err != nil {
					log.Printf("Error happened when updating pending orders. Err: %s", err)
					continue
				}
			}
		}()
	}

	for range ticker.C {

		orderList, err = orderstorage.LoadPaymentInProgressOrders(ctx, storeDB)
		if err != nil {
			log.Printf("Error happened when retrieving pending orders. Err: %s", err)
			continue
		}

		for _, order := range orderList {
			jobCh <- order

		}

	}
}
