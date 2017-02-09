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
	NE  = "!="
)

// Conference query fields.
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

func errConflict(message string) error {
	return endpoints.NewConflictError("ud859: %s", message)
}

func errInternalServer(cause error, message string) error {
	return endpoints.NewInternalServerError("ud859: %s (%v)", message, cause)
}

func errUnauthorized(cause error, message string) error {
	return endpoints.NewUnauthorizedError("ud859: %s (%v)", message, cause)
}

func errBadRequest(cause error, message string) error {
	return endpoints.NewBadRequestError("ud859: %s (%v)", message, cause)
}

func errNotFound(cause error, message string) error {
	return endpoints.NewNotFoundError("ud859: %s (%v)", message, cause)
}

// MarshalJSON marshals the Filter as JSON data.
func (f *Filter) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})
	m["field"] = f.Field
	m["operator"] = f.Op
	m["value"] = f.Value
	return json.Marshal(m)
}

// UnmarshalJSON unmarshals the JSON data into the Filter.
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
	case "NE":
		f.Op = NE
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
