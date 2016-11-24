package ud859

import (
	"strconv"
	"time"

	"golang.org/x/net/context"

	"google.golang.org/appengine/datastore"
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
	pkey, err := profileKey(c)
	if err != nil {
		return nil, err
	}

	// create a new conference
	conference, err := FromConferenceForm(form)
	if err != nil {
		return nil, err
	}

	// incomplete conference key
	ckey := conferenceKey(c, 0, pkey)

	err = datastore.RunInTransaction(c, func(c context.Context) error {
		// save the conference
		key, err := datastore.Put(c, ckey, conference)
		if err != nil {
			return err
		}
		conference.WebsafeKey = key.Encode()

		// create task ...
		return nil
	}, nil)

	if err != nil {
		return nil, err
	}

	return &ConferenceCreated{
		Name:       conference.Name,
		WebsafeKey: conference.WebsafeKey,
	}, nil
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
