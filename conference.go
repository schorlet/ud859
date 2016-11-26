package ud859

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"golang.org/x/net/context"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/mail"
	"google.golang.org/appengine/taskqueue"
)

// conferenceKey returns the datastore key associated with the specified conference ID.
func conferenceKey(c context.Context, conferenceID int64, pkey *datastore.Key) *datastore.Key {
	return datastore.NewKey(c, "Conference", "", conferenceID, pkey)
}

// GetConference returns the Conference with the specified key.
func (ConferenceAPI) GetConference(c context.Context, form *ConferenceKeyForm) (*Conference, error) {
	key, err := datastore.DecodeKey(form.WebsafeKey)
	if err != nil {
		return nil, errBadRequest(err, "invalid conference key")
	}
	return getConference(c, key)
}

func getConference(c context.Context, key *datastore.Key) (*Conference, error) {
	// get the conference
	conference := new(Conference)
	err := datastore.Get(c, key, conference)
	if err != nil {
		return nil, errNotFound(err, "conference not found")
	}

	conference.WebsafeKey = key.Encode()
	return conference, nil
}

// CreateConference creates a Conference with the specified form.
func (ConferenceAPI) CreateConference(c context.Context, form *ConferenceForm) (*ConferenceCreated, error) {
	pid, err := profileID(c)
	if err != nil {
		return nil, err
	}

	// create a new conference
	conference, err := FromConferenceForm(form)
	if err != nil {
		return nil, err
	}

	// get the profile
	profile, err := getProfile(c, pid)
	if err != nil {
		return nil, err
	}
	conference.Organizer = profile.DisplayName

	// conference info
	info, err := conferenceInfo(conference)
	if err != nil {
		return nil, err
	}

	// incomplete conference key
	ckey := conferenceKey(c, 0, pid.key)

	err = datastore.RunInTransaction(c, func(c context.Context) error {
		// save the conference
		key, err := datastore.Put(c, ckey, conference)
		if err != nil {
			return err
		}
		conference.WebsafeKey = key.Encode()

		// create confirmation task
		task := taskqueue.NewPOSTTask("/tasks/send_confirmation_email",
			url.Values{
				"email": {profile.Email},
				"info":  {info},
			})
		_, err = taskqueue.Add(c, task, "")
		return err
	}, nil)

	if err != nil {
		return nil, err
	}

	return &ConferenceCreated{
		Name:       conference.Name,
		WebsafeKey: conference.WebsafeKey,
	}, nil
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

// FromConferenceForm returns a new Conference from the specified ConferenceForm.
func FromConferenceForm(form *ConferenceForm) (*Conference, error) {
	conference := new(Conference)

	conference.Name = form.Name
	conference.Description = form.Description
	conference.Topics = form.Topics
	conference.City = form.City

	if form.EndDate != "" {
		endDate, err := time.Parse(time.RFC3339, form.EndDate)
		if err != nil {
			return nil, errBadRequest(err, "unable to parse end date")
		}
		conference.EndDate = endDate
		conference.Month = int(endDate.Month())
	}

	if form.StartDate != "" {
		startDate, err := time.Parse(time.RFC3339, form.StartDate)
		if err != nil {
			return nil, errBadRequest(err, "unable to parse start date")
		}
		conference.StartDate = startDate
		conference.Month = int(startDate.Month())
	}

	if form.MaxAttendees != "" {
		max, err := strconv.Atoi(form.MaxAttendees)
		if err != nil {
			return nil, errBadRequest(err, "unable to parse max attendees")
		}
		conference.MaxAttendees = max
		conference.SeatsAvailable = max
	}

	return conference, nil
}
