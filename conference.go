package ud859

import (
	"golang.org/x/net/context"

	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/user"
)

func conferenceKey(c context.Context, conferenceID int64) *datastore.Key {
	return datastore.NewKey(c, "Conference", "", conferenceID, profileKey(c))
}

func conferenceSafeKey(c context.Context, websafeKey string) (*datastore.Key, error) {
	return datastore.DecodeKey(websafeKey)
}

func GetConference(c context.Context, websafeKey string) (*Conference, error) {
	key, err := conferenceSafeKey(c, websafeKey)
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

func CreateConference(c context.Context, form *ConferenceForm) (*Conference, error) {
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

	return conference, err
}

func QueryConferences(c context.Context) ([]*Conference, error) {
	return QueryConferencesFilter(c, new(ConferenceQueryForm))
}

func QueryConferencesFilter(c context.Context, form *ConferenceQueryForm) ([]*Conference, error) {
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

	return conferences, nil
}
