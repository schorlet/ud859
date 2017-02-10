package ud859

import (
	"encoding/json"
	"fmt"
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
	errParse := func(err error) error {
		message := fmt.Sprintf("unable to parse filter: %s", data)
		return errBadRequest(err, message)
	}

	m := make(map[string]interface{})
	err := json.Unmarshal(data, &m)
	if err != nil {
		return errParse(err)
	}

	var ok bool
	f.Field, ok = m["field"].(string)
	if !ok {
		return errParse(err)
	}

	f.Op, ok = m["operator"].(string)
	if !ok {
		return errParse(err)
	}
	if err = f.setOp(); err != nil {
		return errParse(err)
	}

	f.Value = m["value"]
	if f.Field == Month || f.Field == MaxAttendees || f.Field == SeatsAvailable {
		err = f.setValueInt()

	} else if f.Field == StartDate || f.Field == EndDate {
		err = f.setValueTime()
	}
	if err != nil {
		return errParse(err)
	}
	return nil
}

func (f *Filter) setOp() (err error) {
	switch f.Op {
	case EQ:
	case LT:
	case GT:
	case LTE:
	case GTE:
	case NE:
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
	default:
		return fmt.Errorf("invalid operator")
	}
	return nil
}

func (f *Filter) setValueInt() (err error) {
	switch v := f.Value.(type) {
	case string:
		f.Value, err = strconv.Atoi(v)
		if err != nil {
			err = fmt.Errorf("invalid value: %v", err)
		}
	case float64:
		f.Value = int(v)
	default:
		err = fmt.Errorf("invalid type of value")
	}
	return
}

func (f *Filter) setValueTime() (err error) {
	switch v := f.Value.(type) {
	case time.Time:
	case string:
		f.Value, err = time.Parse(time.RFC3339, v)
		if err != nil {
			err = fmt.Errorf("invalid value: %v", err)
		}
	default:
		err = fmt.Errorf("invalid type of value")
	}
	return
}
