package ud859

import (
	"encoding/json"
	"time"

	"github.com/GoogleCloudPlatform/go-endpoints/endpoints"
)

// TimeFormat to use with ConferenceForm.
const TimeFormat = "2006-01-02"

// Supported query operators.
const (
	EQ  string = "="
	LT         = "<"
	GT         = ">"
	LTE        = "<="
	GTE        = ">="
)

// Query fields.
const (
	Name           string = "Name"
	City                  = "City"
	Topics                = "Topics"
	StartDate             = "StartDate"
	EndDate               = "EndDate"
	Month                 = "Month"
	MaxAttendees          = "MaxAttendees"
	SeatsAvailable        = "SeatsAvailable"
)

// TeeShirt sizes.
const (
	SizeNO   string = ""
	SizeXS          = "XS"
	SizeS           = "S"
	SizeM           = "M"
	SizeL           = "L"
	SizeXL          = "XL"
	SizeXXL         = "XXL"
	SizeXXXL        = "XXXL"
)

// Common errors.
var (
	ErrUnauthorized     = endpoints.NewUnauthorizedError("ud859: authorization required")
	ErrRegistered       = endpoints.NewConflictError("ud859: already registered")
	ErrNotRegistered    = endpoints.NewConflictError("ud859: not registered")
	ErrNoSeatsAvailable = endpoints.NewConflictError("ud859: no seats available")
)

type (
	// ConferenceAPI is the conference API.
	ConferenceAPI struct{}

	// Conference gives details about a conference.
	Conference struct {
		ID             int64     `json:"-" datastore:"-"`
		WebsafeKey     string    `json:"websafeKey" datastore:"-"`
		Name           string    `json:"name"`
		Description    string    `json:"description"`
		Organizer      string    `json:"organizerDisplayName"`
		Topics         []string  `json:"topics"`
		City           string    `json:"city"`
		StartDate      time.Time `json:"startDate"`
		EndDate        time.Time `json:"endDate"`
		MaxAttendees   int       `json:"maxAttendees"`
		SeatsAvailable int       `json:"seatsAvailable"`
	}

	// Conferences is a list of Conferences.
	Conferences struct {
		Items []*Conference `json:"items"`
	}

	// ConferenceForm gives details about a conference to create.
	ConferenceForm struct {
		Name         string   `json:"name" endpoints:"req"`
		Description  string   `json:"description"`
		Topics       []string `json:"topics"`
		City         string   `json:"city"`
		StartDate    string   `json:"startDate"`
		EndDate      string   `json:"endDate"`
		MaxAttendees string   `json:"maxAttendees"`
	}

	// ConferenceKeyForm is a conference public key.
	ConferenceKeyForm struct {
		WebsafeKey string `json:"websafeConferenceKey" endpoints:"req"`
	}

	// ConferenceQueryForm collects filters for searching for Conferences.
	ConferenceQueryForm struct {
		Filters          []*Filter `json:"filters"`
		inequalityFilter *Filter
	}

	// Profile gives details about an identified user.
	Profile struct {
		Email        string  `json:"-"`
		DisplayName  string  `json:"displayName"`
		TeeShirtSize string  `json:"teeShirtSize"`
		Conferences  []int64 `json:"-"`
	}

	// ProfileForm gives details about a profile to create or update.
	ProfileForm struct {
		DisplayName  string `json:"displayName"`
		TeeShirtSize string `json:"teeShirtSize"`
	}

	// Announcement is an announcement.
	Announcement struct {
		Message string `json:"message"`
	}

	// Filter describes a query restriction.
	Filter struct {
		Field string      `endpoints:"req"`
		Op    string      `endpoints:"req"`
		Value interface{} `endpoints:"req"`
	}
)

// MarshalJSON returns *f as the JSON encoding of f.
func (f *Filter) MarshalJSON() (b []byte, err error) {
	m := make(map[string]interface{})
	m["field"] = f.Field
	m["operator"] = f.Op
	m["value"] = f.Value
	return json.Marshal(m)
}

// UnmarshalJSON sets *f to a copy of data.
func (f *Filter) UnmarshalJSON(data []byte) error {
	m := make(map[string]interface{})
	err := json.Unmarshal(data, &m)
	if err != nil {
		return errBadRequest(err, "unable to parse filter")
	}

	f.Field = m["field"].(string)
	f.Op = m["operator"].(string)
	f.Value = m["value"]

	if f.Field == MaxAttendees || f.Field == SeatsAvailable {
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

func errBadRequest(cause error, message string) error {
	return endpoints.NewBadRequestError("ud859: %s (%v)", message, cause)
}

func errNotFound(cause error, message string) error {
	return endpoints.NewNotFoundError("ud859: %s (%v)", message, cause)
}
