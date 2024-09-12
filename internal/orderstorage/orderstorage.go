// Storage package contains functions for storing photos and projects in a pgx database.
//
// Available at https://github.com/SiberianMonster/memoryprint/tree/development/internal/orderstorage
package orderstorage

import (
	"context"
	"errors"
	"github.com/SiberianMonster/memoryprint/internal/delivery"
	"github.com/SiberianMonster/memoryprint/internal/models"
	"github.com/SiberianMonster/memoryprint/internal/projectstorage"
	"github.com/SiberianMonster/memoryprint/internal/userstorage"
	"github.com/SiberianMonster/memoryprint/internal/emailutils"
	"github.com/SiberianMonster/memoryprint/internal/config"
	"log"
	"time"
	"strconv"
	"math"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var err error

func CalculateBasePrice(ctx context.Context, storeDB *pgxpool.Pool, size string, variant string, cover string, surface string, countPages uint) (float64, error) {

	var totalBaseprice float64
	var basePrice float64
	var extraPriceperpage float64

	err := storeDB.QueryRow(ctx, "SELECT baseprice, extrapage FROM prices WHERE size = ($1) AND variant = ($2) AND cover = ($3) AND surface = ($4);", size, variant, cover, surface).Scan(&basePrice, &extraPriceperpage)
	
	if err != nil {
		log.Printf("Error happened when retrieving base price from pgx table. Err: %s", err)
		return totalBaseprice, err
	}
	
	log.Println(extraPriceperpage)
	log.Println(basePrice)
	extraPrice := extraPriceperpage*float64((countPages-23))
	totalBaseprice = basePrice + extraPrice


	return totalBaseprice, nil
	
}

func FindPrice(ctx context.Context, storeDB *pgxpool.Pool, size string, variant string, cover string) (float64, float64, error) {

	var basePrice float64
	var extraPriceperpage float64

	err := storeDB.QueryRow(ctx, "SELECT baseprice, extrapage FROM prices WHERE size = ($1) AND variant = ($2) AND cover = ($3);", size, variant, cover).Scan(&basePrice, &extraPriceperpage)
	
	if err != nil {
		log.Printf("Error happened when retrieving base price from pgx table. Err: %s", err)
		return basePrice, extraPriceperpage, err
	}
	
	return basePrice, extraPriceperpage, nil
	
}

func CalculateAlternativePrice(ctx context.Context, storeDB *pgxpool.Pool, size string, variant string, cover string, surface string, countPages uint) (float64, float64, error) {

	var totalPageprice float64
	var totalCoverprice float64
	var altPagePrice float64
	var altCoverPrice float64
	var extraPriceperpage float64
	var err error
	log.Println(variant)
	log.Println(surface)
	log.Println(cover)
	if surface == "GLOSS" {
		err = storeDB.QueryRow(ctx, "SELECT baseprice, extrapage FROM prices WHERE size = ($1) AND variant = ($2) AND surface = ($3) AND cover = ($4);", size, variant, "MATTE", cover).Scan(&altPagePrice, &extraPriceperpage)
	} else if surface == "MATTE" {
		err = storeDB.QueryRow(ctx, "SELECT baseprice, extrapage FROM prices WHERE size = ($1) AND variant = ($2) AND surface = ($3) AND cover = ($4);", size, variant, "GLOSS", cover).Scan(&altPagePrice, &extraPriceperpage)
	}
	if err != nil {
		log.Printf("Error happened when retrieving alternative price from pgx table. Err: %s", err)
		return 0.0, 0.0, err
	}
	log.Println(altPagePrice)
	log.Println(extraPriceperpage)
	extraPrice := extraPriceperpage*float64(countPages-23)
	totalPageprice = altPagePrice + extraPrice
	log.Println(countPages)
	log.Println(extraPrice)
	log.Println(totalPageprice)

	if cover == "HARD" {
		err = storeDB.QueryRow(ctx, "SELECT baseprice, extrapage FROM prices WHERE size = ($1) AND variant = ($2) AND surface = ($3) AND cover = ($4);", size, variant, surface, "LEATHERETTE").Scan(&altCoverPrice, &extraPriceperpage)
	} else if cover == "LEATHERETTE" {
		err = storeDB.QueryRow(ctx, "SELECT baseprice, extrapage FROM prices WHERE size = ($1) AND variant = ($2) AND surface = ($3) AND cover = ($4);", size, variant, surface, "HARD").Scan(&altCoverPrice, &extraPriceperpage)
	}
	if err != nil {
		log.Printf("Error happened when retrieving alternative price from pgx table. Err: %s", err)
		return 0.0, 0.0, err
	}
	
	extraPrice = extraPriceperpage*float64(countPages-23)
	totalCoverprice = altCoverPrice + extraPrice
	log.Println(extraPrice)
	log.Println(totalPageprice)
	log.Println(totalCoverprice)


	return totalPageprice, totalCoverprice, nil
	
}

func CalculateBasePriceByID(ctx context.Context, storeDB *pgxpool.Pool, pID uint) (float64, error) {

	var totalBaseprice float64
	var basePrice float64
	var extraPriceperpage float64
	var size, variant, cover, surface string
	var countPages int

	err := storeDB.QueryRow(ctx, "SELECT size, variant, cover, surface, count_pages FROM projects WHERE projects_id = ($1);", pID).Scan(&size, &variant, &cover, &surface, &countPages)
	
	if err != nil {
		log.Printf("Error happened when retrieving project data from pgx table. Err: %s", err)
		return totalBaseprice, err
	}

	err = storeDB.QueryRow(ctx, "SELECT baseprice, extrapage FROM prices WHERE size = ($1) AND variant = ($2) AND cover = ($3) AND surface = ($4);", size, variant, cover, surface).Scan(&basePrice, &extraPriceperpage)
	
	if err != nil {
		log.Printf("Error happened when retrieving base price from pgx table. Err: %s", err)
		return totalBaseprice, err
	}
	
	extraPrice := extraPriceperpage*float64((countPages-23))
	totalBaseprice = basePrice + extraPrice
	log.Println(extraPrice)
	log.Println(totalBaseprice)


	return totalBaseprice, nil
	
}

// LoadCart function performs the operation of loading cart for user from pgx database with a query.
func LoadCart(ctx context.Context, storeDB *pgxpool.Pool, userID uint) (models.ResponseCart, error) {

	var responseCart models.ResponseCart
	responseCart.Projects = []models.CartObj{}
	var orderID uint
	err := storeDB.QueryRow(ctx, "SELECT orders_id FROM orders WHERE users_id = ($1) and status = ($2);", userID, "AWAITING PAYMENT").Scan(&orderID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				log.Printf("Error happened when retrieving unpaid order info from pgx table. Err: %s", err)
				return responseCart, err
	}
	log.Println(orderID)
	rows, err := storeDB.Query(ctx, "SELECT projects_id FROM orders_has_projects WHERE orders_id = ($1);", orderID)
	if err != nil {
		log.Printf("Error happened when retrieving order projects from pgx table. Err: %s", err)
		return responseCart, err
	}
	defer rows.Close()


	for rows.Next() {
		var photobook models.CartObj
		var pID uint
		var leatherID *uint
		if err = rows.Scan(&pID); err != nil {
			log.Printf("Error happened when scanning projects. Err: %s", err)
			return responseCart, err
		}
		log.Println(pID)

		err := storeDB.QueryRow(ctx, "SELECT name, size, variant, paper, cover, count_pages, leather_id FROM projects WHERE projects_id = ($1);", pID).Scan(&photobook.Name, &photobook.Size, &photobook.Variant, &photobook.Surface, &photobook.Cover, &photobook.CountPages, &leatherID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			log.Printf("Error happened when retrieving project info from pgx table. Err: %s", err)
			return responseCart, err
		}
		if photobook.Cover == "LEATHERETTE" && leatherID != nil {
			err := storeDB.QueryRow(ctx, "SELECT colourlink FROM leather WHERE leather_id = ($1);", leatherID).Scan(&photobook.LeatherImage)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				log.Printf("Error happened when retrieving leather cover info from pgx table. Err: %s", err)
				return responseCart, err
			}
		}
		photobook.BasePrice, err = CalculateBasePrice(ctx, storeDB, photobook.Size, photobook.Variant, photobook.Cover, photobook.Surface, uint(photobook.CountPages))
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			log.Printf("Error happened when counting baseprice. Err: %s", err)
			return responseCart, err
		}
		photobook.UpdatedPagesPrice, photobook.UpdatedCoverPrice, err = CalculateAlternativePrice(ctx, storeDB, photobook.Size, photobook.Variant, photobook.Cover, photobook.Surface, uint(photobook.CountPages))
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			log.Printf("Error happened when counting alternative price. Err: %s", err)
			return responseCart, err
		}
		photobook.ProjectID = pID
		photobook.FrontPage, err = projectstorage.RetrieveFrontPage(ctx, storeDB, pID, false)
		responseCart.Projects = append(responseCart.Projects, photobook)

	}

	return responseCart, nil

}

// CreateOrder function performs the operation of creating a new order for project print for user from pgx database with a query.
func CreateOrder(ctx context.Context, storeDB *pgxpool.Pool, userID uint, order models.NewOrder) (uint, error) {

	var orderID uint
	t := time.Now()
	
	_, err := storeDB.Exec(ctx, "UPDATE projects SET preview_link = ($1), print_link = ($2), status = ($3) WHERE projects_id = ($4);",
			order.PreviewLink,
			order.PrintLink,
			"PUBLISHED",
			order.ProjectID,
	)
	if err != nil {
			log.Printf("Error happened when updating project to published into pgx table. Err: %s", err)
			return orderID, err
	}
	err = storeDB.QueryRow(ctx, "SELECT orders_id FROM orders WHERE users_id = ($1) and status = ($2) ORDER BY created_at;", userID, "AWAITING PAYMENT").Scan(&orderID)
	if err == nil {
		_, err := storeDB.Exec(ctx, "UPDATE orders SET last_updated_at = ($1) WHERE orders_id = ($2);",
			t,
			orderID,
		)
		if err != nil {
			log.Printf("Error happened when updating order last updated at into pgx table. Err: %s", err)
			return orderID, err
		}
		_, err = storeDB.Exec(ctx, "INSERT INTO orders_has_projects (orders_id, projects_id) VALUES ($1, $2);",
			orderID,
			order.ProjectID) 
		if err != nil {
				log.Printf("Error happened when adding new project to order into pgx table. Err: %s", err)
				return orderID, err
		}
	} else if errors.Is(err, pgx.ErrNoRows) {
		
		err := storeDB.QueryRow(ctx, "INSERT INTO orders (status, created_at, last_updated_at, users_id) VALUES ($1, $2, $3, $4) RETURNING orders_id ;",
		"AWAITING PAYMENT",
		t,
		t,
		order.ProjectID).Scan(&orderID)
		if err != nil {
			log.Printf("Error happened when creating order entry into pgx table. Err: %s", err)
			return orderID, err
		}
		_, err = storeDB.Exec(ctx, "INSERT INTO orders_has_projects (orders_id, projects_id) VALUES ($1, $2);",
		orderID,
		order.ProjectID) 
		if err != nil {
			log.Printf("Error happened when adding new project to order into pgx table. Err: %s", err)
			return orderID, err
		}
	} else if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("Error happened when creating order into pgx table. Err: %s", err)
		return orderID, err
	}
	
	return orderID, nil

}

// OrderPayment function performs the operation of creating payment for the order from pgx database with a query.
func OrderPayment(ctx context.Context, storeDB *pgxpool.Pool, orderObj models.RequestOrderPayment, userID uint) (float64, uint, error) {

	t := time.Now()
	var orderID uint
	var deliveryID uint
	var deliveryObj models.Delivery
	var contacts models.Contacts

	deliveryObj = orderObj.DeliveryData
	var ApiPaymentObj models.ApiResponseDeliveryCost
	var rApiCost models.RequestDeliveryCost
	var fromLoc models.Location
	var toLoc models.Location
	var p models.Package
	var s models.Service
	var depositPrice float64

	if deliveryObj.Method == "door_to_door" {
		rApiCost.TariffCode = 139
	} else if deliveryObj.Method == "door_to_office" {
		rApiCost.TariffCode = 138
	}
	toLoc.Address = deliveryObj.Address
	toLoc.PostalCode = deliveryObj.PostalCode
	toLoc.Code = deliveryObj.Code
	rApiCost.ToLocation = toLoc
	p.Weight = (300 * len(orderObj.Projects))
	p.Height = 3 * len(orderObj.Projects)
	p.Width = 30 
	p.Length = 30 
	s.Code = "CARTON_BOX_500GR"
	s.Parameter = strconv.Itoa(len(orderObj.Projects))
	rApiCost.Packages = append(rApiCost.Packages, p)
	rApiCost.Services = append(rApiCost.Services, s)
	fromLoc.PostalCode = "129323"
	fromLoc.City = "Москва"
	fromLoc.Address = "проезд Серебрякова, 7"
	rApiCost.FromLocation = fromLoc

	

	ApiPaymentObj, err = delivery.CalculateDelivery(rApiCost)

	if err != nil {
		log.Printf("Error happened when ccalculating delivery. Err: %s", err)
			return depositPrice, orderID, err
	}

	contacts = orderObj.ContactData
	err = storeDB.QueryRow(ctx, "INSERT INTO delivery (status, created_at, method, address, amount, postal_code, code) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING delivery_id;",
		"DRAFT",
		t,
		deliveryObj.Method,
		deliveryObj.Address,
		ApiPaymentObj.TotalSum,
		deliveryObj.PostalCode,
		deliveryObj.Code).Scan(&deliveryID)
	if err != nil {
			log.Printf("Error happened when creating draft delivery entry into pgx table. Err: %s", err)
			return depositPrice, orderID, err
	}
	var requestP models.RequestPromooffer
	var responseP models.ResponsePromocodeUse
	var deposit float64
	requestP.Projects = orderObj.Projects
	requestP.Code = orderObj.Promocode
	
	responseP, err = userstorage.UsePromocode(ctx, storeDB, requestP)
	deposit, _, err = userstorage.UseCertificate(ctx, storeDB, orderObj.Giftcertificate, userID)
	if err != nil {
		log.Printf("Error happened when calculating discounted price. Err: %s", err)
		return depositPrice, orderID, err
	}

	var usedDeposit float64
	var GiftcertificatesID uint
	priceWithDelivery := responseP.DiscountedPrice + ApiPaymentObj.TotalSum
	if deposit != 0.0 {
		depositPrice = math.Max(0, priceWithDelivery - deposit)
		usedDeposit = priceWithDelivery - depositPrice
		log.Println(usedDeposit)
		log.Println(depositPrice)
		
		err = storeDB.QueryRow(ctx, "SELECT giftcertificates_id FROM giftcertificates WHERE code = ($1);", orderObj.Giftcertificate).Scan(&GiftcertificatesID)
		if err != nil {
			log.Printf("Failed to retrieve giftcertificates id. Err: %s", err)
			return depositPrice, orderID, err
		}
		_, err = storeDB.Exec(ctx, "UPDATE giftcertificates SET status = ($1) WHERE giftcertificates_id = ($2);",
		"RESERVED",
		GiftcertificatesID,
		)
		if err != nil {
				log.Printf("Error happened when updating gift certificate status into pgx table. Err: %s", err)
				return depositPrice, orderID, err
		}
	} else {
		depositPrice = priceWithDelivery
	}
	
	var PromoffersID uint
	err = storeDB.QueryRow(ctx, "SELECT promooffers_id FROM promooffers WHERE code = ($1);", orderObj.Promocode).Scan(&PromoffersID)
	if err != nil {
		log.Printf("Failed to retrieve promooffers id. Err: %s", err)
		return depositPrice, orderID, err
	}
	
	err := storeDB.QueryRow(ctx, "INSERT INTO orders (status, created_at, last_updated_at, users_id, firstname, lastname, email, phone, baseprice, finalprice, promooffers_id, giftcertificates_id, package_box, giftcertificates_deposit, delivery_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15) RETURNING orders_id;",
		"PAYMENTINPROGRESS",
		t,
		t,
		userID,
		contacts.FirstName,
		contacts.LastName,
		contacts.Email,
		contacts.Phone,
		responseP.BasePrice,
		depositPrice,
		PromoffersID,
		GiftcertificatesID, 
		orderObj.PackageBox, 
		usedDeposit,
		deliveryID).Scan(&orderID)
	if err != nil {
			log.Printf("Error happened when creating order entry into pgx table. Err: %s", err)
			return depositPrice, orderID, err
	}
	

	for _, project := range orderObj.Projects {
        _, err = storeDB.Exec(ctx, "INSERT INTO orders_has_projects (orders_id, projects_id) VALUES ($1, $2);",
		orderID,
		project,
		)
		if err != nil {
			log.Printf("Error happened when inserting orders_has_projects into pgx table. Err: %s", err)
			return depositPrice, orderID, err
		}
	}
		
	
	return depositPrice, orderID, err

}

// CancelPayment function performs the operation of cancelling payment for the order from pgx database with a query.
func CancelPayment(ctx context.Context, storeDB *pgxpool.Pool, orderID uint, userID uint) (error) {

	t := time.Now()
	var awaitedOrderID uint
	var deliveryID uint
	var transactionID uint
	
	// set order status to awaiting payment
	// set delivery status to cancelled
	// set transaction status to refunded

	_, err := storeDB.Exec(ctx, "UPDATE orders SET last_updated_at = ($1), status = ($2) WHERE orders_id = ($3);",
			t,
			"CANCELLED",
			orderID,
	)
	if err != nil {
		log.Printf("Error happened when cancelling order into pgx table. Err: %s", err)
		return err
	}
	
	err = storeDB.QueryRow(ctx, "SELECT orders_id FROM orders WHERE users_id = ($1) and status = ($2) ORDER BY created_at;", userID, "AWAITING PAYMENT").Scan(&awaitedOrderID)
	if err != nil {
		log.Printf("Error happened when searching for draft order into pgx table. Err: %s", err)
		return err
	}

	rows, err := storeDB.Query(ctx, "SELECT projects_id FROM orders_has_projects WHERE orders_id = ($1);", awaitedOrderID)
	if err != nil {
		log.Printf("Error happened when retrieving projects for cancelled order from pgx table. Err: %s", err)
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var projectID uint
		if err = rows.Scan(&projectID); err != nil {
			log.Printf("Error happened when scanning project ID. Err: %s", err)
			return err
		}
		_, err = storeDB.Exec(ctx, "INSERT INTO orders_has_projects (orders_id, projects_id) VALUES ($1, $2);",
			awaitedOrderID,
			projectID) 
		if err != nil {
				log.Printf("Error happened when rolling back project to draft order into pgx table. Err: %s", err)
				return err
		}
	}
	err = storeDB.QueryRow(ctx, "SELECT delivery_id FROM orders WHERE orders_id = ($1);", orderID).Scan(&deliveryID)
	if err != nil {
		log.Printf("Error happened when searching for delivery and transaction id for order into pgx table. Err: %s", err)
		return err
	}

	err = storeDB.QueryRow(ctx, "SELECT LAST(transaction_id) FROM orders_has_transactions WHERE orders_id = ($1);", orderID).Scan(&transactionID)
	if err = rows.Err(); err != nil {
		log.Printf("Error happened when retrieving transaction info from pgx table. Err: %s", err)
		return err
	}

	_, err = storeDB.Exec(ctx, "UPDATE transactions SET status = ($1) WHERE transactions_id = ($2);",
			"REFUNDED",
			transactionID,
	)
	if err != nil {
		log.Printf("Error happened when cancelling transaction into pgx table. Err: %s", err)
		return err
	}

	_, err = storeDB.Exec(ctx, "UPDATE delivery SET status = ($1) WHERE delivery_id = ($2);",
			"CANCELLED",
			deliveryID,
	)
	if err != nil {
		log.Printf("Error happened when cancelling delivery into pgx table. Err: %s", err)
		return err
	}
	var promocodeID uint
	var giftcertificateID uint
	var deposit float64
	var currentdeposit float64
	var promostatus bool
	err = storeDB.QueryRow(ctx, "SELECT promooffers_id, giftcertificates_id, giftcertificates_deposit FROM orders WHERE orders_id = ($1);", orderID).Scan(&promocodeID, &giftcertificateID, &deposit)
	if err != nil {
		log.Printf("Error happened when searching for promocode for order into pgx table. Err: %s", err)
		return err
	}
	if promocodeID != 0 {
		err = storeDB.QueryRow(ctx, "SELECT is_used FROM promooffers WHERE promooffers_id = ($1);", promocodeID).Scan(&promostatus)
		if err != nil {
			log.Printf("Error happened when searching for promocode for order into pgx table. Err: %s", err)
			return err
		}
		if promostatus == true {
			_, err = storeDB.Exec(ctx, "UPDATE promooffers SET is_used = ($1) WHERE promooffers_id = ($2);",
			false,
			promocodeID,
			)
			if err != nil {
				log.Printf("Error happened when restoring onetime promocode into pgx table. Err: %s", err)
				return err
			}
		}
	}

	if giftcertificateID != 0 && deposit != 0 {
		err = storeDB.QueryRow(ctx, "SELECT currentdeposit FROM giftcertificates WHERE giftcertificates_id = ($1);", giftcertificateID).Scan(&currentdeposit)
		if err != nil {
			log.Printf("Error happened when searching for gift certificate for order into pgx table. Err: %s", err)
			return err
		}
		currentdeposit = currentdeposit + deposit
		_, err = storeDB.Exec(ctx, "UPDATE giftcertificates SET currentdeposit = ($1) WHERE giftcertificates_id = ($2);",
		currentdeposit,
			giftcertificateID,
			)
			if err != nil {
				log.Printf("Error happened when restoring gift certificate deposit into pgx table. Err: %s", err)
				return err
			}
		
	}

	
	
	return nil

}

// RetrieveOrders function performs the operation of retrieving user orders from pgx database with a query.
func RetrieveOrders(ctx context.Context, storeDB *pgxpool.Pool, userID uint, isActive bool, offset uint, limit uint) (models.ResponseOrders, error) {

	orderSlice := []models.ResponseOrder{}
	var orderset models.ResponseOrders

	rows, err := storeDB.Query(ctx, "SELECT orders_id, status, created_at, baseprice, finalprice, delivery_id, promooffers_id, giftcertificates_id, giftcertificates_deposit FROM orders WHERE users_id = ($1) ORDER BY orders_id DESC LIMIT ($2) OFFSET ($3);", userID, limit, offset)
	if err != nil {
		log.Printf("Error happened when retrieving orders from pgx table. Err: %s", err)
		return orderset, err
	}
	defer rows.Close()
	var countActive int
	var countClosed int

	for rows.Next() {

		var orderObj models.ResponseOrder
		var oID uint
		var createTimeStorage time.Time
		var deliveryID *uint
		var promooffersID *uint
		var giftcertificateID *uint

		if err = rows.Scan(&oID, &orderObj.Status, &createTimeStorage, &orderObj.BasePrice, &orderObj.FinalPrice, &deliveryID, &promooffersID, &giftcertificateID, &orderObj.CertificateDeposit); err != nil {
			log.Printf("Error happened when scanning orders. Err: %s", err)
			return orderset, err
		}

		var deliveryAmount float64
		if orderObj.Status == "IN_DELIVERY" {
			err = storeDB.QueryRow(ctx, "SELECT trackingnumber, amount FROM delivery WHERE delivery_id = ($1);", deliveryID).Scan(&orderObj.TrackingNumber, &deliveryAmount)
			if err != nil && err != pgx.ErrNoRows {
				log.Printf("Error happened when retrieving project data from db. Err: %s", err)
				return orderset, err
			}
		}
		
		orderObj.OrderID = oID
		orderObj.CreatedAt = createTimeStorage.Unix()
		var certValue float64
		if orderObj.CertificateDeposit != nil {
			certValue = *orderObj.CertificateDeposit
		}
		var finalValue float64
		if orderObj.FinalPrice != nil {
			finalValue = *orderObj.FinalPrice
		}
		var baseValue float64
		if orderObj.BasePrice != nil {
			baseValue = *orderObj.BasePrice
		}
		orderObj.PromocodeDiscount = finalValue - baseValue - certValue + deliveryAmount
		err = storeDB.QueryRow(ctx, "SELECT discount, category FROM promooffers WHERE promooffers_id = ($1);", promooffersID).Scan(&orderObj.PromocodeDiscountPercent, &orderObj.PromocodeCategory)
		if err != nil && err != pgx.ErrNoRows {
			log.Printf("Error happened when retrieving promocode data from db. Err: %s", err)
			return orderset, err
		}
		prows, err := storeDB.Query(ctx, "SELECT projects_id FROM orders_has_projects WHERE orders_id = ($1);", oID)
		if err != nil {
			log.Printf("Error happened when retrieving order projects from pgx table. Err: %s", err)
			return orderset, err
		}

		for prows.Next() {
			var photobook models.PaidCartObj
			var pID uint
			var leatherID *uint
			if err = prows.Scan(&pID); err != nil {
				log.Printf("Error happened when scanning projects. Err: %s", err)
				return orderset, err
			}

			err := storeDB.QueryRow(ctx, "SELECT name, size, variant, paper, cover, count_pages, leather_id FROM projects WHERE projects_id = ($1);", pID).Scan(&photobook.Name, &photobook.Size, &photobook.Variant, &photobook.Surface, &photobook.Cover, &photobook.CountPages, &leatherID)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				log.Printf("Error happened when retrieving project info from pgx table. Err: %s", err)
				return orderset, err
			}
			if photobook.Cover == "LEATHERETTE" && leatherID != nil {
				err := storeDB.QueryRow(ctx, "SELECT colourlink FROM leather WHERE leather_id = ($1);", leatherID).Scan(&photobook.LeatherImage)
				if err != nil && !errors.Is(err, pgx.ErrNoRows) {
					log.Printf("Error happened when retrieving leather cover info from pgx table. Err: %s", err)
					return orderset, err
				}
			}
			photobook.BasePrice, err = CalculateBasePrice(ctx, storeDB, photobook.Size, photobook.Variant, photobook.Cover, photobook.Surface, uint(photobook.CountPages))
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				log.Printf("Error happened when counting baseprice. Err: %s", err)
				return orderset, err
			}
			photobook.FrontPage, err = projectstorage.RetrieveFrontPage(ctx, storeDB, pID, false) 
			photobook.ProjectID = pID

			orderObj.Projects = append(orderObj.Projects, photobook)
		}
		defer prows.Close()
		log.Println(orderObj)
		if isActive == true && orderObj.Status != "COMPLETED" && orderObj.Status != "CANCELLED" {
			orderSlice = append(orderSlice, orderObj)
			countActive = countActive + 1
		} else if orderObj.Status == "COMPLETED" && orderObj.Status == "CANCELLED" {
			orderSlice = append(orderSlice, orderObj)
			countClosed= countClosed + 1
		}
		
		
	defer rows.Close()
		
		
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error happened when retrieving orders from pgx table. Err: %s", err)
		return orderset, err
	}
	orderset.Orders = orderSlice
	if isActive == true {
		orderset.CountAll = countActive
	} else {
		orderset.CountAll = countClosed
	}
		
	return orderset, nil

}

// RetrieveSingleOrder function performs the operation of retrieving single order from pgx database with a query.
func RetrieveSingleOrder(ctx context.Context, storeDB *pgxpool.Pool, orderID uint) (models.ResponseOrder, error) {

	var orderObj models.ResponseOrder
	var createTimeStorage time.Time
	var deliveryID uint
	err := storeDB.QueryRow(ctx, "SELECT orders_id, status, created_at, baseprice, finalprice, delivery_id FROM orders WHERE orders_id = ($1);", orderID).Scan(&orderObj.OrderID, &orderObj.Status, &createTimeStorage, &orderObj.BasePrice, &orderObj.FinalPrice, &deliveryID)
		
	if err != nil {
			log.Printf("Error happened when scanning orders. Err: %s", err)
			return orderObj, err
	}

		
	orderObj.CreatedAt = createTimeStorage.Unix()
	prows, err := storeDB.Query(ctx, "SELECT projects_id FROM orders_has_projects WHERE orders_id = ($1) ORDER DESC;", orderID)
	if err != nil {
			log.Printf("Error happened when retrieving order projects from pgx table. Err: %s", err)
			return orderObj, err
	}

	for prows.Next() {
			var photobook models.PaidCartObj
			var pID uint
			var leatherID uint
			if err = prows.Scan(&pID); err != nil {
				log.Printf("Error happened when scanning projects. Err: %s", err)
				return orderObj, err
			}

			err := storeDB.QueryRow(ctx, "SELECT name, size, variant, paper, cover, count_pages, leather_id FROM projects WHERE projects_id = ($1);", pID).Scan(&photobook.Name, &photobook.Size, &photobook.Variant, &photobook.Surface, &photobook.Cover, &photobook.CountPages, &leatherID)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				log.Printf("Error happened when retrieving project info from pgx table. Err: %s", err)
				return orderObj, err
			}
			if photobook.Cover == "LEATHERETTE" && leatherID != 0 {
				err := storeDB.QueryRow(ctx, "SELECT colourlink FROM leather WHERE leather_id = ($1);", leatherID).Scan(&photobook.LeatherImage)
				if err != nil && !errors.Is(err, pgx.ErrNoRows) {
					log.Printf("Error happened when retrieving leather cover info from pgx table. Err: %s", err)
					return orderObj, err
				}
			}
			photobook.BasePrice, err = CalculateBasePrice(ctx, storeDB, photobook.Size, photobook.Variant, photobook.Cover, photobook.Surface, uint(photobook.CountPages))
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				log.Printf("Error happened when counting baseprice. Err: %s", err)
				return orderObj, err
			}
			photobook.FrontPage, err = projectstorage.RetrieveFrontPage(ctx, storeDB, pID, false) 
			
			orderObj.Projects = append(orderObj.Projects, photobook)
	}
	defer prows.Close()

	return orderObj, nil

}

		
// RetrieveAdminOrders function performs the operation of retrieving orders for admin from pgx database with a query.
func RetrieveAdminOrders(ctx context.Context, storeDB *pgxpool.Pool, userID uint, orderID uint, isActive bool, createdAfter uint, createdBefore uint, email string, status string, offset uint, limit uint) (models.ResponseAdminOrders, error) {

	orderSlice := []models.ResponseAdminOrder{}
	orderset := models.ResponseAdminOrders{}
	var err error
	var rows pgx.Rows

	if email != "" {
		userID, err = userstorage.GetUserID(ctx, storeDB, email) 
		if err != nil {
			log.Printf("Error happened when retrieving user id by email from pgx table. Err: %s", err)
			return orderset, err
		}
	}
	rows, err = storeDB.Query(ctx, "SELECT orders_id, users_id, commentary, status, created_at, baseprice, finalprice, videolink, delivery_id, promooffers_id, giftcertificates_id, giftcertificates_deposit FROM orders ORDER BY orders_id DESC LIMIT ($1) OFFSET ($2);", limit, offset)
	if err != nil {
		log.Printf("Error happened when retrieving orders from pgx table. Err: %s", err)
		return orderset, err
	}
	defer rows.Close()
	if orderID != 0 && userID != 0 && status != "" {
		rows, err := storeDB.Query(ctx, "SELECT orders_id, users_id, commentary, status, created_at, baseprice, finalprice, videolink, delivery_id, promooffers_id, giftcertificates_id, giftcertificates_deposit FROM orders WHERE orders_id = ($1) AND users_id = ($2) AND status = ($3) ORDER BY orders_id DESC LIMIT ($4) OFFSET ($5);", orderID, userID, status, limit, offset)
		if err != nil {
			log.Printf("Error happened when retrieving orders from pgx table. Err: %s", err)
			return orderset, err
		}
		defer rows.Close()
	} else if orderID != 0 && userID != 0 {
		rows, err := storeDB.Query(ctx, "SELECT orders_id, users_id, commentary, status, created_at, baseprice, finalprice, videolink, delivery_id, promooffers_id, giftcertificates_id, giftcertificates_deposit FROM orders WHERE orders_id = ($1) AND users_id = ($2) ORDER BY orders_id DESC LIMIT ($3) OFFSET ($4);", orderID, userID, limit, offset)
		if err != nil {
			log.Printf("Error happened when retrieving orders from pgx table. Err: %s", err)
			return orderset, err
		}
		defer rows.Close()
	} else if orderID != 0 && status != "" {
			rows, err := storeDB.Query(ctx, "SELECT orders_id, users_id, commentary, status, created_at, baseprice, finalprice, videolink, delivery_id, promooffers_id, giftcertificates_id, giftcertificates_deposit FROM orders WHERE orders_id = ($1) AND status = ($2) ORDER BY orders_id DESC LIMIT ($3) OFFSET ($4);", orderID, status, limit, offset)
			if err != nil {
				log.Printf("Error happened when retrieving orders by status from pgx table. Err: %s", err)
				return orderset, err
			}
			defer rows.Close()
	} else if orderID != 0 {
				rows, err := storeDB.Query(ctx, "SELECT orders_id, users_id, commentary, status, created_at, baseprice, finalprice, videolink, delivery_id, promooffers_id, giftcertificates_id, giftcertificates_deposit FROM orders WHERE orders_id = ($1) ORDER BY orders_id DESC LIMIT ($2) OFFSET ($3);", orderID, limit, offset)
				if err != nil {
					log.Printf("Error happened when retrieving orders from pgx table. Err: %s", err)
					return orderset, err
				}
				defer rows.Close()
			
	} else if userID != 0 && status != "" {
			rows, err := storeDB.Query(ctx, "SELECT orders_id, users_id, commentary, status, created_at, baseprice, finalprice, videolink, delivery_id, promooffers_id, giftcertificates_id, giftcertificates_deposit FROM orders WHERE status = ($1) AND users_id = ($2) ORDER BY orders_id DESC LIMIT ($3) OFFSET ($4);", status, userID, limit, offset)
			if err != nil {
				log.Printf("Error happened when retrieving orders from pgx table. Err: %s", err)
				return orderset, err
			}
			defer rows.Close()
	} else if userID != 0 {
		rows, err := storeDB.Query(ctx, "SELECT orders_id, users_id, commentary, status, created_at, baseprice, finalprice, videolink, delivery_id, promooffers_id, giftcertificates_id, giftcertificates_deposit FROM orders WHERE users_id = ($1) ORDER BY orders_id DESC LIMIT ($2) OFFSET ($3);", userID, limit, offset)
		if err != nil {
			log.Printf("Error happened when retrieving orders from pgx table. Err: %s", err)
			return orderset, err
		}
		defer rows.Close()
	} else if status != "" {
		rows, err := storeDB.Query(ctx, "SELECT orders_id, users_id, commentary, status, created_at, baseprice, finalprice, videolink, delivery_id, promooffers_id, giftcertificates_id, giftcertificates_deposit FROM orders WHERE status = ($1) ORDER BY orders_id DESC LIMIT ($2) OFFSET ($3);", status, limit, offset)
		if err != nil {
			log.Printf("Error happened when retrieving orders from pgx table. Err: %s", err)
			return orderset, err
		}
		defer rows.Close()
	}

	for rows.Next() {

		var orderObj models.ResponseAdminOrder
		var oID uint
		var createTimeStorage time.Time
		var deliveryID *uint
		var promooffersID *uint
		var giftcertificateID *uint

		if err = rows.Scan(&oID, &orderObj.UserID, &orderObj.Commentary, &orderObj.Status, &createTimeStorage, &orderObj.BasePrice, &orderObj.FinalPrice, &orderObj.VideoLink, &deliveryID, &promooffersID, &giftcertificateID, &orderObj.CertificateDeposit); err != nil {
			log.Printf("Error happened when scanning orders. Err: %s", err)
			return orderset, err
		}
		err = storeDB.QueryRow(ctx, "SELECT trackingnumber FROM delivery WHERE delivery_id = ($1);", deliveryID).Scan(&orderObj.TrackingNumber)
		if err != nil && err != pgx.ErrNoRows {
			log.Printf("Error happened when retrieving project data from db. Err: %s", err)
			return orderset, err
		}
		var user models.UserInfo
		user, err = userstorage.GetUserData(ctx, storeDB, orderObj.UserID)
		if err != nil {
			log.Printf("Error happened when retrieving user data from pgx table. Err: %s", err)
			return orderset, err
		}
		orderObj.Email = user.Email
		orderObj.OrderID = oID
		var deliveryAmount float64
		if orderObj.Status == "IN_DELIVERY" {
			err = storeDB.QueryRow(ctx, "SELECT trackingnumber, amount FROM delivery WHERE delivery_id = ($1);", deliveryID).Scan(&orderObj.TrackingNumber, &deliveryAmount)
			if err != nil && err != pgx.ErrNoRows {
				log.Printf("Error happened when retrieving project data from db. Err: %s", err)
				return orderset, err
			}
		}
		
		var certValue float64
		if orderObj.CertificateDeposit != nil {
			certValue = *orderObj.CertificateDeposit
		}
		var finalValue float64
		if orderObj.FinalPrice != nil {
			finalValue = *orderObj.FinalPrice
		}
		var baseValue float64
		if orderObj.BasePrice != nil {
			baseValue = *orderObj.BasePrice
		}
		orderObj.PromocodeDiscount = finalValue - baseValue - certValue + deliveryAmount
		err = storeDB.QueryRow(ctx, "SELECT discount, category FROM promooffers WHERE promooffers_id = ($1);", promooffersID).Scan(&orderObj.PromocodeDiscountPercent, &orderObj.PromocodeCategory)
		if err != nil && err != pgx.ErrNoRows {
			log.Printf("Error happened when retrieving promocode data from db. Err: %s", err)
			return orderset, err
		}
		prows, err := storeDB.Query(ctx, "SELECT projects_id FROM orders_has_projects WHERE orders_id = ($1);", oID)
		if err != nil {
			log.Printf("Error happened when retrieving order projects from pgx table. Err: %s", err)
			return orderset, err
		}


		for prows.Next() {
			var photobook models.PaidCartObj
			var pID uint
			var leatherID *uint
			if err = prows.Scan(&pID); err != nil {
				log.Printf("Error happened when scanning projects. Err: %s", err)
				return orderset, err
			}

			err := storeDB.QueryRow(ctx, "SELECT name, size, variant, paper, cover, count_pages, leather_id FROM projects WHERE projects_id = ($1);", pID).Scan(&photobook.Name, &photobook.Size, &photobook.Variant, &photobook.Surface, &photobook.Cover, &photobook.CountPages, &leatherID)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				log.Printf("Error happened when retrieving project info from pgx table. Err: %s", err)
				return orderset, err
			}
			if photobook.Cover == "LEATHERETTE" && leatherID != nil {
				err := storeDB.QueryRow(ctx, "SELECT colourlink FROM leather WHERE leather_id = ($1);", leatherID).Scan(&photobook.LeatherImage)
				if err != nil && !errors.Is(err, pgx.ErrNoRows) {
					log.Printf("Error happened when retrieving leather cover info from pgx table. Err: %s", err)
					return orderset, err
				}
			}
			photobook.BasePrice, err = CalculateBasePrice(ctx, storeDB, photobook.Size, photobook.Variant, photobook.Cover, photobook.Surface, uint(photobook.CountPages))
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				log.Printf("Error happened when counting baseprice. Err: %s", err)
				return orderset, err
			}
			photobook.FrontPage, err = projectstorage.RetrieveFrontPage(ctx, storeDB, pID, false) 
			photobook.ProjectID = pID

			orderObj.Projects = append(orderObj.Projects, photobook)
		defer prows.Close()
		if createdAfter != 0 {
			if createdBefore != 0 {
				if createdAfter <= uint(orderObj.CreatedAt) && createdBefore >= uint(orderObj.CreatedAt) {
					if isActive == true && orderObj.Status != "COMPLETED" && orderObj.Status != "CANCELLED" {
						orderSlice = append(orderSlice, orderObj)
					} else  if orderObj.Status == "COMPLETED" && orderObj.Status == "CANCELLED" {
						orderSlice = append(orderSlice, orderObj)
					}
				}
			} else {
				if createdAfter <= uint(orderObj.CreatedAt) {
					if isActive == true && orderObj.Status != "COMPLETED" && orderObj.Status != "CANCELLED" {
						orderSlice = append(orderSlice, orderObj)
					} else  if orderObj.Status == "COMPLETED" && orderObj.Status == "CANCELLED" {
						orderSlice = append(orderSlice, orderObj)
					}
				}
			}
		} else if createdBefore != 0 {
			if createdBefore >= uint(orderObj.CreatedAt) {
				if isActive == true && orderObj.Status != "COMPLETED" && orderObj.Status != "CANCELLED" {
					orderSlice = append(orderSlice, orderObj)
				} else  if orderObj.Status == "COMPLETED" && orderObj.Status == "CANCELLED" {
					orderSlice = append(orderSlice, orderObj)
				}
			}
		} else {
			if isActive == true && orderObj.Status != "COMPLETED" && orderObj.Status != "CANCELLED" {
				orderSlice = append(orderSlice, orderObj)
			} else  if orderObj.Status == "COMPLETED" && orderObj.Status == "CANCELLED" {
				orderSlice = append(orderSlice, orderObj)
			}
			
		}
		
		defer rows.Close()
		
		
	}

	if err != pgx.ErrNoRows && err != nil {
		log.Printf("Error happened when retrieving orders from pgx table. Err: %s", err)
		return orderset, err
	}
	orderset.Orders = orderSlice
	var countAllString string
	err = storeDB.QueryRow(ctx, "SELECT COUNT(orders_id) FROM orders;").Scan(&countAllString)
	if err != nil && err != pgx.ErrNoRows{
			log.Printf("Error happened when counting orders in pgx table. Err: %s", err)
			return orderset, err
	}

	if orderID != 0 && userID != 0 && status != "" {
		err = storeDB.QueryRow(ctx, "SELECT COUNT(orders_id) FROM orders WHERE orders_id = ($1) AND users_id = ($2) AND status = ($3);", orderID, userID, status).Scan(&countAllString)
		if err != nil && err != pgx.ErrNoRows{
			log.Printf("Error happened when counting orders in pgx table. Err: %s", err)
			return orderset, err
		}
	} else if orderID != 0 && userID != 0 {
		err = storeDB.QueryRow(ctx, "SELECT COUNT(orders_id) FROM orders WHERE orders_id = ($1) AND users_id = ($2);", orderID, userID).Scan(&countAllString)
		if err != nil && err != pgx.ErrNoRows{
			log.Printf("Error happened when counting orders in pgx table. Err: %s", err)
			return orderset, err
		}
	} else if orderID != 0 && status != "" {
			err = storeDB.QueryRow(ctx, "SELECT COUNT(orders_id) FROM orders WHERE orders_id = ($1) AND status = ($2);", orderID, status).Scan(&countAllString)
			if err != nil && err != pgx.ErrNoRows{
				log.Printf("Error happened when counting orders in pgx table. Err: %s", err)
				return orderset, err
			}
	} else if orderID != 0 {
				err = storeDB.QueryRow(ctx, "SELECT COUNT(orders_id) FROM orders WHERE orders_id = ($1);", orderID).Scan(&countAllString)
				if err != nil && err != pgx.ErrNoRows{
					log.Printf("Error happened when counting orders in pgx table. Err: %s", err)
					return orderset, err
				}
			
	} else if userID != 0 && status != "" {
			err = storeDB.QueryRow(ctx, "SELECT COUNT(orders_id) FROM orders WHERE status = ($1) AND users_id = ($2);", status, userID).Scan(&countAllString)
			if err != nil && err != pgx.ErrNoRows{
				log.Printf("Error happened when counting orders in pgx table. Err: %s", err)
				return orderset, err
			}
	} else if userID != 0 {
		err = storeDB.QueryRow(ctx, "SELECT COUNT(orders_id) FROM orders WHERE users_id = ($1);", userID).Scan(&countAllString)
		if err != nil && err != pgx.ErrNoRows{
			log.Printf("Error happened when counting orders in pgx table. Err: %s", err)
			return orderset, err
		}
	} else if status != "" {
		err = storeDB.QueryRow(ctx, "SELECT COUNT(orders_id) FROM orders WHERE status = ($1);", status).Scan(&countAllString)
		if err != nil && err != pgx.ErrNoRows{
			log.Printf("Error happened when counting orders in pgx table. Err: %s", err)
			return orderset, err
		}
	}
		
	orderset.CountAll, _ = strconv.Atoi(countAllString)
	
	}
	return orderset, nil
}

// LoadOrder function performs the operation of retrieving order info from pgx database with a query.
func LoadOrder(ctx context.Context, storeDB *pgxpool.Pool, orderID uint) (models.ResponseOrderInfo, error) {

	orderObj := models.ResponseOrderInfo{}
	var contactData models.Contacts
	var deliveryData models.Delivery
	var deliveryID uint
	var promocodeID uint

	err := storeDB.QueryRow(ctx, "SELECT status, users_id, delivery_id, firstname, lastname, email, phone, giftcertificates_deposit, promooffers_id FROM orders WHERE orders_id = ($1);", orderID).Scan(&orderObj.Status, &orderObj.UserID, &deliveryID, &contactData.FirstName, &contactData.LastName, &contactData.Email, &contactData.Phone, &orderObj.GiftcertificateDeposit, &promocodeID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				log.Printf("Error happened when retrieving order info from pgx table. Err: %s", err)
				return orderObj, err
	}
	err = storeDB.QueryRow(ctx, "SELECT transactions_id FROM orders_has_transactions WHERE orders_id = ($1) ORDER BY transactions_id;", orderID).Scan(&orderObj.TransactionID)
	if err != nil {
		log.Printf("Error happened when retrieving transaction info from pgx table. Err: %s", err)
		return orderObj, err
	}
	err = storeDB.QueryRow(ctx, "SELECT method, address, code, postal_code, amount FROM delivery WHERE delivery_id = ($1);", deliveryID).Scan(&deliveryData.Method, &deliveryData.Address, &deliveryData.Code, &deliveryData.PostalCode, &deliveryData.Amount)
	if err != nil {
		log.Printf("Error happened when retrieving delivery info from pgx table. Err: %s", err)
		return orderObj, err
	}
	err = storeDB.QueryRow(ctx, "SELECT code, discount FROM promooffers WHERE promooffers_id = ($1);", promocodeID).Scan(&orderObj.Promocode, &orderObj.PromocodeDiscountPercent)
	if err != nil {
		log.Printf("Error happened when retrieving promooffer info from pgx table. Err: %s", err)
		return orderObj, err
	}
	orderObj.ContactData = contactData
	orderObj.DeliveryData = deliveryData
	prows, err := storeDB.Query(ctx, "SELECT projects_id FROM orders_has_projects WHERE orders_id = ($1);", orderID)
	if err != nil {
		log.Printf("Error happened when retrieving order projects from pgx table. Err: %s", err)
		return orderObj, err
	}


	for prows.Next() {
		var pID uint
		var previewObj models.PreviewObject
		if err = prows.Scan(&pID); err != nil {
			log.Printf("Error happened when scanning projects. Err: %s", err)
			return orderObj, err
		}

		err := storeDB.QueryRow(ctx, "SELECT name, preview_link FROM projects WHERE projects_id = ($1);", pID).Scan(&previewObj.Name, &previewObj.Link)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			log.Printf("Error happened when retrieving project preview link from pgx table. Err: %s", err)
			return orderObj, err
		}
		orderObj.PreviewLinks = append(orderObj.PreviewLinks, previewObj)
	}
	return orderObj, nil

}

// AdminLoadOrder function performs the operation of retrieving order info from pgx database with a query.
func AdminLoadOrder(ctx context.Context, storeDB *pgxpool.Pool, orderID uint) (models.ResponseOrderInfo, error) {

	orderObj := models.ResponseOrderInfo{}
	var contactData models.Contacts
	var deliveryData models.Delivery
	var deliveryID uint
	var promoofferID uint

	err := storeDB.QueryRow(ctx, "SELECT users_id, delivery_id, firstname, lastname, email, phone, giftcertificates_deposit, promooffers_id FROM orders WHERE orders_id = ($1);", orderID).Scan(&orderObj.UserID, &deliveryID, &contactData.FirstName, &contactData.LastName, &contactData.Email, &contactData.Phone, &orderObj.GiftcertificateDeposit, &promoofferID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				log.Printf("Error happened when retrieving order info from pgx table. Err: %s", err)
				return orderObj, err
	}

	err = storeDB.QueryRow(ctx, "SELECT transactions_id FROM orders_has_transactions WHERE orders_id = ($1) ORDER BY transactions_id;", orderID).Scan(&orderObj.TransactionID)
	if err != nil {
		log.Printf("Error happened when retrieving transaction info from pgx table. Err: %s", err)
		return orderObj, err
	}

	err = storeDB.QueryRow(ctx, "SELECT method, address, amount, postal_code, code FROM delivery WHERE delivery_id = ($1);", deliveryID).Scan(&deliveryData.Method, &deliveryData.Address, &deliveryData.Amount, &deliveryData.PostalCode, &deliveryData.Code)
	if err != nil {
		log.Printf("Error happened when retrieving delivery info from pgx table. Err: %s", err)
		return orderObj, err
	}
	err = storeDB.QueryRow(ctx, "SELECT discount FROM promooffers WHERE promooffers_id = ($1);", promoofferID).Scan(&orderObj.PromocodeDiscountPercent)
	if err != nil {
		log.Printf("Error happened when retrieving promooffer info from pgx table. Err: %s", err)
		return orderObj, err
	}
	orderObj.ContactData = contactData
	orderObj.DeliveryData = deliveryData
	prows, err := storeDB.Query(ctx, "SELECT projects_id FROM orders_has_projects WHERE orders_id = ($1);", orderID)
	if err != nil {
		log.Printf("Error happened when retrieving order projects from pgx table. Err: %s", err)
		return orderObj, err
	}


	for prows.Next() {
		var pID uint
		var previewObj models.PreviewObject
		if err = prows.Scan(&pID); err != nil {
			log.Printf("Error happened when scanning projects. Err: %s", err)
			return orderObj, err
		}

		err := storeDB.QueryRow(ctx, "SELECT name, preview_link FROM projects WHERE projects_id = ($1);", pID).Scan(&previewObj.Name, &previewObj.Link)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			log.Printf("Error happened when retrieving project preview link from pgx table. Err: %s", err)
			return orderObj, err
		}
		orderObj.PreviewLinks = append(orderObj.PreviewLinks, previewObj)
	}
	return orderObj, nil

}

// LoadDelivery function performs the operation of retrieving delivery info from pgx database with a query.
func LoadDelivery(ctx context.Context, storeDB *pgxpool.Pool, orderID uint) (models.ResponseDeliveryInfo, error) {

	orderObj := models.ResponseDeliveryInfo{}
	var contactData models.Contacts
	var deliveryID uint
	var deliveryTimeFrom *time.Time
	var deliveryTimeTo *time.Time
	var address *string
	var trackingnumber *string
	var code *string

	err := storeDB.QueryRow(ctx, "SELECT users_id, delivery_id, firstname, lastname, email, phone FROM orders WHERE orders_id = ($1);", orderID).Scan(&orderObj.UserID, &deliveryID, &contactData.FirstName, &contactData.LastName, &contactData.Email, &contactData.Phone)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				log.Printf("Error happened when retrieving delivery id info from pgx table. Err: %s", err)
				return orderObj, err
	}
	orderObj.ContactData = contactData
	err = storeDB.QueryRow(ctx, "SELECT method, address, code, deliverystatus, deliveryid, trackingnumber, expected_delivery_from, expected_delivery_to FROM delivery WHERE delivery_id = ($1);", deliveryID).Scan(&orderObj.Method, &address, &code, &orderObj.DeliveryStatus, &orderObj.DeliveryID, &trackingnumber, &deliveryTimeFrom, &deliveryTimeTo)
	if err != nil {
		log.Printf("Error happened when retrieving delivery info from pgx table. Err: %s", err)
		return orderObj, err
	}
	if code != nil {
		orderObj.Code = code
	} else {
		orderObj.Address = address
	}
	if trackingnumber != nil {
	   orderObj.TrackingNumber = trackingnumber
	}
	if deliveryTimeFrom != nil {
		orderObj.ExpectedDeliveryFrom = deliveryTimeFrom.Unix()
	}
	if deliveryTimeTo != nil {
		orderObj.ExpectedDeliveryTo = deliveryTimeTo.Unix()
	}
	
	
	return orderObj, nil

}



// UpdateOrderStatus function performs the operation of updating the order status from pgx database with a query.
func UpdateOrderStatus(ctx context.Context, storeDB *pgxpool.Pool, orderID uint, statusObj models.RequestUpdateOrderStatus) (error) {


	_, err := storeDB.Exec(ctx, "UPDATE orders SET status = ($1) WHERE orders_id = ($2);",
			statusObj.Status,
			orderID,
	)
	if err != nil {
		log.Printf("Error happened when updating order status into pgx table. Err: %s", err)
		return err
	}

	return nil

}

// UpdateOrderCommentary function performs the operation of updating the order commentary from pgx database with a query.
func UpdateOrderCommentary(ctx context.Context, storeDB *pgxpool.Pool, orderID uint, commentaryObj models.RequestUpdateOrderCommentary) (error) {


	_, err := storeDB.Exec(ctx, "UPDATE orders SET commentary = ($1) WHERE orders_id = ($2);",
			commentaryObj.Commentary,
			orderID,
	)
	if err != nil {
		log.Printf("Error happened when updating order commentary into pgx table. Err: %s", err)
		return err
	}

	return nil

}

// UploadOrderVideo function performs the operation of uploading video for the order from pgx database with a query.
func UploadOrderVideo(ctx context.Context, storeDB *pgxpool.Pool, orderID uint, videoObj models.OrderVideo) (error) {


	_, err := storeDB.Exec(ctx, "UPDATE orders SET videolink = ($1) WHERE orders_id = ($2);",
			videoObj.VideoLink,
			orderID,
	)
	if err != nil {
		log.Printf("Error happened when updating order video into pgx table. Err: %s", err)
		return err
	}

	return nil

}

// DownloadOrderVideo function performs the operation of downloading video for the order from pgx database with a query.
func DownloadOrderVideo(ctx context.Context, storeDB *pgxpool.Pool, orderID uint) (models.OrderVideo, error) {

	var videoObj models.OrderVideo
	err := storeDB.QueryRow(ctx, "SELECT video FROM orders WHERE orders_id = ($1);", orderID).Scan(&videoObj.VideoLink)
	if err != nil {
				log.Printf("Error happened when searching for order video from pgx table. Err: %s", err)
				return videoObj, err
	}
	if err != nil {
		log.Printf("Error happened when downloading order video from pgx table. Err: %s", err)
		return videoObj, err
	}

	return videoObj, nil

}



// UpdateTransaction function performs the operation of creatign the new transaction row for the  order in pgx database with a query.
func UpdateTransaction(ctx context.Context, storeDB *pgxpool.Pool, orderID uint, transaction models.ResponseTransaction, finalPrice float64, goodType string) (error) {

	t := time.Now()
	var tID uint
	err = storeDB.QueryRow(ctx, "INSERT INTO transactions (status, created_at, amount, bankorderid, paymentmethod) VALUES ($1, $2, $3, $4, $5) RETURNING transactions_id ;",
		"INPROGRESS",
		t,
		finalPrice,
		transaction.OrderID,
		"BANK CARD").Scan(&tID)
	if err != nil {
			log.Printf("Error happened when creating transaction entry into pgx table. Err: %s", err)
			return err
	}
	if goodType == "PHOTOBOOK" {
		_, err = storeDB.Exec(ctx, "INSERT INTO orders_has_transactions (orders_id, transactions_id) VALUES ($1, $2);",
			orderID,
			tID) 
		if err != nil {
				log.Printf("Error happened when adding new orders_has_transactions to order into pgx table. Err: %s", err)
				return err
		}
	} else if goodType == "CERTIFICATE" {
		_, err = storeDB.Exec(ctx, "INSERT INTO giftcertificates_has_transactions (giftcertificates_id, transactions_id) VALUES ($1, $2);",
			orderID,
			tID) 
		if err != nil {
				log.Printf("Error happened when adding new giftcertificates_has_transactions to order into pgx table. Err: %s", err)
				return err
		}
	}

	return nil

}

// UpdateSuccessfulTransaction function performs the operation of updating transaction and order in pgx database with a query.
func UpdateSuccessfulTransaction(ctx context.Context, storeDB *pgxpool.Pool, orderID uint) (error) {

	t := time.Now()
	var tID uint
	err := storeDB.QueryRow(ctx, "SELECT LAST(transactions_id) FROM orders_has_transactions WHERE orders_id = ($1);", orderID).Scan(&tID)
	if err != nil {
		log.Printf("Error happened when retrieving transaction info from pgx table. Err: %s", err)
		return err
	}
	_, err = storeDB.Exec(ctx, "UPDATE transactions SET status = ($1) WHERE transactions_id = ($2);",
			"SUCCESSFUL",
			tID,
	)
	if err != nil {
		log.Printf("Error happened when updating transaction status into pgx table. Err: %s", err)
		return err
	}
	_, err = storeDB.Exec(ctx, "UPDATE orders SET status = ($1), last_updated_at = ($2) WHERE orders_id = ($3);",
			"PAID",
			t,
			orderID,
	)
	if err != nil {
		log.Printf("Error happened when updating order status into pgx table. Err: %s", err)
		return err
	}
	// promocodes
	// giftcertificate
	var promocodeID uint
	var giftcertificateID uint
	var deposit float64
	var currentdeposit float64
	var oneTime bool
	err = storeDB.QueryRow(ctx, "SELECT promocodes_id, giftcertificates_id, giftcertificates_deposit FROM orders WHERE orders_id = ($1);", orderID).Scan(&promocodeID, &giftcertificateID, &deposit)
	if err != nil {
		log.Printf("Error happened when searching for promocode for order into pgx table. Err: %s", err)
		return err
	}
	if promocodeID != 0 {
		err = storeDB.QueryRow(ctx, "SELECT is_onetime FROM promocodes WHERE promocodes_id = ($1);", promocodeID).Scan(&oneTime)
		if err != nil {
			log.Printf("Error happened when searching for promocode for order into pgx table. Err: %s", err)
			return err
		}
		
		if oneTime == true {
			_, err = storeDB.Exec(ctx, "UPDATE promocodes SET status = ($1), is_used = ($2) WHERE promocodes_id = ($3);",
			"ONETIME_USED",
			true,
			promocodeID,
			)
			if err != nil {
				log.Printf("Error happened when updating onetime promocode into pgx table. Err: %s", err)
				return err
			}
		}
	}

	if giftcertificateID != 0 && deposit != 0 {
		err = storeDB.QueryRow(ctx, "SELECT currentdeposit FROM giftcertificates WHERE giftcertificates_id = ($1);", promocodeID).Scan(&currentdeposit)
		if err != nil {
			log.Printf("Error happened when searching for gift certificate for order into pgx table. Err: %s", err)
			return err
		}
		currentdeposit = currentdeposit - deposit
		_, err = storeDB.Exec(ctx, "UPDATE giftcertificates SET currentdeposit = ($1) WHERE giftcertificates_id = ($2);",
		    currentdeposit,
			giftcertificateID,
			)
			if err != nil {
				log.Printf("Error happened when using gift certificate deposit into pgx table. Err: %s", err)
				return err
			}
		
	}

	
	return nil

}

// UpdateUnSuccessfulTransaction function performs the operation of updating transaction and order in pgx database with a query.
func UpdateUnSuccessfulTransaction(ctx context.Context, storeDB *pgxpool.Pool, orderID uint) (error) {

	var tID uint
	err := storeDB.QueryRow(ctx, "SELECT LAST(transactions_id) FROM orders_has_transactions WHERE orders_id = ($1);", orderID).Scan(&tID)
	if err != nil {
		log.Printf("Error happened when retrieving transaction info from pgx table. Err: %s", err)
		return err
	}
	_, err = storeDB.Exec(ctx, "UPDATE transactions SET status = ($1) WHERE transactions_id = ($2);",
			"UNSUCCESSFUL",
			tID,
	)
	if err != nil {
		log.Printf("Error happened when updating transaction status into pgx table. Err: %s", err)
		return err
	}
	
	return nil

}

// GetBankTransactionID function performs the operation of retrieving transaction id for order in pgx database with a query.
func GetBankTransactionID(ctx context.Context, storeDB *pgxpool.Pool, orderID uint) (string, error) {

	var tID uint
	var bankID string 
	err := storeDB.QueryRow(ctx, "SELECT transactions_id FROM orders_has_transactions WHERE orders_id = ($1) ORDER BY transactions_id DESC LIMIT 1;", orderID).Scan(&tID)
	if err != nil {
		log.Printf("Error happened when retrieving transaction info from pgx table. Err: %s", err)
		return bankID, err
	}
	err = storeDB.QueryRow(ctx, "SELECT bankorderid FROM transactions WHERE transactions_id = ($1);", tID).Scan(&bankID)
	if err != nil {
		log.Printf("Error happened when retrieving bank transaction info from pgx table. Err: %s", err)
		return bankID, err
	}
	
	return bankID, nil

}


// LoadPaidOrders function performs the operation of retrieving order in PAID status from pgx database with a query.
func LoadPaidOrders(ctx context.Context, storeDB *pgxpool.Pool) ([]models.PaidOrderObj, error) {

	var orders []models.PaidOrderObj

	rows, err := storeDB.Query(ctx, "SELECT orders_id, last_updated_at, firstname, email FROM orders WHERE status = ($1);", "PAID")
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				log.Printf("Error happened when retrieving paid orders info from pgx table. Err: %s", err)
				return orders, err
	}

	
	for rows.Next() {
		var paidOrder models.PaidOrderObj
		var updateTimeStorage time.Time
		if err = rows.Scan(&paidOrder.OrdersID, &updateTimeStorage, &paidOrder.Username, &paidOrder.Email); err != nil {
			log.Printf("Error happened when scanning projects. Err: %s", err)
			return orders, err
		}
		paidOrder.LastEditedAt = updateTimeStorage

		orders = append(orders, paidOrder)
	}
	return orders, nil

}

// OrdersToPrint function performs the operation of updating order status from PAID to INPRINT from pgx database with a query.
func OrdersToPrint(ctx context.Context, storeDB *pgxpool.Pool, order models.PaidOrderObj) (error) {

	now:=time.Now()
	tdifference := now.Sub(order.LastEditedAt).Hours()
	if tdifference > 2.0 || tdifference == 2.0 {

		//orderObj, err := RetrieveSingleOrder(ctx , storeDB, order.OrdersID) 

		// Send paid order email
		from := "support@memoryprint.ru"
		to := []string{order.Email}
		subject := "Ваш заказ оплачен!"
		mailType := emailutils.MailPaidOrder
		mailData := &emailutils.MailData{
			Username: order.Username,
			Ordernum: order.OrdersID,
			//Order: orderObj,
		}

		ms := &emailutils.SGMailService{config.YandexApiKey}
		mailReq := emailutils.NewMail(from, to, subject, mailType, mailData)
		err = emailutils.SendMail(mailReq, ms)
		if err != nil {
			log.Printf("unable to send mail", "error", err)
			return err
		}
		_, err = storeDB.Exec(ctx, "UPDATE orders SET status = ($1) WHERE orders_id = ($2);",
			"IN_PRINT",
			order.OrdersID,
		)
		if err != nil {
				log.Printf("Error happened when updating paid order status into pgx table. Err: %s", err)
				return err
		}
	}
	
	return nil

}




