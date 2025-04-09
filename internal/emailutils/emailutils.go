package emailutils

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"log"
	"crypto/rand"
	"math/big"
	"html/template"
	"net/smtp"
	"strings"

	"github.com/SiberianMonster/memoryprint/internal/config"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()_+"


// GenerateRandomString generate a string of random characters of given length
func GenerateRandomString(n int) string {
	password := make([]byte, n)
    charsetLength := big.NewInt(int64(len(charset)))
    for i := range password {
        index, _ := rand.Int(rand.Reader, charsetLength)
        password[i] = charset[index.Int64()]
    }

    return string(password)
}

// MailService represents the interface for our mail service.
type MailService interface {
	CreateMail(mailReq *Mail) error
	SendMail(mailReq *Mail) error
	NewMail(from string, to []string, subject string, mailType int, data *MailData) *Mail
}

// List of Mail Types we are going to send.
const (
	MailWelcome int = 1
	MailPassTemp int = 2
	MailPaidOrder int = 3
	MailOrderInDelivery int = 4
	MailViewerInvitation int = 5
	MailGiftCertificate int = 6
	MailAdminPaidOrder int = 7
	MailAdminCancelledOrder int = 8
)

// MailData represents the data to be sent to the template of the mail.
type MailData struct {
	Username string
	Code	 string
	OwnerName string
	OwnerEmail	 string
	UserEmail	 string
	TempPass	 string
	Ordernum uint
	Trackingnum string
	SubscriptionLink string 
	ShareLink string
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
}

// NewSGMailService returns a new instance of SGMailService
func NewSGMailService() *SGMailService {
	return &SGMailService{config.YandexApiKey}
}


// CreateMail takes in a mail request and constructs a sendgrid mail type.
func CreateMail(mailReq *Mail, ms *SGMailService) error {

	var err error
	if mailReq.mtype == MailWelcome {
		err = mailReq.ParseTemplate("confirm_mail.html", mailReq.data)
	} else if mailReq.mtype == MailPassTemp {
		err = mailReq.ParseTemplate("password_reset.html", mailReq.data)
	} else if mailReq.mtype == MailPaidOrder {
		err = mailReq.ParseTemplate("paid_order_mail.html", mailReq.data)
	
	} else if mailReq.mtype == MailOrderInDelivery {
		err = mailReq.ParseTemplate("delivery_order_mail.html", mailReq.data)
		
	} else if mailReq.mtype == MailViewerInvitation {
		err = mailReq.ParseTemplate("viewer_invitation.html", mailReq.data)
	
	
	} else if mailReq.mtype == MailGiftCertificate {
		err = mailReq.ParseTemplate("gift_certificate.html", mailReq.data)
	
	} else if mailReq.mtype == MailAdminPaidOrder {
			err = mailReq.ParseTemplate("paid_order_admin_mail.html", mailReq.data)
		
	} else if mailReq.mtype == MailAdminCancelledOrder {
			err = mailReq.ParseTemplate("cancelled_order_admin_mail.html", mailReq.data)
			
	}

	if err != nil{
		return err
	}

	return nil
}

// SendMail creates a sendgrid mail from the given mail request and sends it.
func SendMail(mailReq *Mail, ms *SGMailService) error {

	var auth smtp.Auth
	auth = smtp.PlainAuth("", mailReq.from, ms.YandexApiKey, "smtp.yandex.ru")
	//auth = smtp.PlainAuth("", mailReq.from, "smtp-tls-key", "0.0.0.0")
	print(ms.YandexApiKey)

	err := CreateMail(mailReq, ms)
	if err != nil{
		log.Printf("unable to send mail", "error", err)
		return err
	}
	msg := mailReq.BuildMessage()
	addr := "smtp.yandex.ru:465"
	//addr := "0.0.0.0:1025"

	tlsconfig := &tls.Config{
		InsecureSkipVerify: true,
		//ServerName:         "0.0.0.0",
		ServerName:         "smtp.yandex.ru",
	}

	conn, err := tls.Dial("tcp", addr, tlsconfig)
	//conn, err := smtp.Dial(addr)
	if err != nil {
		log.Printf("failed to create conn")
		return err
	}
	//conn.StartTLS(tlsconfig)

    client, err := smtp.NewClient(conn,  "smtp.yandex.ru")
 	if err != nil {
 		log.Printf("failed to create client")
 		return err
 	}
 
 	// step 1: Use Auth
 	if err = client.Auth(auth); err != nil {
 		log.Printf("failed to create auth")
 		return err
 	}
	// Auth
    //if err = conn.Auth(auth); err != nil {
    //    log.Panic(err)
    //}

	// step 2: add all from and to
	if err = client.Mail(mailReq.from); err != nil {
			log.Printf("failed to create sender")
			return err
	}

    // To && From
    //if err = conn.Mail(mailReq.from); err != nil {
    //    log.Panic(err)
    //}

	for _, k := range mailReq.to {
		if err = client.Rcpt(k); err != nil {
		//if err = conn.Rcpt(k); err != nil {
			log.Printf("failed to create recepient")
			return err
		}
	}

    // Data
    //w, err := conn.Data()
    //if err != nil {
    //    log.Panic(err)
    //}


	//client, err := smtp.NewClient(conn,  "smtp.yandex.ru")
	//client, err := smtp.NewClient(conn,  "0.0.0.0")
	//if err != nil {
	//	log.Printf("failed to create client")
	//	return err
	//}

	// step 1: Use Auth
	//if err = client.Auth(auth); err != nil {
	//	log.Printf("failed to create auth")
	//	return err
	//}

	// step 2: add all from and to
	//if err = client.Mail(mailReq.from); err != nil {
	//	log.Printf("failed to create sender")
	//	return err
	//}

	//for _, k := range mailReq.to {
	//	if err = client.Rcpt(k); err != nil {
	//		log.Printf("failed to create recepient")
	//		return err
	//	}
	//}

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
	//conn.Quit()

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

