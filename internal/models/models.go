package models

import (
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
	Username string `json:"username"`
	Password string `json:"password"`
	Email string `json:"email"`
	TokenHash string `json:"tokenhash"`
	Category string `json:"category"`
	Status string `json:"status"`
}

type VerificationDataType int

const (
	MailConfirmation VerificationDataType = iota + 1
	MailDesignerOrder VerificationDataType = iota + 2
	MailPassReset VerificationDataType = iota + 3
)



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
	PasswordRe string `json: "password_re"`
	Code 		string `json: "code"`
}


// swagger:model ArtObject 
type ArtObject struct {
	// ID of object -- can be photos, decorations, backgrounds or layouts. Used to store changes from user editing: shadow/transparency/colors, position etc
	// in: int
	ObjectID    uint     `json:"object_id"`
	// Category of object -- applicable for decorations and backgrounds. Can be "HOLIDAY", "WEDDING", "CELEBRATION", "CHILDREN", "PETS", "GENERAL"
	// in: string
	Category string `json:"category"`
	// Type of object -- applicable for decorations. Can be "RIBBON", "FRAME", "STICKER"
	// in: string
	Type string `json:"type"`
	// Name of object 
	// in: string
	Name       string  `json:"name"`
	// Style of object -- this field can be used to store characteristics of the edited object: angle, shadow, transparency etc
	// in: string
	Style      string  `json:"style"`
	// Position of object -- top
	// in: float64
	Ptop       float64 `json:"ptop"`
	// Position of object -- left
	// in: float64
	Pleft      float64 `json:"pleft"`
}

// swagger:model ArtObject 
type PersonalisedObject struct {
	// ID of object -- can be decorations and backgrounds uploaded or liked by the user
	// in: int
	ObjectID    uint     `json:"object_id"`
	// Link of object -- Yandex disk link for retrieving the object
	// in: string
	Link    string `json:"link"`
	// Category of object -- applicable for decorations and backgrounds. Can be "HOLIDAY", "WEDDING", "CELEBRATION", "CHILDREN", "PETS", "GENERAL"
	// in: string
	Category string `json:"category"`
	// Type of object -- applicable for decorations. Can be "RIBBON", "FRAME", "STICKER"
	// in: string
	Type string `json:"type"`
	// Boolean for liked objects
	// in: boolean
	IsFavourite bool `json:"is_favourite"`
	// Boolean for personal uploaded objects
	// in: boolean
	IsPersonal bool `json:"is_personal"`
}

// swagger:model RetrievedPersonalisedObj 
type RetrievedPersonalisedObj struct {
	// Backgrounds -- backgrounds uploaded and / or liked by the user
	// in: slice of personalised objects
	Backgrounds    []PersonalisedObject     `json:"backgrounds"`
	// Decor -- decorations uploaded and / or liked by the user
	// in: slice of personalised objects
	Decor []PersonalisedObject `json:"decor"`
  }

// swagger:model TextObject 
type TextObject struct {
	// CustomText of text box object has the actual content of the text box. In general, the model is used to store changes from user editing: shadow/transparency/colors, position etc
	// in: string
	CustomText       string  `json:"custom_text"`
	// Style of object -- this field can be used to store characteristics of the edited object: angle, shadow, transparency etc
	// in: string
	Style      string  `json:"style"`
	// Position of object -- top
	// in: float64
	Ptop       float64 `json:"ptop"`
	// Position of object -- left
	// in: float64
	Pleft      float64 `json:"pleft"`
}


// swagger:model Photo 
type Photo struct {
	// ID of object -- used for user photo upload
	// in: int
	PhotoID uint
	// Link of object -- Yandex disk link for retrieving the object
	// in: string
	Link    string
}

// swagger:model UserOrder 
type UserOrder struct {
	// Link of object -- Yandex disk link for retrieving the ready for print photobook
	// in: string
	Link string `json:"link"`
	// Status of the order -- can be "SUBMITTED", "PAID", "PRINTED", "DELIVERED", "COMPLETED", "CANCELLED"
	// in: string
	Status string `json:"status"`
	// Number of pages, used for price calculation
	// in: int
	Pagesnum int `json:"pagesnum"`
	// Covertype, can be "HARD" and "SOFT"
	// in: string
	Covertype string `json:"covertype"`
	// Bindingtype, can be "CLASSIC" and "LAYFLAT"
	// in: string
	Bindingtype string `json:"bindingtype"`
	// Papertype, can be "CLASSIC" and "SILK"
	// in: string
	Papertype string `json:"papertype"`
	// The ID of the promocode applied by the user
	// in: int
	PromooffersID    uint     `json:"promooffers_id"`
  }

type AdminOrder struct {
	OrderID    uint     `json:"order_id"`
	Link string `json:"link"`
	Status string `json:"status"`
	Pagesnum int `json:"pagesnum"`
	Covertype string `json:"covertype"`
	Bindingtype string `json:"bindingtype"`
	Papertype string `json:"papertype"`
	UploadedAt string `json:"uploaded_at"`
	LastUpdatedAt string `json:"last_updated_at"`
	UsersID    uint     `json:"users_id"`
	UserEmail string `json:"user_email"`
	PromooffersID    uint     `json:"promooffers_id"`
	PaID    uint     `json:"pa_id"`
  }

type PAAssignment struct {
	PaID    uint     `json:"pa_id"`
	OrderID    uint     `json:"order_id"`
  }


// swagger:model ProjectEditorObj
type ProjectEditorObj struct {
	// ProjectID of the shared project. The model is used for the "Share" option
	// in: int
	ProjectID    uint     `json:"project_id"`
	// Category of the new project editor. The category can take "OWNER" and "EDITOR" values
	// in: string
	Category string `json:"category"`
	// Email of the new project editor. 
	// in: string
	Email string `json:"email"`
	// HardCopy of the photobook for the viewer
	// in: string
	Link string `json:"link"`
	
  }

// swagger:model NewBlankProjectObj
type NewBlankProjectObj struct {
	// ProjectID of the created project. The model is used when a new blank project is created. It returns the id of the project, and ids of project pages, that will be further used for saving changes
	// in: int
	ProjectID    uint     `json:"project_id"`
	// PageIDs, slice of ids of pages of the created project. 
	// in: list
	PagesIDs []uint `json:"pages_ids"`
  }

// swagger:model SavedProjectObj
type SavedProjectObj struct {
	// ProjectID of the created project. The model is used to save changes for an existing project
	// in: int
	Project    ProjectObj     `json:"project"`
	// Pages, slice of dicts that contain information about objects associated with each page
	// in: list
	Pages []Page `json:"pages"`
  }

// swagger:model ProjectObj
type ProjectObj struct {
	// ProjectID of the saved project. The model is used to save changes for an existing project
	// in: int
	ProjectID    uint     `json:"project_id"`
	// Name of the saved project. 
	// in: string
	Name string `json:"name"`
	// Number of pages of the saved project. 
	// in: int
	PageNumber int `json:"page_number"`
	// Yandex Disk Link for the exported cover image. 
	// in: string
	CoverImage string `json:"cover_image"`
	// Photobook format. Available options: "SQUARE", "HORIZONTAL", "VERTICAL"
	// in: string
	Orientation string `json:"orientation"`
	// Covertype, can be "HARD" and "SOFT". Optional for project, compulsory for the order
	// in: string
	Covertype string `json:"covertype"`
	// Bindingtype, can be "CLASSIC" and "LAYFLAT". Optional for project, compulsory for the order
	// in: string
	Bindingtype string `json:"bindingtype"`
	// Papertype, can be "CLASSIC" and "SILK". Optional for project, compulsory for the order
	// in: string
	Papertype string `json:"papertype"`
	// The ID of the promocode applied by the user
	// in: int
	PromooffersID    uint     `json:"promooffers_id"`
	// The ID of the template used to create photobook
	// in: int
	TemplateID    uint     `json:"template_id"`
	LastEditedAt string `json:"last_edited_at"`
  }

// swagger:model TemplateProjectObj
type TemplateProjectObj struct {
	// TemplateID of the saved project. The model is used to save changes for an existing template
	// in: int
	TemplateID    uint     `json:"template_id"`
	// Name of the saved project. 
	// in: string
	Name string `json:"name"`
	// Number of pages of the saved project. 
	// in: int
	PageNumber int `json:"page_number"`
	// Link for the exported cover image. 
	// in: string
	CoverImage string `json:"cover_image"`
	// Photobook format. Available options: "SQUARE", "HORIZONTAL", "VERTICAL"
	// in: string
	Orientation string `json:"orientation"`
	// Category of template. Can be "HOLIDAY", "WEDDING", "CELEBRATION", "CHILDREN", "PETS", "GENERAL"
	// in: string
	Category string `json:"category"`
	// Yandex Disk Link for the exported template. 
	// in: string
	HardCopy string `json:"hardcopy"`
}

// swagger:model RetrievedUserProject
type RetrievedUserProject struct {
	Ownership    ProjectEditorObj     `json:"ownership"`
	Project ProjectObj `json:"project"`
  }

type DesignerProjectObj struct {
	ProjectID    uint     `json:"project_id"`
	Name string `json:"name"`
	PageNumber int `json:"page_number"`
	CoverImage string `json:"cover_image"`
	Orientation string `json:"orientation"`
	Description string `json:"description"`
	Photos []Photo `json:"photos"`
	Covertype string `json:"covertype"`
	Bindingtype string `json:"bindingtype"`
	Papertype string `json:"papertype"`
	PromooffersID    uint     `json:"promooffers_id"`
  }

// swagger:model Page
type Page struct {
	// PageID of the project page. The model is used to save changes made on the page
	// in: int
	PageID uint `json:"page_id"`
	// ProjectID of the saved project. 
	// in: int
	ProjectID uint `json:"project_id"`
	// Decorations, slice of dicts that store data about decors on the page. 
	// in: list
	Decorations []ArtObject `json:"decorations"`
	// Photos, slice of dicts that store data about photos on the page. 
	// in: list
	Photos []ArtObject `json:"photos"`
	// Backgrounds, slice of dicts that store data about backgrounds on the page. 
	// in: list
	Background []ArtObject `json:"background"`
	// Layout, slice of dicts that store data about layout on the page. 
	// in: list
	Layout []ArtObject `json:"layout"`
	// Texts, slice of dicts that store data about texts on the page. 
	// in: list
	TextObj []TextObject `json:"text_obj"`
  }

type Decoration struct {
	// DecorationID of the decoration. The model is used to store data about decoration object
	// in: int
	DecorationID uint `json:"decoration_id"`
	// Yandex Disk Link 
	// in: string
	Link    string `json:"link"`
	// Type of object -- applicable for decorations. Can be "RIBBON", "FRAME", "STICKER"
	// in: string
	Type string `json:"type"`
	// Category of object -- applicable for decorations and backgrounds. Can be "HOLIDAY", "WEDDING", "CELEBRATION", "CHILDREN", "PETS", "GENERAL"
	// in: string
	Category string `json:"category"`
}

type Layout struct {
	// LayoutID . The model is used to store data about layout object
	// in: int
	LayoutID uint `json:"layout_id"`
	// Yandex Disk Link 
	// in: string
	Link    string `json:"link"`
	Type string `json:"type"`
	Category string `json:"category"`
}

type Background struct {
	// BackgroundID . The model is used to store data about background object
	// in: int
	BackgroundID uint `json:"background_id"`
	// Yandex Disk Link 
	// in: string
	Link    string `json:"link"`
	// Category of object -- applicable for decorations and backgrounds. Can be "HOLIDAY", "WEDDING", "CELEBRATION", "CHILDREN", "PETS", "GENERAL"
	// in: string
	Category string `json:"category"`
}


type ProjectSession struct {
	// Decorations, slice of dicts that store data about available decorations. The model is used to upload to the online editor all existing decor, backgrounds and layouts. 
	// in: list
	Decorations []Decoration `json:"decorations"`
	// Backgrounds, slice of dicts that store data about available backgrounds. 
	// in: list
	Background []Background `json:"background"`
	// Layouts, slice of dicts that store data about available layouts. 
	// in: list
	Layout []Layout `json:"layout"`
  }

type Prices struct {
	PricesID uint `json:"prices_id"`
	Price    float64 `json:"price"`
	Pagesnum int `json:"pagesnum"`
	Priceperpage    float64 `json:"priceperpage"`
	Covertype    string `json:"covertype"`
	Bindingtype    string `json:"bindingtype"`
	Papertype    string `json:"papertype"`
}


type PromoOffer struct {
	PromooffersID    uint     `json:"promooffers_id"`
	Discount float64 `json:"discount"`
	ISOnetime bool `json:"is_onetime"`
	ISUsed bool `json:"is_used"`
	ExpiresAt string `json:"expires_at"`
	
  }
