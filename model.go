package ud859

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/GoogleCloudPlatform/go-endpoints/endpoints"
)

// Supported query operators.
const (
	EQ  = "="
	LT  = "<"
	GT  = ">"
	LTE = "<="
	GTE = ">="
)

// Query fields.
const (
	Name           = "NAME"
	City           = "CITY"
	Topics         = "TOPIC"
	StartDate      = "START_DATE"
	EndDate        = "END_DATE"
	Month          = "MONTH"
	MaxAttendees   = "MAX_ATTENDEES"
	SeatsAvailable = "SEATS_AVAILABLE"
)

// Common errors.
var (
	ErrUnauthorized     = endpoints.NewUnauthorizedError("ud859: authorization required")
	ErrRegistered       = endpoints.NewConflictError("ud859: already registered")
	ErrNotRegistered    = endpoints.NewConflictError("ud859: not registered")
	ErrNoSeatsAvailable = endpoints.NewConflictError("ud859: no seats available")
)

func errBadRequest(cause error, message string) error {
	return endpoints.NewBadRequestError("ud859: %s (%v)", message, cause)
}

func errNotFound(cause error, message string) error {
	return endpoints.NewNotFoundError("ud859: %s (%v)", message, cause)
}

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
	f.Value = m["value"]

	f.Op = m["operator"].(string)
	switch f.Op {
	case "EQ":
		f.Op = EQ
	case "LT":
		f.Op = LT
	case "GT":
		f.Op = GT
	case "LTEQ":
		f.Op = LTE
	case "GTEQ":
		f.Op = GTE
	}

	if f.Field == Month || f.Field == MaxAttendees ||
		f.Field == SeatsAvailable {
		switch v := f.Value.(type) {
		case string:
			f.Value, err = strconv.Atoi(v)
			if err != nil {
				return errBadRequest(err, "unable to parse "+f.Field)
			}
		case float64:
			f.Value = int(v)
		default:
			return errBadRequest(err, "unable to parse "+f.Field)
		}

	} else if f.Field == StartDate || f.Field == EndDate {
		switch v := f.Value.(type) {
		case time.Time:
		case string:
			f.Value, err = time.Parse(time.RFC3339, v)
			if err != nil {
				return errBadRequest(err, "unable to parse "+f.Field)
			}
		default:
			return errBadRequest(err, "unable to parse "+f.Field)
		}
	}
	return nil
}
