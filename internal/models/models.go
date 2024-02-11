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

type VerificationDataType int

const (
	MailConfirmation VerificationDataType = iota + 1
	MailDesignerOrder VerificationDataType = iota + 2
	MailPassReset VerificationDataType = iota + 3
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
	Link    string     `json:"link"`
	Type    string     `json:"link"`
	Category    string     `json:"link"`
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
	Link    string `json:"link"`
}


type Layout struct {
	// LayoutID . The model is used to store data about layout object
	// in: int
	LayoutID uint `json:"layout_id"`
	// Yandex Disk Link 
	// in: string
	CountImages    string `json:"count_images"`
	Link    string `json:"link"`
	Data        json.RawMessage      `json:"data"`
	IsFavourite bool `json:"is_favourite"`
}

type Background struct {

	BackgroundID uint `json:"background_id"`
	Link    string `json:"link"`
	Type string `json:"type"`
	IsFavourite bool `json:"is_favourite"`
	IsPersonal bool `json:"is_personal"`
}

type Decoration struct {

	DecorationID uint `json:"decoration_id"`
	Link    string `json:"link"`
	Category string `json:"category"`
	Type string `json:"type"`
	IsFavourite bool `json:"is_favourite"`
	IsPersonal bool `json:"is_personal"`
}

  // swagger:model ProjectObj
type ProjectObj struct {
	Name string `json:"name"`
	Size string `json:"size"`
	Variant string `json:"variant"`
	LastEditedAt string `json:"last_edited_at"`
	CreatedAt string `json:"created_at"`
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

// swagger:model SavedProjectObj
type SavedProjectObj struct {
	Project    ProjectObj     `json:"project"`
	Pages []Page `json:"pages"`
  }


type ResponseProject struct {
	ProjectID    uint     `json:"project_id"`
	Name *string `json:"name"`
	CountPages int `json:"count_pages"`
	PreviewImageLink *string `json:"preview_image_link"`
	Size string `json:"size"`
	Variant string `json:"variant"`
	LastEditedAt string `json:"last_edited_at"`
  }

type NewBlankProjectObj struct {
	Name string `json:"name"`
	Size string `json:"size"`
	Variant string `json:"variant"`
	Cover string `json:"cover"`
	Surface string `json:"surface"`
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


type UploadImage struct {
	Image []byte `json:"image"`
	RemoveBackground    bool `json:"remove_background"`
	Extention string `json:"extention"`
}

type RequestBackground struct {
	Offset    uint     `json:"offset"`
	Limit    uint     `json:"limit"`
	Type string `json:"type"`
  }

type RequestDecoration struct {
	Offset    uint     `json:"offset"`
	Limit    uint     `json:"limit"`
	Type string `json:"type"`
	Category string `json:"category"`
  }

type RequestLayout struct {
	Offset    uint     `json:"offset"`
	Limit    uint     `json:"limit"`
	CountImages uint `json:"count_images"`
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

type ResponseTemplate struct {
	TemplateID    uint     `json:"template_id"`
	Name string `json:"name"`
	Size string `json:"size"`
	Category string `json:"category"`
	CreatedAt string `json:"created_at"`
	LastEditedAt string `json:"last_edited_at"`
	Pages []Page `json:"pages"`
  }

type NewTemplateObj struct {
	Size string `json:"size"`
	Category string `json:"category"`
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
	
}

type ResponseUploadedPhoto struct {

	PhotoID uint `json:"photo_id"`
	
}

type ResponseLayout struct {

	Layouts []Layout `json:"layouts"`
	CountAll    string `json:"count_all"`
	
}

type ResponseCreatedLayout struct {

	LayoutID uint `json:"layout_id"`
	
}

type ResponseBackground struct {

	Backgrounds []Background `json:"backgrounds"`
	CountAll    string `json:"count_all"`
	
}

type ResponseCreatedBackground struct {

	BackgroundID uint `json:"background_id"`
	
}

type ResponseDecoration struct {

	Decorations []Decoration `json:"decorations"`
	CountAll    string `json:"count_all"`
	
}

type ResponseCreatedDecoration struct {

	DecorationID uint `json:"decoration_id"`
	
}
