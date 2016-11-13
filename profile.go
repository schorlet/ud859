package ud859

import (
	"golang.org/x/net/context"

	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/user"
)

func profileKey(c context.Context) *datastore.Key {
	if u := user.Current(c); u != nil {
		return datastore.NewKey(c, "Profile", u.String(), 0, nil)
	}
	return nil
}

func GetProfile(c context.Context) (*Profile, error) {
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

func SaveProfile(c context.Context, form *ProfileForm) error {
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

func ConferencesCreated(c context.Context) ([]*Conference, error) {
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

	return conferences, nil
}

func ConferencesToAttend(c context.Context) ([]*Conference, error) {
	// get the profile
	profile, err := GetProfile(c)
	if err != nil {
		return nil, err
	}

	if len(profile.Conferences) == 0 {
		conferences := make([]*Conference, 0)
		return conferences, nil
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

	return conferences, nil
}

func GotoConference(c context.Context, websafeKey string) error {
	// get the profile
	profile, err := GetProfile(c)
	if err != nil {
		return err
	}

	// get the conference
	conference, err := GetConference(c, websafeKey)
	if err != nil {
		return errBadRequest(err, "conference does not exist")
	}

	if profile.Registered(conference.ID) {
		return ErrRegistered
	}
	if conference.SeatsAvailable <= 0 {
		return ErrNoSeatsAvailable
	}

	return datastore.RunInTransaction(c, func(c context.Context) error {
		// register to the conference
		profile.Goto(conference.ID)
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

func CancelConference(c context.Context, websafeKey string) error {
	// get the profile
	profile, err := GetProfile(c)
	if err != nil {
		return err
	}

	// get the conference
	conference, err := GetConference(c, websafeKey)
	if err != nil {
		return errBadRequest(err, "conference does not exist")
	}

	if !profile.Registered(conference.ID) {
		return ErrNotRegistered
	}

	return datastore.RunInTransaction(c, func(c context.Context) error {
		// unregister from the conference
		profile.Cancel(conference.ID)
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
