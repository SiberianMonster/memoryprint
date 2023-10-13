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
func AddOrder(ctx context.Context, storeDB *pgxpool.Pool, orderLink string, userID uint) (uint, error) {

	var orderID uint
	t := time.Now()
	_, err = storeDB.Exec(ctx, "INSERT INTO orders (link, uploaded_at, last_updated_at, status, users_id) VALUES ($1, $2, $3, $4, $5);",
		orderLink,
		t,
		t,
		"SUBMITTED",
		userID,
	)
	if err != nil {
		log.Printf("Error happened when inserting a new order entry into pgx table. Err: %s", err)
		return userID, err
	}
	err = storeDB.QueryRow(ctx, "SELECT order_id FROM orders WHERE link=($1);", orderLink).Scan(&orderID)
	if err != nil {
		log.Printf("Error happened when retrieving usersid from the db. Err: %s", err)
		return orderID, err
	}

	return orderID, nil

}

// DeleteOrder function performs the operation of deleting order by id from pgx database with a query.
func DeleteOrder(ctx context.Context, storeDB *pgxpool.Pool, orderID uint) (uint, error) {

	_, err = storeDB.Exec(ctx, "DELETE FROM orders WHERE order_id=($1);",
		orderID,
	)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("Error happened when deleting order from pgx table. Err: %s", err)
		return orderID, err
	}

	_, err = storeDB.Exec(ctx, "DELETE FROM pa_has_orders WHERE order_id=($1);",
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

	_, err = storeDB.Exec(ctx, "INSERT INTO pa_has_orders (users_id, order_id) VALUES ($1, $2);",
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
	rows, err := storeDB.Query(ctx, "SELECT link, status FROM orders WHERE users_id = ($1);", userID)
	if err != nil {
		log.Printf("Error happened when retrieving orders from pgx table. Err: %s", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var order models.UserOrder
		if err = rows.Scan(&order.Link, &order.Status); err != nil {
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

// RetrieveOrders function performs the operation of retrieving all orders from pgx database with a query (for admins).
func RetrieveOrders(ctx context.Context, storeDB *pgxpool.Pool) ([]models.AdminOrder, error) {

	var orderslice []models.AdminOrder
	rows, err := storeDB.Query(ctx, "SELECT order_id, link, status, uploaded_at, last_updated_at, users_id FROM orders;")
	if err != nil {
		log.Printf("Error happened when retrieving orders from pgx table. Err: %s", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var order models.AdminOrder
		var uploadTimeStorage time.Time
		var updateTimeStorage time.Time
		if err = rows.Scan(&order.OrderID, &order.Link, &order.Status, &uploadTimeStorage, &updateTimeStorage, &order.UsersID); err != nil {
			log.Printf("Error happened when scanning orders. Err: %s", err)
			return nil, err
		}
		err = storeDB.QueryRow(ctx, "SELECT email FROM users WHERE users_id = ($1);", order.UsersID).Scan(&order.UserEmail)
		if err != nil && err != pgx.ErrNoRows {
			log.Printf("Error happened when retrieving user email from db. Err: %s", err)
			return nil, err
		}
		err = storeDB.QueryRow(ctx, "SELECT users_id FROM pa_has_orders WHERE order_id = ($1);", order.OrderID).Scan(&order.PaID)
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

	_, err = storeDB.Exec(ctx, "UPDATE orders SET status = ($1) WHERE order_id = ($2);",
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
	rows, err := storeDB.Query(ctx, "SELECT order_id FROM pa_has_orders WHERE users_id = ($1);", userID)
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
		err = storeDB.QueryRow(ctx, "SELECT link, status FROM orders WHERE order_id = ($1) ORDER BY last_updated_at;", orderID).Scan(&order.Link, &order.Status)
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

