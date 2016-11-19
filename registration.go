package ud859

import (
	"golang.org/x/net/context"

	"google.golang.org/appengine/datastore"
)

func (p *Profile) register(conferenceID int64) {
	p.Conferences = append(p.Conferences, conferenceID)
}

func (p *Profile) unRegister(conferenceID int64) {
	for i, id := range p.Conferences {
		if id == conferenceID {
			p.Conferences = append(p.Conferences[:i], p.Conferences[i+1:]...)
			break
		}
	}
}

// HasRegistered returns true if the user has registered to the specified conference ID.
func (p Profile) HasRegistered(conferenceID int64) bool {
	for _, id := range p.Conferences {
		if id == conferenceID {
			return true
		}
	}
	return false
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
