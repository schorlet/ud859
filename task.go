package ud859

import (
	"fmt"
	"net/http"
	"net/url"

	"golang.org/x/net/context"

	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/mail"
	"google.golang.org/appengine/taskqueue"
)

func init() {
	http.HandleFunc("/tasks/send_confirmation_email", sendConfirmationEmail)
}

func sendConfirmation(c context.Context, email, body string) error {
	task := taskqueue.NewPOSTTask("/tasks/send_confirmation_email",
		url.Values{
			"email": {email},
			"body":  {body},
		})
	_, err := taskqueue.Add(c, task, "")
	return err
}

// sends an email to the user who has just created a conference.
func sendConfirmationEmail(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	email := r.FormValue("email")
	body := r.FormValue("body")
	if email == "" || body == "" {
		return
	}

	msg := &mail.Message{
		Sender:  fmt.Sprintf("noreply@%s.appspotmail.com", appengine.AppID(c)),
		To:      []string{email},
		Subject: "You created a new Conference!",
		Body:    "Hi, you have created the following conference:\n" + body,
	}

	if err := mail.Send(c, msg); err != nil {
		log.Errorf(c, "could not send email: %v", err)
		http.Error(w, "", http.StatusInternalServerError)
	}
}
