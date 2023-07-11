package mail

import (
	"html/template"
	"path/filepath"
	"strings"

	"github.com/mxcd/go-config/config"
	"github.com/rs/zerolog/log"

	gomail "gopkg.in/gomail.v2"
)

var mailer *gomail.Dialer
var mailerInitOk bool = false
var devMode bool = false
var templates = make(map[string]*template.Template)

var templatePaths = []string{
	"mail-templates/email-confirmation.html",
	"mail-templates/password-reset.html",
}

func InitMailer() error {
	mailer = gomail.NewDialer(
		config.Get().String("SMTP_HOST"),
		config.Get().Int("SMTP_PORT"),
		config.Get().String("SMTP_USERNAME"),
		config.Get().String("SMTP_PASSWORD"),
	)

	sendCloser, dialError := mailer.Dial()
	if dialError != nil {
		if config.Get().Bool("DEV_MODE") {
			log.Warn().Msg("Error connecting to SMTP server, continuing in dev mode")
		} else {
			log.Fatal().Msg("Error connecting to SMTP server")
		}
	} else {
		mailerInitOk = true
		defer sendCloser.Close()
		log.Info().Msg("Connection to SMTP server OK")
	}

	err := loadTemplates()
	if err != nil {
		log.Fatal().Msg("Error loading mail templates")
		return err
	}

	return nil
}

func loadTemplates() error {
	for _, path := range templatePaths {
		t, err := template.ParseFiles(path)
		if err != nil {
			log.Fatal().Msgf("Error parsing mail template at %s", path)
			return err
		}
		name := strings.Split(filepath.Base(path), ".")[0]
		log.Trace().Msgf("Loaded mail template %s", name)
		templates[name] = t
	}
	return nil
}

func GetMailer() *gomail.Dialer {
	log.Trace().Msg("Getting mailer")
	// only allow for mailer initialization at runtime in dev mode
	// TODO check self-healing and degration of mailer
	if !mailerInitOk && devMode {
		log.Warn().Msg("Initializing mailer at runtime")
		InitMailer()
	}
	return mailer
}

func GetTemplate(name string) *template.Template {
	return templates[name]
}
