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
