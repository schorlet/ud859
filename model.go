package ud859

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"google.golang.org/appengine/datastore"
)

const TimeFormat = "2006-01-02"

const (
	EQ  string = "="
	LT         = "<"
	GT         = ">"
	LTE        = "<="
	GTE        = ">="
)

const (
	Name         string = "Name"
	City                = "City"
	Topics              = "Topics"
	StartDate           = "StartDate"
	EndDate             = "EndDate"
	Month               = "Month"
	MaxAttendees        = "MaxAttendees"
)

const (
	SIZE_NO int8 = iota
	SIZE_XS
	SIZE_S
	SIZE_M
	SIZE_L
	SIZE_XL
	SIZE_XXL
	SIZE_XXXL
)

type (
	Conference struct {
		ID             int64     `json:"-" datastore:"-"`
		WebsafeKey     string    `json:"websafeKey" datastore:"-"`
		Name           string    `json:"name"`
		Description    string    `json:"description"`
		Organizer      string    `json:"organizer"`
		Topics         []string  `json:"topics"`
		City           string    `json:"city"`
		StartDate      time.Time `json:"startDate"`
		EndDate        time.Time `json:"endDate"`
		MaxAttendees   int       `json:"maxAttendees"`
		SeatsAvailable int       `json:"seatsAvailable"`
	}
	// "month": 9,
	// "organizerDisplayName": "DawoonC",
	// "kind": "conference#resourcesItem"

	Conferences struct {
		Items []*Conference `json:"items"`
	}

	ConferenceForm struct {
		Name         string `json:"name"`
		Description  string `json:"description"`
		Topics       string `json:"topics"`
		City         string `json:"city"`
		StartDate    string `json:"startDate"`
		EndDate      string `json:"endDate"`
		MaxAttendees string `json:"maxAttendees"`
	}

	ConferenceQueryForm struct {
		Filters          []*filter `json:"filters"`
		inequalityFilter *filter
	}

	Profile struct {
		Email        string  `json:"-"`
		DisplayName  string  `json:"displayName"`
		TeeShirtSize int8    `json:"teeShirtSize"`
		Conferences  []int64 `json:"-"`
	}

	ProfileForm struct {
		DisplayName  string `json:"displayName"`
		TeeShirtSize int8   `json:"teeShirtSize"`
	}

	Announcement struct {
		Message string `json:"message"`
	}

	filter struct {
		Field string      `json:"field"`
		Op    string      `json:"op"`
		Value interface{} `json:"value"`
	}

	statusError struct {
		Cause   error
		Message string
		Status  int
	}
)

func (f *filter) UnmarshalJSON(data []byte) error {
	m := make(map[string]interface{})
	err := json.Unmarshal(data, &m)
	if err != nil {
		return errBadRequest(err, "unable to parse filter")
	}
	if len(m) != 3 {
		return errBadRequest(nil, "unable to parse filter")
	}

	f.Field = m["field"].(string)
	f.Op = m["op"].(string)
	f.Value = m["value"]

	if f.Field == MaxAttendees {
		f.Value = int(f.Value.(float64))

	} else if f.Field == StartDate || f.Field == EndDate {
		switch v := f.Value.(type) {
		case time.Time:
		case string:
			f.Value, err = time.Parse(time.RFC3339, v)
			if err != nil {
				f.Value, err = time.Parse(TimeFormat, v)
				if err != nil {
					return errBadRequest(err, "unable to parse "+f.Field)
				}
			}
		default:
			return errBadRequest(err, "unable to parse "+f.Field)
		}
	}
	return nil
}

var (
	ErrUnauthorized     = newErrorCode("authorization required", http.StatusUnauthorized)
	ErrRegistered       = newError("already registered")
	ErrNotRegistered    = newError("not registered")
	ErrNoSeatsAvailable = newError("no seats available")
)

func newError(message string) *statusError {
	return newErrorCode(message, http.StatusForbidden)
}

func newErrorCode(message string, code int) *statusError {
	return &statusError{
		Message: message,
		Status:  code,
	}
}

func errBadRequest(cause error, message string) *statusError {
	return &statusError{
		Cause:   cause,
		Message: message,
		Status:  http.StatusBadRequest,
	}
}

func errNotFound(cause error, message string) *statusError {
	return &statusError{
		Cause:   cause,
		Message: message,
		Status:  http.StatusNotFound,
	}
}

func (e statusError) Error() string {
	return fmt.Sprintf("ud859: %s (%v)", e.Message, e.Cause)
}

func (q *ConferenceQueryForm) Filter(field string, op string, value interface{}) *ConferenceQueryForm {
	q.Filters = append(q.Filters, &filter{field, op, value})
	return q
}

func (q *ConferenceQueryForm) CheckFilters() error {
	var found bool

	// verify that the inequality filter applys only on the same field
	for _, filter := range q.Filters {
		if filter.Op != EQ {
			if found && filter.Field != q.inequalityFilter.Field {
				return newError("only one inequality filter is allowed")
			}

			found = true
			q.inequalityFilter = filter
		}
	}
	return nil
}

func (q ConferenceQueryForm) String() string {
	s := "query: "
	for _, filter := range q.Filters {
		s += fmt.Sprintf("[%s %s %v]", filter.Field, filter.Op, filter.Value)
	}
	return s
}

func (q ConferenceQueryForm) Query() (*datastore.Query, error) {
	// log.Printf("%s", q)
	err := q.CheckFilters()
	if err != nil {
		return nil, err
	}

	query := datastore.NewQuery("Conference")

	if q.inequalityFilter != nil {
		// order first by the inequality filter
		query = query.Order(string(q.inequalityFilter.Field))
	}
	query = query.Order("Name")

	for _, filter := range q.Filters {
		query = query.Filter(
			fmt.Sprintf("%s %s", filter.Field, filter.Op), filter.Value)
	}

	return query, nil
}

func FromConferenceForm(form *ConferenceForm) (*Conference, error) {
	conference := new(Conference)

	conference.Name = form.Name
	conference.Description = form.Description
	conference.Topics = strings.Split(form.Topics, ",")
	conference.City = form.City

	startDate, err := time.Parse(TimeFormat, form.StartDate)
	if err != nil {
		return nil, errBadRequest(err, "unable to parse start date")
	}
	conference.StartDate = startDate

	endDate, err := time.Parse(TimeFormat, form.EndDate)
	if err != nil {
		return nil, errBadRequest(err, "unable to parse end date")
	}
	conference.EndDate = endDate

	max, err := strconv.Atoi(form.MaxAttendees)
	if err != nil {
		return nil, errBadRequest(err, "unable to parse max attendees")
	}
	conference.MaxAttendees = max
	conference.SeatsAvailable = max

	return conference, nil
}

func (p *Profile) Goto(conferenceID int64) {
	p.Conferences = append(p.Conferences, conferenceID)
}

func (p *Profile) Cancel(conferenceID int64) {
	for i, id := range p.Conferences {
		if id == conferenceID {
			p.Conferences = append(p.Conferences[:i], p.Conferences[i+1:]...)
			break
		}
	}
}

func (p Profile) Registered(conferenceID int64) bool {
	for _, id := range p.Conferences {
		if id == conferenceID {
			return true
		}
	}
	return false
}
