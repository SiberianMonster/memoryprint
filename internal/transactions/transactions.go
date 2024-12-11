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
	uniqueNumber := goodType + strconv.Itoa(int(orderID)) + "attempt" + randomString(100)
	returnURL := "https://memoriprint.ru/paymentresults/" + uniqueNumber
    queryValues := url.Values{}
    queryValues.Add("userName", config.BankUsername)
    queryValues.Add("password", config.BankPassword)
	queryValues.Add("returnUrl", returnURL)
	queryValues.Add("orderNumber", uniqueNumber)
	priceInCents := finalPrice * 100.0
	queryValues.Add("amount", strconv.Itoa(int(priceInCents)))
    registrationURL.RawQuery = queryValues.Encode()
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
		return transaction.FormURL, nil

	}
	return paymentLink, errors.New("failed request to bank")
}

func FindTransactionStatus(orderID uint) (string, error) {

	statusURL := &url.URL{
        Scheme: "https",
        Host:   config.BankDomain,
		Path:   "/payment/rest/getOrderStatusExtended.do",
    }
    queryValues := url.Values{}
    queryValues.Add("userName", config.BankUsername)
    queryValues.Add("password", config.BankPassword)
	queryValues.Add("orderNumber", strconv.Itoa(int(orderID)))
    statusURL.RawQuery = queryValues.Encode()

	ctx, cancel := context.WithTimeout(context.Background(), config.ContextDBTimeout)
	// не забываем освободить ресурс
	defer cancel()
	statusTransaction := "PENDING"


	request, err := http.NewRequestWithContext(ctx, http.MethodPost, statusURL.String(), nil)
	if err != nil {
		log.Printf("Error in checking transaction status for the order %s", strconv.Itoa(int(orderID)))
		return statusTransaction, errors.New("failed response from bank")
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := http.DefaultClient
	response, err := client.Do(request)

	if err != nil {
		log.Printf("Error in getting payment data for the order %s", strconv.Itoa(int(orderID)))
		return statusTransaction, errors.New("failed response from bank")
	}

	if response.StatusCode == http.StatusInternalServerError {
		log.Printf("Internal server error happened when getting payment data %s order ", strconv.Itoa(int(orderID)))
		return statusTransaction, errors.New("failed response from bank")
	}

	if response.StatusCode == http.StatusOK {
		var transaction models.ResponseTransactionStatus
		err := json.NewDecoder(response.Body).Decode(&transaction)
		defer response.Body.Close()
		if err != nil {
			log.Printf("Unable to decode bank response for the order %s",  strconv.Itoa(int(orderID)))
			return statusTransaction, errors.New("failed reading response from bank")
		}
		if transaction.ActionCode != 0 {
			err = orderstorage.UpdateUnSuccessfulTransaction(ctx, config.DB, orderID)
			if err != nil {
				log.Printf("Unable to update transaction entry for the order %s", strconv.Itoa(int(orderID)))
			}
			log.Printf("Unsuccessful transaction for the order %s",  strconv.Itoa(int(orderID)))
			return "UNSUCCESSFUL", errors.New("failed reading response from bank")
		}
		err = orderstorage.UpdateSuccessfulTransaction(ctx, config.DB, orderID)
		if err != nil {
			log.Printf("Unable to update transaction entry for the order %s", strconv.Itoa(int(orderID)))
		}
		return "SUCCESSFUL", nil

	}
	return statusTransaction, errors.New("failed response from bank")
}
func CancelTransaction(orderID uint) error {

	ctx, cancel := context.WithTimeout(context.Background(), config.ContextDBTimeout)
	// не забываем освободить ресурс
	defer cancel()
	log.Println(orderID)
	banktransactionID, _ := orderstorage.GetBankTransactionID(ctx, config.DB, orderID)
	cancellationURL := &url.URL{
        Scheme: "https",
        Host:   config.BankDomain,
        Path:   "/payment/rest/reverse.do",
    }
	queryValues := url.Values{}
    queryValues.Add("userName", config.BankUsername)
    queryValues.Add("password", config.BankPassword)
	queryValues.Add("orderNumber", banktransactionID)
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
		if transaction.ErrorCode != "7" && transaction.ErrorCode != "6"{

			return errors.New("failed reading response from bank")
		}

		return nil

	}
	return errors.New("failed response from bank")
}