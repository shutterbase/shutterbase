package mail

import (
	"bytes"
	"fmt"

	"github.com/shutterbase/shutterbase/ent"

	"github.com/mxcd/go-config/config"
	"github.com/rs/zerolog/log"

	gomail "gopkg.in/gomail.v2"
)

func SendEmailConfirmation(user *ent.User) error {
	FROM_MAIL := config.Get().String("MAIL_FROM_MAIL")
	SUBJECT_LINE := config.Get().String("MAIL_EMAIL_CONFIRMATION_SUBJECT")
	API_BASE_URL := config.Get().String("API_BASE_URL")

	data := struct {
		User             *ent.User
		Subject          string
		ConfirmationLink string
		EditLink         string
	}{
		User:             user,
		Subject:          SUBJECT_LINE,
		ConfirmationLink: fmt.Sprintf("%s/confirm/?email=%s&key=%s", API_BASE_URL, user.Email, user.ValidationKey),
	}

	var tpl bytes.Buffer
	template := GetTemplate("email-confirmation")
	err := template.Execute(&tpl, data)
	if err != nil {
		log.Print("Error running mail template")
		log.Print(err)
		return err
	}

	html := tpl.String()

	msg := gomail.NewMessage()
	msg.SetHeader("From", FROM_MAIL)
	msg.SetHeader("To", user.Email)
	msg.SetHeader("Subject", SUBJECT_LINE)
	msg.SetBody("text/html", html)

	err = GetMailer().DialAndSend(msg)
	if err != nil {
		log.Error().Msg("Error sending mail")
	}

	return err
}
