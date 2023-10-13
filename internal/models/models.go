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
	PassReset
)

// VerificationData represents the type for the data stored for verification.
type VerificationData struct {
	Email     string    `json:"email" validate:"required" sql:"email"`
	Code      string    `json:"code" validate:"required" sql:"code"`
	ExpiresAt time.Time `json:"expiresat" sql:"expiresat"`
	Type      VerificationDataType    `json:"type" sql:"type"`
}


type PasswordResetReq struct {
	Password string `json: "password"`
	PasswordRe string `json: "password_re"`
	Code 		string `json: "code"`
}

type ArtObject struct {
	ObjectID    uint     `json:"object_id"`
	Category string `json:"category"`
	Type string `json:"type"`
	Name       string  `json:"name"`
	Style      string  `json:"style"`
	Ptop       float64 `json:"ptop"`
	Pleft      float64 `json:"pleft"`
}

type TextObject struct {

	CustomText       string  `json:"custom_text"`
	Style      string  `json:"style"`
	Ptop       float64 `json:"ptop"`
	Pleft      float64 `json:"pleft"`
}


type Photo struct {
	PhotoID uint
	Link    string
}

type UserOrder struct {
	Link string `json:"link"`
	Status string `json:"status"`
  }

type AdminOrder struct {
	OrderID    uint     `json:"order_id"`
	Link string `json:"link"`
	Status string `json:"status"`
	UploadedAt string `json:"uploaded_at"`
	LastUpdatedAt string `json:"last_updated_at"`
	UsersID    uint     `json:"users_id"`
	UserEmail string `json:"user_email"`
	PaID    uint     `json:"pa_id"`
  }

type PAAssignment struct {
	PaID    uint     `json:"pa_id"`
	OrderID    uint     `json:"order_id"`
  }

type ProjectEditorObj struct {
	ProjectID    uint     `json:"project_id"`
	Category string `json:"category"`
	Email string `json:"email"`
  }

type ProjectObj struct {
	ProjectID    uint     `json:"project_id"`
	Name string `json:"name"`
	CoverImage string `json:"cover_image"`
	Status string `json:"status"`
	LastEditedAt string `json:"last_edited_at"`
  }

type NewProjectObj struct {
	Name string `json:"name"`
	PageNumber int `json:"page_number"`
  }

type Page struct {
	PageID uint `json:"page_id"`
	ProjectID uint `json:"project_id"`
	Decorations []ArtObject `json:"decorations"`
	Photos []ArtObject `json:"photos"`
	Background []ArtObject `json:"background"`
	Layout []ArtObject `json:"layout"`
	TextObj []TextObject `json:"text_obj"`
  }

type Decoration struct {
	DecorationID uint `json:"decoration_id"`
	Link    string `json:"link"`
	Type int `json:"type"`
	Category int `json:"category"`
}

type Layout struct {
	LayoutID uint `json:"layout_id"`
	Link    string `json:"link"`
	Category int `json:"category"`
}

type Background struct {
	BackgroundID uint `json:"background_id"`
	Link    string `json:"link"`
	Category int `json:"category"`
}


type ProjectSession struct {
	Decorations []Decoration `json:"decorations"`
	Background []Background `json:"background"`
	Layout []Layout `json:"layout"`
  }