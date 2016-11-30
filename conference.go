package ud859

import (
	"bytes"
	"html/template"
	"net/url"
	"strconv"
	"time"

	"golang.org/x/net/context"

	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/taskqueue"
)

// Conference gives details about a conference.
type Conference struct {
	WebsafeKey     string    `json:"websafeKey" datastore:"-"`
	Name           string    `json:"name" datastore:"NAME"`
	Description    string    `json:"description" datastore:",noindex"`
	Organizer      string    `json:"organizerDisplayName" datastore:",noindex"`
	Topics         []string  `json:"topics" datastore:"TOPIC"`
	City           string    `json:"city" datastore:"CITY"`
	StartDate      time.Time `json:"startDate" datastore:"START_DATE"`
	EndDate        time.Time `json:"endDate" datastore:"END_DATE"`
	Month          int       `json:"-" datastore:"MONTH"`
	MaxAttendees   int       `json:"maxAttendees" datastore:"MAX_ATTENDEES"`
	SeatsAvailable int       `json:"seatsAvailable" datastore:"SEATS_AVAILABLE"`
}

// Conferences is a list of Conferences.
type Conferences struct {
	Items []*Conference `json:"items"`
}

// ConferenceForm gives details about a conference to create.
type ConferenceForm struct {
	Name         string   `json:"name" endpoints:"req"`
	Description  string   `json:"description"`
	Topics       []string `json:"topics"`
	City         string   `json:"city"`
	StartDate    string   `json:"startDate"`
	EndDate      string   `json:"endDate"`
	MaxAttendees string   `json:"maxAttendees"`
}

// ConferenceKeyForm is a conference public key.
type ConferenceKeyForm struct {
	WebsafeKey string `json:"websafeConferenceKey" endpoints:"req"`
}

// ConferenceCreated gives details about the created conference.
type ConferenceCreated struct {
	Name       string `json:"name"`
	WebsafeKey string `json:"websafeConferenceKey"`
}

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
	info, err := conferenceText(conference)
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

	// clear cache
	_ = deleteCacheNoFilters.Call(c)

	return &ConferenceCreated{
		Name:       conference.Name,
		WebsafeKey: conference.WebsafeKey,
	}, nil
}

// FromConferenceForm returns a new Conference from the specified ConferenceForm.
func FromConferenceForm(form *ConferenceForm) (*Conference, error) {
	var (
		err       error
		startDate time.Time
		endDate   time.Time
		month     int
		attendees int
	)

	if form.StartDate != "" {
		startDate, err = time.Parse(time.RFC3339, form.StartDate)
		if err != nil {
			return nil, errBadRequest(err, "unable to parse start date")
		}
		month = int(startDate.Month())
	}

	if form.EndDate != "" {
		endDate, err = time.Parse(time.RFC3339, form.EndDate)
		if err != nil {
			return nil, errBadRequest(err, "unable to parse end date")
		}
	}

	if form.MaxAttendees != "" {
		attendees, err = strconv.Atoi(form.MaxAttendees)
		if err != nil {
			return nil, errBadRequest(err, "unable to parse max attendees")
		}
	}

	return &Conference{
		Name:           form.Name,
		Description:    form.Description,
		Topics:         form.Topics,
		City:           form.City,
		StartDate:      startDate,
		EndDate:        endDate,
		Month:          month,
		MaxAttendees:   attendees,
		SeatsAvailable: attendees,
	}, nil
}

var conferenceTemplate = template.Must(template.New("text").Parse(`
	Name: {{.Name}}
	Description: {{.Description}}
	Topics: {{.Topics}}
	City: {{.City}}
	StartDate: {{.StartDate}}
	EndDate: {{.EndDate}}
	MaxAttendees: {{.MaxAttendees}}
`))

func conferenceText(c *Conference) (string, error) {
	buf := new(bytes.Buffer)
	err := conferenceTemplate.Execute(buf, c)
	return buf.String(), err
}
