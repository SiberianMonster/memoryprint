package models

import (
	"encoding/json"
	"time"

)


const (
	CustomerCategory        = "CUSTOMER"
	AdminCategory = "ADMIN"
	PrintAgentUserCategory = "PRINTING AGENT"
	ActiveStatus = "ACTIVE"
	DisactivatedStatus    = "DISACTIVATED"
	VerifiedStatus = "VERIFIED"
	UnverifiedStatus    = "UNVERIFIED"
	RibbonCategory  = "RIBBON"
	TextCategory = "CUSTOMTEXT"
	EditedStatus = "EDITED"
	SubmittedStatus = "SUBMITTED"
	PaidStatus = "PAID"
	ProcessingStatus = "IN PROCESSING"
	CompletedStatus = "COMPLETED"
	CancelledStatus = "CANCELLED"
	OwnerCategory        = "OWNER"
	EditorCategory        = "EDITOR"
	ViewerCategory        = "VIEWER"
)

type User struct {
	ID uint `json:"id"`
	Name string `json:"name"`
	Password string `json:"password"`
	Email string `json:"email"`
	TokenHash string `json:"tokenhash"`
	Category string `json:"category"`
	Status string `json:"status"`
}

type UserInfo struct {
	ID uint `json:"id"`
	Name string `json:"name"`
	Password string `json:"password"`
	Email string `json:"email"`
	TokenHash string `json:"tokenhash"`
	Category string `json:"category"`
	Status string `json:"status"`
	CartObjects uint `json:"cart_objects"`
}

type Viewer struct {
	Name string `json:"name"`
	Email string `json:"email"`
	
}

type Price struct {
	Cover string `json:"cover"`
	Variant string `json:"variant"`
	Surface string `json:"surface"`
	Size string `json:"size"`
	BasePrice float64 `json:"base_price"`
	ExtraPage float64 `json:"extra_page"`
}

type ResponsePrice struct {
	Prices []Price `json:"prices"`
}

type Colour struct {
	ID uint `json:"id"`
	ColourLink string `json:"colour_link"`
	HexCode string `json:"hex_code"`
}

type ResponseColour struct {
	Colours []Colour `json:"colours"`
}

type GiftCertificate struct {
	ID uint `json:"id"`
	Code string `json:"code"`
	Deposit      float64    `json:"deposit"`
	Recipientemail string `json:"recipientemail" validate:"required,email"`
	Recipientname string `json:"recipientname" validate:"required"`
	Buyerfirstname string `json:"buyerfirstname" validate:"required"`
	Buyerlastname string `json:"buyerlastname" validate:"required"`
	Buyeremail string `json:"buyeremail" validate:"required,email"`
	Buyerphone string `json:"buyerphone" validate:"required,e164"`
	MailAt int64 `json:"mail_at"`
	TransactionID string `json:"transaction_id"`
}

type NewPromooffer struct {
	Code string `json:"code"`
	Discount      float64    `json:"discount"`
	Category string `json:"category"`
	ExpiresAt int64 `json:"expires_at"`
	IsOnetime bool `json:"is_onetime"`
	Templates []Template `json:"templates"`
	UsersID int64 `json:"users_id"`
}

type Promooffer struct {
	Code string `json:"code"`
	Discount      float64    `json:"discount"`
	Category string `json:"category"`
	ExpiresAt int64 `json:"expires_at"`
	Templates []Template `json:"templates"`
}

type Promooffers struct {
	Promocodes []Promooffer `json:"promocodes"  validate:"required"`
	
}

type CheckPromooffer struct {
	Code string `json:"code"  validate:"required"`
	
}


type RequestPromooffer struct {
	Projects    []uint     `json:"projects" validate:"required"`
	Code string `json:"code" validate:"required"`
  }

type PromocodeCheck struct {
	Code string `json:"code" validate:"required"`
	
}

type CheckPromocode struct {
	Status    string     `json:"status" validate:"required"`
	Promocode ResponsePromocode `json:"promocode" validate:"required"`
  }

type ResponsePromocode struct {

	Discount    float64 `json:"discount"`
	Category string `json:"category"`
	ExpiresAt int64 `json:"expires_at"`
	
}

type ResponsePromocodeUse struct {

	PromocodeID uint `json:"promocode_id"`
	Discount    float64 `json:"discount"`
	Category    string `json:"category"`
	BasePrice float64 `json:"base_price"`
	DiscountedPrice float64 `json:"discounted_price"`
}

type RequestCertificate struct {
	DiscountedPrice    float64     `json:"discounted_price" validate:"required"`
	Code string `json:"code" validate:"required"`
  }

type ResponseCertificate struct {

	Deposit float64 `json:"deposit"`
}

type TransactionLink struct {
	PaymentLink uint `json:"payment_link" validate:"required"`
	
}

type SignUpUser struct {
	Name string `json:"name" validate:"required,min=1,max=20"`
	Password string `json:"password" validate:"required,min=6,max=20"`
	Email string `json:"email" validate:"required,email"`
}

type UpdatedUsername struct {
	Name string `json:"name"`
}

type UpdatedUser struct {
	Password string `json:"password" validate:"required,min=6,max=20"`
	NewPassword string `json:"new_password" validate:"min=6,max=20,necsfield=Password"`
}

type LoginUser struct {
	Password string `json:"password" validate:"required,min=6,max=20"`
	Email string `json:"email" validate:"required,email"`
}

type RestoreUser struct {
	Email string `json:"email" validate:"required,email"`
}

type VerificationDataType int

const (
	MailConfirmation VerificationDataType = iota + 1
	MailDesignerOrder VerificationDataType = iota + 2
	MailPassReset VerificationDataType = iota + 3
	MailGiftCertificate VerificationDataType = iota + 4
)

type ErrorResp struct {
    Error struct {
        Foo string
    }
}

// VerificationData represents the type for the data stored for verification.
type VerificationData struct {
	ID uint `json:"id"`
	Email     string    `json:"email"`
	Code      string    `json:"code"`
	ExpiresAt time.Time `json:"expires_at"`
	Type      int    `json:"type"`
}


type PasswordResetReq struct {
	Password string `json: "password"`
}



// swagger:model ArtObject 
type PersonalisedObject struct {
	// ID of object -- can be decorations and backgrounds uploaded or liked by the user
	// in: int
	ObjectID    uint     `json:"object_id"`
	// Link of object -- Yandex disk link for retrieving the object
	// in: string
	Link    string     `json:"link" validate:"required"`
	Type    string     `json:"type"`
	Category    string     `json:"category"`
	IsFavourite bool `json:"is_favourite"`
	// Boolean for personal uploaded objects
	// in: boolean
	IsPersonal bool `json:"is_personal"`
}


// swagger:model Photo 
type Photo struct {
	// ID of object -- used for user photo upload
	// in: int
	PhotoID uint `json:"photo_id"`
	// Link of object -- Yandex disk link for retrieving the object
	// in: string
	Link    string `json:"link" validate:"required"`
	UploadedAt  int64 `json:"uploaded_at"`
}


type Layout struct {
	// LayoutID . The model is used to store data about layout object
	// in: int
	LayoutID uint `json:"layout_id"`
	// Yandex Disk Link 
	// in: string
	CountImages    uint `json:"count_images" validate:"gte=1,lte=6"`
	Size    string `json:"size" validate:"required,oneof=SMALL_SQUARE SQUARE VERTICAL HORIZONTAL"`
	Link    string `json:"link" validate:"required"`
	Data        json.RawMessage      `json:"data" validate:"required"`
	IsFavourite bool `json:"is_favourite"`
}

type Background struct {

	BackgroundID uint `json:"background_id"`
	Link    string `json:"link"`
	Type string `json:"type" validate:"required,oneof=VACATION WEDDING HOLIDAYS CHILDREN ANIMALS UNIVERSAL"`
	IsFavourite bool `json:"is_favourite"`
	IsPersonal bool `json:"is_personal"`
}

type Decoration struct {

	DecorationID uint `json:"decoration_id"`
	Link    string `json:"link" validate:"required"`
	Category string `json:"category" validate:"required,oneof=VACATION WEDDING HOLIDAYS CHILDREN ANIMALS UNIVERSAL"`
	Type string `json:"type" validate:"required,oneof=RIBBON FRAME STICKER"` 
	IsFavourite bool `json:"is_favourite"`
	IsPersonal bool `json:"is_personal"`
}

  // swagger:model ProjectObj
type ProjectObj struct {
	Name string `json:"name"`
	Size string `json:"size"`
	Cover string `json:"cover"`
	Variant string `json:"variant"`
	Surface string `json:"surface"`
	PreviewLink string `json:"preview_link"`
	PrintLink string `json:"print_link"`
	CountPages int`json:"count_pages"`
	LastEditedAt int64`json:"updated_at"`
	CreatedAt int64 `json:"created_at"`
	LeatherID uint `json:"leather_id"`
  }

// swagger:model Page
type Page struct {
	// PageID of the project page. The model is used to save changes made on the page
	// in: int
	PageID uint `json:"page_id"`
	Type string `json:"type"`
	Sort uint `json:"sort"`
	CreatingImageLink *string `json:"creating_image_link"`
	Data        json.RawMessage      `json:"data"`
	UsedPhotoIDs []uint `json:"used_photo_ids"`
	
  }


// swagger:model TemplatePage
type TemplatePage struct {
	// PageID of the project page. The model is used to save changes made on the page
	// in: int
	PageID uint `json:"page_id"`
	Type string `json:"type"`
	Sort uint `json:"sort"`
	CreatingImageLink *string `json:"creating_image_link"`
	Data        json.RawMessage      `json:"data"`
	
  }


// swagger:model SavedTemplateObj
type SavedTemplateObj struct {
	Name string `json:"name"`
	Size string `json:"size"`
	LastEditedAt int64 `json:"updated_at"`
	CreatedAt int64 `json:"created_at"`
	Pages []TemplatePage `json:"pages"`
  }

type ResponseProjects struct {
	Projects    []ResponseProject     `json:"projects"`
	CountAll    int `json:"count_all"`

}

type ResponseProject struct {
	ProjectID    uint     `json:"project_id"`
	Name *string `json:"name"`
	Status string `json:"status"`
	Size string `json:"size"`
	FrontPage FrontPage `json:"front_page"`
  }

type NewBlankProjectObj struct {
	Name string `json:"name"`
	Size string `json:"size" validate:"required,oneof=SMALL_SQUARE SQUARE VERTICAL HORIZONTAL"`
	Variant string `json:"variant"  validate:"required,oneof=STANDARD PREMIUM"`
	Cover string `json:"cover" validate:"required,oneof=HARD LEATHERETTE"`
	Surface string `json:"surface" validate:"required,oneof=GLOSS MATTE"`
	PreviewLink string `json:"preview_link"`
	PrintLink string `json:"print_link"`
	CountPages int`json:"count_pages"`
	LeatherID uint `json:"leather_id"`
	TemplateID uint `json:"template_id"`
  }

type CartObj struct {
	ProjectID    uint     `json:"project_id"`
	Name string `json:"name"`
	Size string `json:"size" validate:"required,oneof=SMALL_SQUARE SQUARE VERTICAL HORIZONTAL"`
	Variant string `json:"variant"  validate:"required,oneof=STANDARD PREMIUM"`
	Cover string `json:"cover" validate:"required,oneof=HARD LEATHERETTE"`
	Surface string `json:"surface" validate:"required,oneof=GLOSS MATTE"`
	Category string `json:"category" validate:"required,oneof=VACATION WEDDING HOLIDAYS CHILDREN ANIMALS UNIVERSAL"`
	PreviewLink string `json:"preview_link"`
	FrontPage FrontPage `json:"front_page"`
	CountPages int `json:"count_pages"`
	BasePrice float64 `json:"base_price"`
	UpdatedPagesPrice float64 `json:"updated_pages_price"`
	UpdatedCoverPrice float64 `json:"updated_cover_price"`
	CoverBool bool `json:"cover_bool"`
	LeatherImage string `json:"leather_image"`
  }

type PaidCartObj struct {
	ProjectID    uint     `json:"project_id"`
	Name string `json:"name"`
	Size string `json:"size" validate:"required,oneof=SMALL_SQUARE SQUARE VERTICAL HORIZONTAL"`
	Variant string `json:"variant"  validate:"required,oneof=STANDARD PREMIUM"`
	Cover string `json:"cover" validate:"required,oneof=HARD LEATHERETTE"`
	Surface string `json:"surface" validate:"required,oneof=GLOSS MATTE"`
	FrontPage FrontPage `json:"front_page"`
	CountPages int `json:"count_pages"`
	BasePrice float64 `json:"base_price"`
	LeatherImage string `json:"leather_image"`
  }

type ResponseCart struct {
	Projects   []CartObj     `json:"projects"`

}

type RequestOrderPayment struct {

	Projects    []uint     `json:"projects" validate:"required"`
	ContactData Contacts `json:"contact_data" validate:"required"`
	DeliveryData Delivery `json:"delivery_data" validate:"required"`
	PackageBox bool `json:"package_box"`
	Giftcertificate string `json:"giftcertificate"`
	Promocode string `json:"promocode"`
  }

type ResponseOrderInfo struct {

	UserID    uint     `json:"user_id" validate:"required"`
	ContactData Contacts `json:"contact_data" validate:"required"`
	DeliveryData Delivery `json:"delivery_data" validate:"required"`
	GiftcertificateDeposit float64 `json:"giftcertificate_deposit"`
	PromocodeDiscountPercent float64 `json:"promocode_discount_percent"`
	Promocode string `json:"promocode"`
	TransactionID uint `json:"transaction_id"`
	PreviewLinks    []PreviewObject     `json:"preview_links" validate:"required"`
  }



type ResponseDeliveryInfo struct {

	OrderID    uint     `json:"order_id" validate:"required"`
	UserID    uint     `json:"user_id" validate:"required"`
	TrackingNumber string `json:"tracking_number" validate:"required"`
	DeliveryStatus string `json:"delivery_status" validate:"required"`
	DeliveryID string `json:"delivery_id" validate:"required"`
	Address string `json:"address" validate:"required"`
	Code uint `json:"address" validate:"required"`
	ContactData Contacts `json:"contact_data" validate:"required"`
	Method string `json:"method" validate:"required"`
	ExpectedDeliveryFrom int64 `json:"expected_delivery_from" validate:"required"`
	ExpectedDeliveryTo int64 `json:"expected_delivery_to" validate:"required"`
  }


type ResponseApiDeliveryInfo struct {

	OrderID    uint     `json:"order_id" validate:"required"`
	Projects uint `json:"projects" validate:"required"`
	TrackingNumber string `json:"tracking_number" validate:"required"`
	DeliveryStatus string `json:"delivery_status" validate:"required"`
	DeliveryID string `json:"delivery_id" validate:"required"`
	Address string `json:"address" validate:"required"`
	PostalCode string `json:"postal_code" validate:"required"`
	Code string `json:"code" validate:"required"`
	ContactData Contacts `json:"contact_data" validate:"required"`
	Method string `json:"method" validate:"required"`
	Amount uint `json:"amount" validate:"required"`
  }

type PreviewObject struct {
	Name string `json:"name"`
	Link string `json:"link"`
}

type Contacts struct {
	FirstName string `json:"first_name"`
	LastName string `json:"last_name"`
	Email string `json:"email" validate:"required,email"`
	Phone string `json:"phone" validate:"required,e164"`
}

type Delivery struct {
	Method string `json:"method" validate:"required"`
	Code string `json:"code"`
	PostalCode string `json:"method"`
	Address string `json:"address"`
	Amount float64 `json:"amount" validate:"required"`
}

type ResponsePayment struct {

	FormUrl    string    `json:"form_url"`
  }

type UpdateSurface struct {

	Surface string `json:"surface" validate:"required,oneof=GLOSS MATTE"`
  }

type UpdateCover struct {

	Cover string `json:"cover" validate:"required,oneof=HARD LEATHERETTE"`
	LeatherID uint `json:"leather_id" validate:"required`
  }

type NewOrder struct {

	ProjectID    uint     `json:"project_id"`
	PreviewLink string `json:"preview_link"`
	PrintLink string `json:"print_link"`

  }

type SavePage struct {
	// PageID of the project page. The model is used to save changes made on the page
	// in: int
	PageID uint `json:"page_id"`
	PreviewImageLink *string `json:"preview_image_link"`
	CreatingImageLink *string `json:"creating_image_link"`
	Data        json.RawMessage      `json:"data"`
	UsedPhotoIDs []uint `json:"used_photo_ids"`
  }

type FrontPage struct {
	// PageID of the project page. The model is used to save changes made on the page
	// in: int
	CreatingImageLink *string `json:"creating_image_link"`
  }


type UploadImage struct {
	Image []byte `json:"image"`
	RemoveBackground    bool `json:"remove_background"`
	Extention string `json:"extention"`
}

type RequestBackground struct {
	Offset    uint     `json:"offset"`
	Limit    uint     `json:"limit"`
	Type string `json:"type"`
	IsFavourite bool `json:"is_favourite"`
	IsPersonal bool `json:"is_personal"`
  }

type RequestDecoration struct {
	Offset    uint     `json:"offset"`
	Limit    uint     `json:"limit"`
	Type string `json:"type"`
	Category string `json:"category"`
	IsFavourite bool `json:"is_favourite"`
	IsPersonal bool `json:"is_personal"`
  }

type RequestTemplate struct {
	Offset    uint     `json:"offset"`
	Limit    uint     `json:"limit"`
	Category string `json:"category"`
	Size string `json:"size"`
	Status string `json:"status" "`
  }

type RequestLayout struct {
	Offset    uint     `json:"offset"`
	Limit    uint     `json:"limit"`
	Size    string     `json:"size"`
	CountImages uint `json:"countimages"`
	IsFavourite bool `json:"isfavourite"`
  }

type RequestSavePages struct {
	Pages []SavePage `json:"pages"`
}


type NewPage struct {
	
	CloneID uint `json:"clone_id"`
	Sort uint `json:"sort"`

  }

type RequestAddPage struct {
	
	Pages []NewPage `json:"pages"`
	

  }

type RequestDeletePage struct {
	
	PageIDs []uint `json:"page_ids"`
	

  }

type OrderPage struct {
	
	PageID uint `json:"page_id"`
	Sort uint `json:"sort"`

  }

type RequestReorderPage struct {
	
	Pages []OrderPage `json:"pages"`
	

  }

type ResponseTemplates struct {

	Templates []Template `json:"templates"`
	CountAll    int `json:"count_all"`
	
}


// swagger:model SavedProjectObj
type SavedProjectObj struct {
	Project    ProjectObj     `json:"project"`
	Pages []Page `json:"pages"`
  }

type Template struct {
	TemplateID    uint     `json:"template_id"`
	Name string `json:"name"`
	Size string `json:"size"`
	Status string `json:"status"  validate:"required,oneof=PUBLISHED EDITED"`
	FrontPage FrontPage `json:"front_page"`
  }

type Project struct {
	TemplateID    uint     `json:"template_id"`
	Name string `json:"name"`
	Size string `json:"size"`
	PreviewImageLink string `json:"preview_image_link"`
	Status string `json:"status"  validate:"required,oneof=PUBLISHED EDITED"`
	FrontPage FrontPage `json:"front_page"`
  }



type NewTemplateObj struct {
	Category string `json:"category" validate:"required,oneof=VACATION WEDDING HOLIDAYS CHILDREN ANIMALS UNIVERSAL"`
	Name string `json:"name" validate:"required,min=2,max=40"`
	Size string `json:"size"`
}

type RequestPhotos struct {
	Offset    uint     `json:"offset"`
	Limit    uint     `json:"limit"`
  }

type ResponseCreatedProject struct {

	ProjectID uint `json:"project_id"`
	
}

type ResponseCreatedTemplate struct {

	TemplateID uint `json:"template_id"`
	
}

type ResponsePhotos struct {

	Photos []Photo `json:"photos"`
	CountAll    int `json:"count_all"`
	
}

type ResponseUploadedPhoto struct {

	PhotoID uint `json:"photo_id"`
	
}

type ResponseLayout struct {

	Layouts []Layout `json:"layouts"`
	CountAll    int `json:"count_all"`
	CountFavourite    int `json:"count_favourite"`
	
}

type ResponseCreatedLayout struct {

	LayoutID uint `json:"layout_id"`
	
}

type ResponseBackground struct {

	Backgrounds []Background `json:"backgrounds"`
	CountAll    int `json:"count_all"`
	CountFavourite    int `json:"count_favourite"`
	CountPersonal    int `json:"count_personal"`
	
}

type ResponseCreatedBackground struct {

	BackgroundID uint `json:"background_id"`
	
}

type ResponseDecoration struct {

	Decorations []Decoration `json:"decorations"`
	CountAll    int `json:"count_all"`
	CountFavourite    int `json:"count_favourite"`
	CountPersonal    int `json:"count_personal"`
	
}

type ResponseCreatedDecoration struct {

	DecorationID uint `json:"decoration_id"`
	
}

type ResponseCreatedCertificate struct {

	GiftcertificateID uint `json:"giftcertificate_id"`
	
}

type ResponseOrder struct {

    OrderID uint `json:"order_id"`
	Projects    []PaidCartObj     `json:"projects" validate:"required"`
	Status string `json:"status" validate:"required"`
	CreatedAt int64 `json:"created_at"`
	BasePrice float64 `json:"base_price"`
	FinalPrice float64 `json:"final_price"`
	TrackingNumber string `json:"tracking_number"`
	PromocodeDiscountPercent float64 `json:"promocode_discount_percent"`
	PromocodeCategory string `json:"promocode_category"`
	PromocodeDiscount float64 `json:"promocode_discount"`
	CertificateDeposit float64 `json:"certificate_deposit"`
  }


type ResponseOrders struct {

	Orders []ResponseOrder `json:"orders"`
	CountAll    int `json:"count_all"`
	
}

type ResponseAdminOrder struct {

    OrderID uint `json:"order_id"`
	UserID uint `json:"user_id"`
	Projects    []PaidCartObj     `json:"projects" validate:"required"`
	Status string `json:"status" validate:"required"`
	Email string `json:"email" validate:"required"`
	Commentary string `json:"commentary" validate:"required"`
	CreatedAt int64 `json:"created_at"`
	BasePrice float64 `json:"base_price"`
	FinalPrice float64 `json:"final_price"`
	TrackingNumber string `json:"tracking_number"`
	VideoLink string `json:"video_link"`
	PromocodeDiscountPercent float64 `json:"promocode_discount_percent"`
	PromocodeCategory string `json:"promocode_category"`
	PromocodeDiscount float64 `json:"promocode_discount"`
	CertificateDeposit float64 `json:"certificate_deposit"`
  }

type ResponseAdminOrders struct {

	Orders []ResponseAdminOrder `json:"orders"`
	CountAll    int `json:"count_all"`
	
} 

type RequestUpdateOrderStatus struct {

	Status string `json:"status" validate:"required"`
  }

type RequestUpdateOrderCommentary struct {

	Commentary string `json:"commentary" validate:"required"`
  }

type OrderVideo struct {

	VideoLink string `json:"video_link" validate:"required"`
  }

type ResponseTransaction struct {
	OrderID string `json:"orderId" validate:"required"`
	ErrorCode uint `json:"errorCode" validate:"required"`
	ErrorMessage string `json:"errorMessage" validate:"required"`
	FormURL string `json:"formUrl" validate:"required"`
}

type ResponseTransactionStatus struct {
	OrderNumber uint `json:"orderNumber" validate:"required"`
	ActionCode uint `json:"actionCode" validate:"required"`
	ActionCodeDescription string `json:"actionCodeDescription" validate:"required"`
	Amount float64 `json:"amount"`
	Date int64 `json:"date"`
	IP string `json:"ip"`
	PaymentWay uint `json:"paymentWay" validate:"required"`
}

type ResponsePaymentStatus struct {
	
	Status string `json:"status" validate:"required"`

}
type UserRequestDeliveryCost struct {
	Method string `json:"method" validate:"required"`
	PostalCode string `json:"postal_code" validate:"required"`
	Address string `json:"address" validate:"required"`
	Code string `json:"address" validate:"required"`
	CountProjects int `json:"count_projects" validate:"required"`
} 

type RequestDeliveryCost struct {
	TariffCode string `json:"tariff_code" validate:"required"`
	FromLocation Location `json:"from_location" validate:"required"`
	ToLocation Location `json:"to_location" validate:"required"`
	Packages Package `json:"packages" validate:"required"`
}

type Location struct {
	
	PostalCode string `json:"postal_code"`
	Address string `json:"address"`
	Code string `json:"code"`

}

type Package struct {
	
	Weight float64 `json:"weight" validate:"required"`
	Length float64 `json:"length" validate:"required"`
	Width float64 `json:"width" validate:"required"`
	Height float64 `json:"height" validate:"required"`
}

type ApiResponseDeliveryCost struct {
	PeriodMin int `json:"period_min" validate:"required"`
	PeriodMax int `json:"period_max" validate:"required"`
	DeliverySum float64 `json:"delivery_sum" validate:"required"`
	WeightCalc float64 `json:"weight_calc" validate:"required"`
	Currency string `json:"period_max" validate:"required"`
	TotalSum float64 `json:"total_sum" validate:"required"`
	Services []Service `json:"services" validate:"required"`
}

type ResponseDeliveryCost struct {

	Amount float64 `json:"amount" validate:"required"`
	ExpectedDeliveryFrom string `json:"expected_delivery_from" validate:"required"`
	ExpectedDeliveryTo string `json:"expected_delivery_to" validate:"required"`
}


type Service struct {
	
	Code string `json:"code" validate:"required"`
	Sum float64 `json:"sum" validate:"required"`
}

// swagger:model PaidOrderObj
type PaidOrderObj struct {
	OrdersID uint `json:"orders_id"`
	LastEditedAt time.Time`json:"last_edited_at"`
	Username string`json:"username"`
	Email string`json:"email"`
}

