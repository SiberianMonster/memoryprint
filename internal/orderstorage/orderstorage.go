// Storage package contains functions for storing photos and projects in a pgx database.
//
// Available at https://github.com/SiberianMonster/memoryprint/tree/development/internal/orderstorage
package orderstorage

import (
	"context"
	"errors"
	"log"
	"time"
	"github.com/SiberianMonster/memoryprint/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var err error


// AddOrder function performs the operation of adding a new photobook order to the db.
func AddOrder(ctx context.Context, storeDB *pgxpool.Pool, orderObj models.AdminOrder, userID uint) (uint, error) {

	var orderID uint
	t := time.Now()
	_, err = storeDB.Exec(ctx, "INSERT INTO orders (link, pagesnum, created_at, covertype, bindingtype, papertype, last_updated_at, promooffers_id, status, users_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);",
		orderObj.Link,
		orderObj.Pagesnum,
		t,
		orderObj.Covertype,
		orderObj.Bindingtype,
		orderObj.Papertype,
		t,
		orderObj.PromooffersID,
		"SUBMITTED",
		userID,
	)
	if err != nil {
		log.Printf("Error happened when inserting a new order entry into pgx table. Err: %s", err)
		return userID, err
	}
	err = storeDB.QueryRow(ctx, "SELECT orders_id FROM orders WHERE link=($1);", orderObj.Link).Scan(&orderID)
	if err != nil {
		log.Printf("Error happened when retrieving usersid from the db. Err: %s", err)
		return orderID, err
	}

	return orderID, nil

}

// DeleteOrder function performs the operation of deleting order by id from pgx database with a query.
func DeleteOrder(ctx context.Context, storeDB *pgxpool.Pool, orderID uint) (uint, error) {

	_, err = storeDB.Exec(ctx, "DELETE FROM orders WHERE orders_id=($1);",
		orderID,
	)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("Error happened when deleting order from pgx table. Err: %s", err)
		return orderID, err
	}

	_, err = storeDB.Exec(ctx, "DELETE FROM pa_has_orders WHERE orders_id=($1);",
		orderID,
	)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("Error happened when deleting order from pgx table. Err: %s", err)
		return orderID, err
	}

	return orderID, nil

}

// AddOrderResponsible function performs the operation of adding entry into pa_has_orders to the db.
func AddOrderResponsible(ctx context.Context, storeDB *pgxpool.Pool, paID uint, orderID uint) (uint, error) {

	_, err = storeDB.Exec(ctx, "INSERT INTO pa_has_orders (users_id, orders_id) VALUES ($1, $2);",
		paID,
		orderID,
	)
	if err != nil {
		log.Printf("Error happened when inserting a new entry into pa_has_orders table. Err: %s", err)
		return paID, err
	}

	return paID, nil

}

// RetrieveUserOrders function performs the operation of retrieving user orders from pgx database with a query.
func RetrieveUserOrders(ctx context.Context, storeDB *pgxpool.Pool, userID uint) ([]models.UserOrder, error) {

	var orderslice []models.UserOrder
	rows, err := storeDB.Query(ctx, "SELECT link, status, pagesnum, covertype, bindingtype, papertype, promooffers_id FROM orders WHERE users_id = ($1);", userID)
	if err != nil {
		log.Printf("Error happened when retrieving orders from pgx table. Err: %s", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var order models.UserOrder
		if err = rows.Scan(&order.Link, &order.Status, &order.Pagesnum, &order.Covertype, &order.Bindingtype, &order.Papertype, &order.PromooffersID); err != nil {
			log.Printf("Error happened when scanning orders. Err: %s", err)
			return nil, err
		}
		orderslice = append(orderslice, order)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error happened when retrieving orders from pgx table. Err: %s", err)
		return nil, err
	}
	return orderslice, nil

}

// RetrieveOrderStatus function performs the operation of retrieving order status from pgx database with a query.
func RetrieveOrderStatus(ctx context.Context, storeDB *pgxpool.Pool, userID uint, orderID uint) (string, error) {

	var orderStatus string
	err := storeDB.QueryRow(ctx, "SELECT status FROM orders WHERE users_id = ($1) and orders_id = ($2);", userID, orderID).Scan(&orderStatus)
	if err != nil {
		log.Printf("Error happened when retrieving order status from db. Err: %s", err)
		return orderStatus, err
	}

	return orderStatus, nil

}

// RetrieveOrders function performs the operation of retrieving all orders from pgx database with a query (for admins).
func RetrieveOrders(ctx context.Context, storeDB *pgxpool.Pool) ([]models.AdminOrder, error) {

	var orderslice []models.AdminOrder
	rows, err := storeDB.Query(ctx, "SELECT orders_id, link, status, uploaded_at, last_updated_at, users_id FROM orders;")
	if err != nil {
		log.Printf("Error happened when retrieving orders from pgx table. Err: %s", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var order models.AdminOrder
		var uploadTimeStorage time.Time
		var updateTimeStorage time.Time
		if err = rows.Scan(&order.OrderID, &order.Link, &order.Status,  &order.Pagesnum, &order.Covertype, &order.Bindingtype, &order.Papertype, &order.PromooffersID, &uploadTimeStorage, &updateTimeStorage, &order.UsersID); err != nil {
			log.Printf("Error happened when scanning orders. Err: %s", err)
			return nil, err
		}
		err = storeDB.QueryRow(ctx, "SELECT email FROM users WHERE users_id = ($1);", order.UsersID).Scan(&order.UserEmail)
		if err != nil && err != pgx.ErrNoRows {
			log.Printf("Error happened when retrieving user email from db. Err: %s", err)
			return nil, err
		}
		err = storeDB.QueryRow(ctx, "SELECT users_id FROM pa_has_orders WHERE orders_id = ($1);", order.OrderID).Scan(&order.PaID)
		if err != nil && err != pgx.ErrNoRows {
			log.Printf("Error happened when retrieving print agency for the order from db. Err: %s", err)
		}
		order.UploadedAt = uploadTimeStorage.Format(time.RFC3339)
		order.LastUpdatedAt = updateTimeStorage.Format(time.RFC3339)
		orderslice = append(orderslice, order)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error happened when retrieving orders from pgx table. Err: %s", err)
		return nil, err
	}
	return orderslice, nil

}

// UpdateOrderPaymentStatus function performs the operation of updating order payment status in pgx database with a query.
func UpdateOrderPaymentStatus(ctx context.Context, storeDB *pgxpool.Pool, status string, orderID uint) (error) {

	_, err = storeDB.Exec(ctx, "UPDATE orders SET status = ($1) WHERE orders_id = ($2);",
	status,
	orderID,
	)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("Error happened when updating user id for view project pgx table. Err: %s", err)
		return err
	}
	
	return nil

}

// PARetrieveOrders function performs the operation of retrieving orders assigned to the printing agent from pgx database with a query.
func PARetrieveOrders(ctx context.Context, storeDB *pgxpool.Pool, userID uint) ([]models.UserOrder, error) {

	var orderslice []models.UserOrder
	rows, err := storeDB.Query(ctx, "SELECT orders_id FROM pa_has_orders WHERE users_id = ($1);", userID)
	if err != nil {
		log.Printf("Error happened when retrieving orders from pgx table. Err: %s", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var order models.UserOrder
		var orderID uint
		if err = rows.Scan(&orderID); err != nil {
			log.Printf("Error happened when scanning orders. Err: %s", err)
			return nil, err
		}
		err = storeDB.QueryRow(ctx, "SELECT link, status FROM orders WHERE orders_id = ($1) ORDER BY last_updated_at;", orderID).Scan(&order.Link, &order.Status)
		if err != nil && err != pgx.ErrNoRows {
			log.Printf("Error happened when retrieving user email from db. Err: %s", err)
			return nil, err
		}
		
		orderslice = append(orderslice, order)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error happened when retrieving orders from pgx table. Err: %s", err)
		return nil, err
	}
	return orderslice, nil

}


// RetrieveAllPrices function performs the operation of retrieving all prices from the db.
func RetrieveAllPrices(ctx context.Context, storeDB *pgxpool.Pool) ([]models.Prices, error) {

	var priceslice []models.Prices
	rows, err := storeDB.Query(ctx, "SELECT prices_id, price, pagesnum, priceperpage, covertype, bindingtype, papertype FROM prices;")
	if err != nil {
		log.Printf("Error happened when retrieving prices from pgx table. Err: %s", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var price models.Prices
		if err = rows.Scan(&price.PricesID, &price.Price, &price.Pagesnum, &price.Priceperpage, &price.Covertype, &price.Bindingtype, &price.Papertype); err != nil {
			log.Printf("Error happened when scanning layouts. Err: %s", err)
			return nil, err
		}
		
		priceslice = append(priceslice, price)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error happened when retrieving prices from pgx table. Err: %s", err)
		return nil, err
	}

	return priceslice, nil

}


// CheckPromoCode function performs the operation of checking the validity of a promo code in pgx database with a query.
func CheckPromoCode(ctx context.Context, storeDB *pgxpool.Pool, promocode string) (models.PromoOffer, error) {

	var promooffer models.PromoOffer
	var expirationTime time.Time
	err = storeDB.QueryRow(ctx, "SELECT promooffers_id, discount, is_onetime, is_used, expires_at FROM users WHERE code=($1);", promocode).Scan(&promooffer.PromooffersID, &promooffer.Discount, &promooffer.ISOnetime, &promooffer.ISUsed, &expirationTime)
	if err != nil {
		log.Printf("Error happened when retrieving promocode from the db. Err: %s", err)
		return promooffer, errors.New("Unable to find the promo code in table")
	}
	today := time.Now() 
	if expirationTime.Before(today) {
		log.Printf("Promooffer already expired on %s", expirationTime.Format(time.RFC3339))
		return promooffer, errors.New("Promooffer expired")
	}
	if promooffer.ISOnetime == true && promooffer.ISUsed == true {
		log.Printf("Promooffer already used")
		return promooffer, errors.New("Promooffer already spent")
	}
	promooffer.ExpiresAt = expirationTime.Format(time.RFC3339)
	return promooffer, nil
	
}

