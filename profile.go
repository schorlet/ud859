package ud859

import (
	"golang.org/x/net/context"

	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/user"
)

// profileKey returns a datastore key for the identified user, or nil.
func profileKey(c context.Context) *datastore.Key {
	if u := user.Current(c); u != nil {
		return datastore.NewKey(c, "Profile", u.String(), 0, nil)
	}
	return nil
}

// GetProfile returns the profile associated with the current user.
func (ConferenceAPI) GetProfile(c context.Context) (*Profile, error) {
	key := profileKey(c)
	if key == nil {
		return nil, ErrUnauthorized
	}

	// get the profile
	profile := new(Profile)
	err := datastore.Get(c, key, profile)

	if err == datastore.ErrNoSuchEntity {
		profile.Email = user.Current(c).Email
		err = nil
	} else if err != nil {
		profile = nil
	}
	return profile, err
}

// SaveProfile creates or updates the profile associated with the current user.
func (ConferenceAPI) SaveProfile(c context.Context, form *ProfileForm) error {
	key := profileKey(c)
	if key == nil {
		return ErrUnauthorized
	}

	// set the form values
	profile := &Profile{
		Email:        user.Current(c).Email,
		DisplayName:  form.DisplayName,
		TeeShirtSize: form.TeeShirtSize,
	}

	_, err := datastore.Put(c, key, profile)
	return err
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

// GotoConference performs the registration to the specified conference.
func (api *ConferenceAPI) GotoConference(c context.Context, form *ConferenceKeyForm) error {
	// get the profile
	profile, err := api.GetProfile(c)
	if err != nil {
		return err
	}

	// get the conference
	conference, err := api.GetConference(c, form)
	if err != nil {
		return errBadRequest(err, "conference does not exist")
	}

	if profile.HasRegistered(conference.ID) {
		return ErrRegistered
	}
	if conference.SeatsAvailable <= 0 {
		return ErrNoSeatsAvailable
	}

	return datastore.RunInTransaction(c, func(c context.Context) error {
		// register to the conference
		profile.register(conference.ID)
		_, err = datastore.Put(c, profileKey(c), profile)
		if err != nil {
			return err
		}

		// decrease the available seats
		conference.SeatsAvailable--
		_, err = datastore.Put(c, conferenceKey(c, conference.ID), conference)
		return err
	}, nil)
}

// CancelConference cancels the registration to the specified conference.
func (api *ConferenceAPI) CancelConference(c context.Context, form *ConferenceKeyForm) error {
	// get the profile
	profile, err := api.GetProfile(c)
	if err != nil {
		return err
	}

	// get the conference
	conference, err := api.GetConference(c, form)
	if err != nil {
		return errBadRequest(err, "conference does not exist")
	}

	if !profile.HasRegistered(conference.ID) {
		return ErrNotRegistered
	}

	return datastore.RunInTransaction(c, func(c context.Context) error {
		// unregister from the conference
		profile.unRegister(conference.ID)
		_, err := datastore.Put(c, profileKey(c), profile)
		if err != nil {
			return err
		}

		// increase the available seats
		conference.SeatsAvailable++
		_, err = datastore.Put(c, conferenceKey(c, conference.ID), conference)
		return err
	}, nil)
}
