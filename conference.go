package ud859

import (
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/context"

	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/user"
)

// conferenceKey returns the datastore key associated with the specified conference ID.
func conferenceKey(c context.Context, conferenceID int64) *datastore.Key {
	return datastore.NewKey(c, "Conference", "", conferenceID, profileKey(c))
}

// GetConference returns the Conference with the specified key.
func (ConferenceAPI) GetConference(c context.Context, form *ConferenceKeyForm) (*Conference, error) {
	key, err := datastore.DecodeKey(form.WebsafeKey)
	if err != nil {
		return nil, errBadRequest(err, "invalid conference key")
	}

	// get the conference
	conference := new(Conference)
	err = datastore.Get(c, key, conference)
	if err == datastore.ErrNoSuchEntity {
		return nil, errNotFound(err, "conference not found")
	} else if err != nil {
		return nil, err
	}

	conference.ID = key.IntID()
	conference.WebsafeKey = key.Encode()

	return conference, nil
}

// CreateConference creates a Conference with the specified form.
func (ConferenceAPI) CreateConference(c context.Context, form *ConferenceForm) (*ConferenceKeyForm, error) {
	if u := user.Current(c); u == nil {
		return nil, ErrUnauthorized
	}

	// create a new conference
	conference, err := FromConferenceForm(form)
	if err != nil {
		return nil, err
	}

	// incomplete conference key
	key := conferenceKey(c, 0)

	err = datastore.RunInTransaction(c, func(c context.Context) error {
		// save the conference
		key, err := datastore.Put(c, key, conference)
		if err != nil {
			return err
		}
		conference.ID = key.IntID()
		conference.WebsafeKey = key.Encode()

		// create task ...
		return nil
	}, nil)

	if err != nil {
		return nil, err
	}

	return &ConferenceKeyForm{conference.WebsafeKey}, nil
}

// FromConferenceForm returns a new Conference from the specified ConferenceForm.
func FromConferenceForm(form *ConferenceForm) (*Conference, error) {
	conference := new(Conference)

	conference.Name = form.Name
	conference.Description = form.Description

	if form.Topics != "" {
		conference.Topics = strings.Split(form.Topics, ",")
	}
	conference.City = form.City

	if form.StartDate != "" {
		startDate, err := time.Parse(TimeFormat, form.StartDate)
		if err != nil {
			return nil, errBadRequest(err, "unable to parse start date")
		}
		conference.StartDate = startDate
	}

	if form.EndDate != "" {
		endDate, err := time.Parse(TimeFormat, form.EndDate)
		if err != nil {
			return nil, errBadRequest(err, "unable to parse end date")
		}
		conference.EndDate = endDate
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
