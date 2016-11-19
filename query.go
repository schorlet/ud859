package ud859

import (
	"golang.org/x/net/context"

	"google.golang.org/appengine/datastore"
)

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

// ConferencesCreated returns the Conferences created by the current user.
func (ConferenceAPI) ConferencesCreated(c context.Context) (*Conferences, error) {
	key := profileKey(c)
	if key == nil {
		return nil, ErrUnauthorized
	}

	// get the conferences whose parent is the profile
	conferences := make([]*Conference, 0)
	query := datastore.NewQuery("Conference").Ancestor(key).Order("Name")

	keys, err := query.GetAll(c, &conferences)
	if err != nil {
		return nil, err
	}

	for i := 0; i < len(conferences); i++ {
		conferences[i].ID = keys[i].IntID()
		conferences[i].WebsafeKey = key.Encode()
	}

	return &Conferences{Items: conferences}, nil
}

// ConferencesToAttend returns the Conferences to addend by the current user.
func (api *ConferenceAPI) ConferencesToAttend(c context.Context) (*Conferences, error) {
	// get the profile
	profile, err := api.GetProfile(c)
	if err != nil {
		return nil, err
	}

	if len(profile.Conferences) == 0 {
		conferences := make([]*Conference, 0)
		return &Conferences{Items: conferences}, nil
	}

	// get the conference keys
	keys := make([]*datastore.Key, len(profile.Conferences))
	for i, conferenceID := range profile.Conferences {
		keys[i] = conferenceKey(c, conferenceID)
	}

	// get the conferences
	conferences := make([]*Conference, len(profile.Conferences))
	err = datastore.GetMulti(c, keys, conferences)
	if err != nil {
		return nil, err
	}

	for i := 0; i < len(conferences); i++ {
		conferences[i].ID = keys[i].IntID()
		conferences[i].WebsafeKey = keys[i].Encode()
	}

	return &Conferences{Items: conferences}, nil
}
