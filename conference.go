package ud859

import (
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

// QueryConferences searches for Conferences with the specified filters.
func (ConferenceAPI) QueryConferences(c context.Context, form *ConferenceQueryForm) (*Conferences, error) {
	query, err := form.Query()
	if err != nil {
		return nil, err
	}

	conferences := make([]*Conference, 0)
	keys, err := query.GetAll(c, &conferences)
	if err != nil {
		return nil, err
	}

	for i := 0; i < len(conferences); i++ {
		conferences[i].ID = keys[i].IntID()
		conferences[i].WebsafeKey = keys[i].Encode()
	}

	return &Conferences{Items: conferences}, nil
}
