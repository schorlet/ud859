package ud859

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"golang.org/x/net/context"

	"google.golang.org/appengine/delay"
	"google.golang.org/appengine/search"
)

// ConferenceDoc is an indexed conference.
type ConferenceDoc struct {
	WebsafeKey     search.Atom `json:"websafeKey" search:"KEY"`
	Name           string      `json:"name" search:"NAME"`
	Description    string      `json:"description" search:"DESCRIPTION"`
	Topics         string      `json:"topics" search:"TOPIC"`
	City           string      `json:"city" search:"CITY"`
	StartDate      time.Time   `json:"startDate" search:"START_DATE"`
	EndDate        time.Time   `json:"endDate" search:"END_DATE"`
	Month          float64     `json:"-" search:"MONTH"`
	MaxAttendees   float64     `json:"maxAttendees" search:"MAX_ATTENDEES"`
	SeatsAvailable float64     `json:"seatsAvailable" search:"SEATS_AVAILABLE"`
}

// FromConference creates a ConferenceDoc from a Conference.
func FromConference(conference *Conference) *ConferenceDoc {
	return &ConferenceDoc{
		WebsafeKey:     search.Atom(conference.WebsafeKey),
		Name:           conference.Name,
		Description:    conference.Description,
		Topics:         strings.Join(conference.Topics, " "),
		City:           conference.City,
		StartDate:      conference.StartDate,
		EndDate:        conference.EndDate,
		Month:          float64(conference.StartDate.Month()),
		MaxAttendees:   float64(conference.MaxAttendees),
		SeatsAvailable: float64(conference.SeatsAvailable),
	}
}

// FromConferenceDoc creates a Conference from a ConferenceDoc.
func FromConferenceDoc(doc *ConferenceDoc) *Conference {
	return &Conference{
		WebsafeKey:     string(doc.WebsafeKey),
		Name:           doc.Name,
		Description:    doc.Description,
		Topics:         strings.Split(doc.Topics, " "),
		City:           doc.City,
		StartDate:      doc.StartDate,
		EndDate:        doc.EndDate,
		Month:          int(doc.StartDate.Month()),
		MaxAttendees:   int(doc.MaxAttendees),
		SeatsAvailable: int(doc.SeatsAvailable),
	}
}

// Query returns the query to apply to the search index.
func (q ConferenceQueryForm) Query() string {
	var str string
	for _, filter := range q.Filters {
		field := filter.Field
		op := filter.Op

		if op == NE {
			field = "NOT " + field
			op = EQ
		}

		switch v := filter.Value.(type) {
		case string:
			str += fmt.Sprintf("%s = (%s) ", field, strings.Map(alphaNumeric, v))
		case time.Time:
			str += fmt.Sprintf("%s %s %s ", field, op, v.Format("2006-01-02"))
		default:
			str += fmt.Sprintf("%s %s %v ", field, op, v)
		}
	}
	return str
}

func alphaNumeric(r rune) rune {
	if unicode.IsLetter(r) || unicode.IsNumber(r) {
		return r
	}
	return ' '
}

// SearchConferences searches for Conferences with the specified filters.
func SearchConferences(c context.Context, form *ConferenceQueryForm) (*Conferences, error) {
	index, err := search.Open("Conference")
	if err != nil {
		return nil, err
	}

	it := index.Search(c, form.Query(), nil)
	conferences := new(Conferences)

	for {
		doc := new(ConferenceDoc)

		_, err := it.Next(doc)
		if err == search.Done {
			break
		} else if err != nil {
			return nil, err
		}

		conference := FromConferenceDoc(doc)
		conferences.Items = append(conferences.Items, conference)
	}

	return conferences, nil
}

func isTesting() bool {
	return clientID == "YOUR-CLIENT-ID"
}

func indexConference(c context.Context, conference *Conference) error {
	if isTesting() {
		// when testing, update the index without delay
		return indexConferenceNow(c, conference)
	}
	return indexConferenceDelay.Call(c, conference)
}

var indexConferenceDelay = delay.Func("index_conference", indexConferenceNow)

func indexConferenceNow(c context.Context, conference *Conference) error {
	index, err := search.Open("Conference")
	if err != nil {
		return err
	}
	doc := FromConference(conference)
	_, err = index.Put(c, conference.WebsafeKey, doc)
	return err
}
