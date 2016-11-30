package ud859

import (
	"fmt"
	"net/http"

	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/mail"
)

func init() {
	http.HandleFunc("/tasks/send_confirmation_email", sendConfirmationEmail)
}

func sendConfirmationEmail(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	email := r.FormValue("email")
	info := r.FormValue("info")
	if email == "" || info == "" {
		return
	}

	msg := &mail.Message{
		Sender:  fmt.Sprintf("noreply@%s.appspotmail.com", appengine.AppID(c)),
		To:      []string{email},
		Subject: "You created a new Conference!",
		Body:    "Hi, you have created the following conference:\n" + info,
	}

	if err := mail.Send(c, msg); err != nil {
		log.Errorf(c, "could not send email: %v", err)
		http.Error(w, "", http.StatusInternalServerError)
	}
}
