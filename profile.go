package ud859

import (
	"golang.org/x/net/context"

	"google.golang.org/appengine/datastore"

	"github.com/GoogleCloudPlatform/go-endpoints/endpoints"
)

// Profile defines an identified user.
type Profile struct {
	Email        string `json:"-"`
	DisplayName  string `json:"displayName"`
	TeeShirtSize string `json:"teeShirtSize"`
	// Conferences is a list of conferences WebsafeKey.
	Conferences []string `json:"conferenceKeysToAttend"`
}

// ProfileForm gives details about a Profile to create or update.
type ProfileForm struct {
	DisplayName  string `json:"displayName"`
	TeeShirtSize string `json:"teeShirtSize"`
}

type identity struct {
	key   *datastore.Key
	email string
}

func profileID(c context.Context) (*identity, error) {
	u, err := endpoints.CurrentUser(c, scopes, audiences, clientIds)
	if err != nil {
		return nil, errUnauthorized(err, "signin required")
	}
	return &identity{
		key:   datastore.NewKey(c, "Profile", u.String(), 0, nil),
		email: u.Email,
	}, nil
}

// GetProfile returns the Profile associated with the current user.
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
		return nil, errInternalServer(err, "unable to get profile")
	}

	profile.Email = pid.email
	return profile, nil
}

// SaveProfile saves the current user's Profile in the datastore from the specified ProfileForm.
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
		if err != nil {
			return errInternalServer(err, "unable to save profile")
		}
		return nil
	}, nil)
}
