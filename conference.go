package ud859

import (
	"bytes"
	"html/template"
	"strconv"
	"time"

	"golang.org/x/net/context"

	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
)

// Conference defines a conference.
type Conference struct {
	WebsafeKey     string    `json:"websafeKey" datastore:"-"`
	Name           string    `json:"name" datastore:",noindex"`
	Description    string    `json:"description" datastore:",noindex"`
	Organizer      string    `json:"organizerDisplayName" datastore:",noindex"`
	Topics         []string  `json:"topics" datastore:",noindex"`
	City           string    `json:"city" datastore:",noindex"`
	StartDate      time.Time `json:"startDate" datastore:"START_DATE"`
	EndDate        time.Time `json:"endDate" datastore:",noindex"`
	Month          int       `json:"-" datastore:",noindex"`
	MaxAttendees   int       `json:"maxAttendees" datastore:",noindex"`
	SeatsAvailable int       `json:"seatsAvailable" datastore:",noindex"`
}

// Conferences is a list of Conferences.
type Conferences struct {
	Items []*Conference `json:"items"`
}

func (c Conferences) Len() int {
	return len(c.Items)
}
func (c Conferences) Swap(i, j int) {
	c.Items[i], c.Items[j] = c.Items[j], c.Items[i]
}
func (c Conferences) Less(i, j int) bool {
	c1, c2 := c.Items[i], c.Items[j]
	return c1.StartDate.Before(c2.StartDate)
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

// ConferenceKeyForm wraps a conference websafeKey.
type ConferenceKeyForm struct {
	WebsafeKey string `json:"websafeConferenceKey" endpoints:"req"`
}

// ConferenceCreated is returned when a conference is created.
type ConferenceCreated struct {
	Name       string `json:"name"`
	WebsafeKey string `json:"websafeConferenceKey"`
}

// GetConference returns the Conference with the specified ConferenceKeyForm.
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

// CreateConference creates a Conference in the datastore from the specified ConferenceForm.
func (ConferenceAPI) CreateConference(c context.Context, form *ConferenceForm) (*ConferenceCreated, error) {
	pid, err := profileID(c)
	if err != nil {
		return nil, err
	}

	// create a new conference
	conference, err := fromConferenceForm(form)
	if err != nil {
		return nil, err
	}

	// get the profile
	profile, err := getProfile(c, pid)
	if err != nil {
		return nil, err
	}
	conference.Organizer = profile.DisplayName

	// incomplete conference key
	ckey := datastore.NewIncompleteKey(c, "Conference", pid.key)

	err = datastore.RunInTransaction(c, func(c context.Context) error {
		// save the conference
		key, err := datastore.Put(c, ckey, conference)
		if err != nil {
			return errInternalServer(err, "unable to create conference")
		}
		conference.WebsafeKey = key.Encode()

		// create indexation task
		err = indexConference(c, conference)
		if err != nil {
			return errInternalServer(err, "unable to index conference")
		}
		return nil
	}, nil)

	if err != nil {
		return nil, err
	}

	// body of the confirmation email
	body, err := conferenceText(conference)
	if err != nil {
		log.Errorf(c, "unable to create conference email: %v", err)
	} else {
		// create confirmation task
		err = sendConfirmation(c, profile.Email, body)
		if err != nil {
			log.Errorf(c, "unable to send conference email: %v", err)
		}
	}

	// clear cache
	err = deleteCacheNoFilters.Call(c)
	if err != nil {
		log.Errorf(c, "unable to clear cache: %v", err)
	}

	return &ConferenceCreated{
		Name:       conference.Name,
		WebsafeKey: conference.WebsafeKey,
	}, nil
}

// fromConferenceForm creates a new Conference from a ConferenceForm.
func fromConferenceForm(form *ConferenceForm) (*Conference, error) {
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
