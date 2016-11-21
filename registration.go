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
func (ConferenceAPI) GotoConference(c context.Context, form *ConferenceKeyForm) error {
	pid, err := profileID(c)
	if err != nil {
		return err
	}
	ckey, err := datastore.DecodeKey(form.WebsafeKey)
	if err != nil {
		return errBadRequest(err, "invalid conference key")
	}

	return datastore.RunInTransaction(c, func(c context.Context) error {
		// get the profile
		profile, err := getProfile(c, pid)
		if err != nil {
			return err
		}

		// get the conference
		conference, err := getConference(c, ckey)
		if err != nil {
			return errBadRequest(err, "conference does not exist")
		}

		if profile.HasRegistered(conference.ID) {
			return ErrRegistered
		}
		if conference.SeatsAvailable <= 0 {
			return ErrNoSeatsAvailable
		}

		// register to the conference
		profile.register(conference.ID)
		_, err = datastore.Put(c, pid.key, profile)
		if err != nil {
			return err
		}

		// decrease the available seats
		conference.SeatsAvailable--
		_, err = datastore.Put(c, ckey, conference)
		return err

	}, &datastore.TransactionOptions{XG: true})
}

// CancelConference cancels the registration to the specified conference.
func (ConferenceAPI) CancelConference(c context.Context, form *ConferenceKeyForm) error {
	pid, err := profileID(c)
	if err != nil {
		return err
	}
	ckey, err := datastore.DecodeKey(form.WebsafeKey)
	if err != nil {
		return errBadRequest(err, "invalid conference key")
	}

	return datastore.RunInTransaction(c, func(c context.Context) error {
		// get the profile
		profile, err := getProfile(c, pid)
		if err != nil {
			return err
		}

		// get the conference
		conference, err := getConference(c, ckey)
		if err != nil {
			return errBadRequest(err, "conference does not exist")
		}

		if !profile.HasRegistered(conference.ID) {
			return ErrNotRegistered
		}

		// unregister from the conference
		profile.unRegister(conference.ID)
		_, err = datastore.Put(c, pid.key, profile)
		if err != nil {
			return err
		}

		// increase the available seats
		conference.SeatsAvailable++
		_, err = datastore.Put(c, ckey, conference)
		return err

	}, &datastore.TransactionOptions{XG: true})
}
