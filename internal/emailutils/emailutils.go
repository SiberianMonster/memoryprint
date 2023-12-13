package emailutils

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"log"
	"math/rand"
	"html/template"
	"net/smtp"
	"strings"

	"github.com/SiberianMonster/memoryprint/internal/config"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"


// GenerateRandomString generate a string of random characters of given length
func GenerateRandomString(n int) string {
	sb := strings.Builder{}
	sb.Grow(n)
	for i := 0; i < n; i++ {
		idx := rand.Int63() % int64(len(letterBytes))
		sb.WriteByte(letterBytes[idx])
	}
	return sb.String()
}

// MailService represents the interface for our mail service.
type MailService interface {
	CreateMail(mailReq *Mail) error
	SendMail(mailReq *Mail) error
	NewMail(from string, to []string, subject string, mailType int, data *MailData) *Mail
}

// List of Mail Types we are going to send.
const (
	MailConfirmation int = 1
	MailDesignerOrder int = 2
	MailPassReset int = 3
	MailViewerInvitationNew int = 4
	MailViewerInvitationExist int = 5
)

// MailData represents the data to be sent to the template of the mail.
type MailData struct {
	Username string
	Code	 string
	OwnerName string
	OwnerEmail	 string
	UserEmail	 string
	TempPass	 string
}

// Mail represents a email request
type Mail struct {
	from  string
	to    []string
	subject string
	body string
	mtype int
	data *MailData
}

func (mail *Mail) BuildMessage() string {
	message := ""
	message += fmt.Sprintf("From: %s\r\n", mail.from)
	if len(mail.to) > 0 {
		message += fmt.Sprintf("To: %s\r\n", strings.Join(mail.to, ";"))
	}

	message += fmt.Sprintf("Subject: %s\r\n", mail.subject)
	message += "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	message += "\r\n" + mail.body

	return message
}

func (r *Mail) ParseTemplate(templateFileName string, data interface{}) error {
	t, err := template.ParseFiles(templateFileName)
	if err != nil {
		return err
	}
	buf := new(bytes.Buffer)
	if err = t.Execute(buf, data); err != nil {
		return err
	}
	r.body = buf.String()
	return nil
}

// SGMailService is the sendgrid implementation of our MailService.
type SGMailService struct {
	YandexApiKey             string
	MailVerifCodeExpiration    int		// in hours
	PassResetCodeExpiration    int		// in minutes
	MailVerifTemplateID        string
	PassResetTemplateID        string
	DesignerOrderTemplateID    string
	ViewerInvitationNewTemplateID    string
	ViewerInvitationExistTemplateID    string
}

// NewSGMailService returns a new instance of SGMailService
func NewSGMailService() *SGMailService {
	return &SGMailService{config.SendGridApiKey, config.MailVerifCodeExpiration, config.PassResetCodeExpiration, config.MailVerifTemplateID, config.PassResetTemplateID, config.DesignerOrderTemplateID, config.ViewerInvitationNewTemplateID, config.ViewerInvitationExistTemplateID}
}


// CreateMail takes in a mail request and constructs a sendgrid mail type.
func CreateMail(mailReq *Mail, ms *SGMailService) error {

	var err error
	if mailReq.mtype == MailConfirmation {
		err = mailReq.ParseTemplate("confirm_mail.html", mailReq.data)
	} else if mailReq.mtype == MailPassReset {
		err = mailReq.ParseTemplate("password_reset.html", mailReq.data)
	} else if mailReq.mtype == MailDesignerOrder {
		err = mailReq.ParseTemplate("confirm_mail.html", mailReq.data)
	
	} else if mailReq.mtype == MailViewerInvitationNew {
		err = mailReq.ParseTemplate("viewer_invitation.html", mailReq.data)
	
	
	} else if mailReq.mtype == MailViewerInvitationExist {
		err = mailReq.ParseTemplate("viewer_invitation_exist.html", mailReq.data)
	
	}

	if err != nil{
		return err
	}

	return nil
}

// SendMail creates a sendgrid mail from the given mail request and sends it.
func SendMail(mailReq *Mail, ms *SGMailService) error {

	var auth smtp.Auth
	auth = smtp.PlainAuth("", mailReq.from, ms.YandexApiKey, "smtp.yandex.com")

	err := CreateMail(mailReq, ms)
	if err != nil{
		log.Printf("unable to send mail", "error", err)
		return err
	}
	msg := mailReq.BuildMessage()
	addr := "smtp.yandex.com:465"

	tlsconfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         "smtp.yandex.com",
	}

	conn, err := tls.Dial("tcp", addr, tlsconfig)
	if err != nil {
		log.Printf("failed to create conn")
		return err
	}

	client, err := smtp.NewClient(conn,  "smtp.yandex.com")
	if err != nil {
		log.Printf("failed to create client")
		return err
	}

	// step 1: Use Auth
	if err = client.Auth(auth); err != nil {
		log.Printf("failed to create auth")
		return err
	}

	// step 2: add all from and to
	if err = client.Mail(mailReq.from); err != nil {
		log.Printf("failed to create sender")
		return err
	}

	for _, k := range mailReq.to {
		if err = client.Rcpt(k); err != nil {
			log.Printf("failed to create recepient")
			return err
		}
	}

	// Data
	w, err := client.Data()
	if err != nil {
		log.Printf("failed to create data")
		return err
	}

	_, err = w.Write([]byte(msg))
	if err != nil {
		log.Printf("failed to write msg")
		return err
	}

	err = w.Close()
	if err != nil {
		log.Printf("failed to close conn")
		return err
	}

	client.Quit()

	log.Printf("mail sent successfully")
	return nil
}

// NewMail returns a new mail request.
func NewMail(from string, to []string, subject string, mailType int, data *MailData) *Mail {
	return &Mail{
		from: 	 from,
		to:  	 to,
		subject: subject,
		mtype: 	 mailType,
		data: 	 data,
	}
}

