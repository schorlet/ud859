package ud859

import (
	"golang.org/x/net/context"

	"google.golang.org/appengine/datastore"

	"github.com/GoogleCloudPlatform/go-endpoints/endpoints"
)

// profileKey returns a datastore key for the identified user.
func profileKey(c context.Context) (*datastore.Key, error) {
	u, err := endpoints.CurrentUser(c, scopes, audiences, clientIds)
	if err != nil {
		return nil, ErrUnauthorized
	}
	return datastore.NewKey(c, "Profile", u.String(), 0, nil), nil
}

type identity struct {
	key   *datastore.Key
	email string
}

func profileID(c context.Context) (*identity, error) {
	u, err := endpoints.CurrentUser(c, scopes, audiences, clientIds)
	if err != nil {
		return nil, ErrUnauthorized
	}
	return &identity{
		key:   datastore.NewKey(c, "Profile", u.String(), 0, nil),
		email: u.Email,
	}, nil
}

// GetProfile returns the profile associated with the current user.
func (ConferenceAPI) GetProfile(c context.Context) (*Profile, error) {
	pid, err := profileID(c)
	if err != nil {
		return nil, err
	}
	return getProfile(c, pid)
}

func getProfile(c context.Context, pid *identity) (*Profile, error) {
	// get the profile
	profile := new(Profile)
	err := datastore.Get(c, pid.key, profile)
	if err != nil && err != datastore.ErrNoSuchEntity {
		return nil, err
	}

	profile.Email = pid.email
	return profile, nil
}

// SaveProfile creates or updates the profile associated with the current user.
func (ConferenceAPI) SaveProfile(c context.Context, form *ProfileForm) error {
	pid, err := profileID(c)
	if err != nil {
		return err
	}

	return datastore.RunInTransaction(c, func(c context.Context) error {
		// get the profile
		profile, err := getProfile(c, pid)
		if err != nil {
			return err
		}

		// set the form values
		profile.DisplayName = form.DisplayName
		profile.TeeShirtSize = form.TeeShirtSize

		_, err = datastore.Put(c, pid.key, profile)
		return err
	}, nil)
}
