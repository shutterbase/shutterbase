package mail

import (
	"bytes"
	"fmt"

	"github.com/shutterbase/shutterbase/ent"

	"github.com/mxcd/go-config/config"
	"github.com/rs/zerolog/log"

	gomail "gopkg.in/gomail.v2"
)

func SendPasswordResetEmail(user *ent.User) error {
	log.Trace().Str("email", user.Email).Msg("Sending password reset email for user")
	FROM_MAIL := config.Get().String("MAIL_FROM_MAIL")
	SUBJECT_LINE := config.Get().String("MAIL_PASSWORD_RESET_SUBJECT")
	APPLICATION_BASE_URL := config.Get().String("APPLICATION_BASE_URL")

	data := struct {
		User              *ent.User
		Subject           string
		PasswordResetLink string
		EditLink          string
	}{
		User:              user,
		Subject:           SUBJECT_LINE,
		PasswordResetLink: fmt.Sprintf("%s/password-reset/?email=%s&key=%s", APPLICATION_BASE_URL, user.Email, user.PasswordResetKey),
	}

	var tpl bytes.Buffer
	template := GetTemplate("password-reset")
	err := template.Execute(&tpl, data)
	if err != nil {
		log.Error().Err(err).Msg("Error running mail template 'password-reset' ")
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
