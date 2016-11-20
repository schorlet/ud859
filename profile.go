package ud859

import (
	"golang.org/x/net/context"

	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/user"
)

// profileKey returns a datastore key for the identified user.
func profileKey(c context.Context) (*datastore.Key, error) {
	if u := user.Current(c); u != nil {
		return datastore.NewKey(c, "Profile", u.String(), 0, nil), nil
	}
	return nil, ErrUnauthorized
}

// GetProfile returns the profile associated with the current user.
func (ConferenceAPI) GetProfile(c context.Context) (*Profile, error) {
	pkey, err := profileKey(c)
	if err != nil {
		return nil, err
	}
	return getProfile(c, pkey)
}

func getProfile(c context.Context, key *datastore.Key) (*Profile, error) {
	// get the profile
	profile := new(Profile)
	err := datastore.Get(c, key, profile)
	if err != nil && err != datastore.ErrNoSuchEntity {
		return nil, err
	}

	profile.Email = user.Current(c).Email
	return profile, nil
}

// SaveProfile creates or updates the profile associated with the current user.
func (ConferenceAPI) SaveProfile(c context.Context, form *ProfileForm) error {
	pkey, err := profileKey(c)
	if err != nil {
		return err
	}

	// set the form values
	profile := &Profile{
		Email:        user.Current(c).Email,
		DisplayName:  form.DisplayName,
		TeeShirtSize: form.TeeShirtSize,
	}

	_, err = datastore.Put(c, pkey, profile)
	return err
}
