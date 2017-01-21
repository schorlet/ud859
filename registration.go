package ud859

import (
	"golang.org/x/net/context"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
)

func (p *Profile) register(websafeKey string) {
	p.Conferences = append(p.Conferences, websafeKey)
}

func (p *Profile) unregister(websafeKey string) {
	for i, key := range p.Conferences {
		if key == websafeKey {
			p.Conferences = append(p.Conferences[:i], p.Conferences[i+1:]...)
			break
		}
	}
}

// IsRegistered returns true if the user is registered to the specified conference websafeKey.
func (p Profile) IsRegistered(websafeKey string) bool {
	for _, key := range p.Conferences {
		if key == websafeKey {
			return true
		}
	}
	return false
}

// GotoConference performs the registration to the specified ConferenceKeyForm.
func (ConferenceAPI) GotoConference(c context.Context, form *ConferenceKeyForm) error {
	pid, err := profileID(c)
	if err != nil {
		return err
	}
	ckey, err := datastore.DecodeKey(form.WebsafeKey)
	if err != nil {
		return errBadRequest(err, "invalid conference key")
	}

	err = datastore.RunInTransaction(c, func(c context.Context) error {
		errc := make(chan error, 2)
		var profile *Profile
		var conference *Conference

		go func() {
			// get the profile
			var err error
			profile, err = getProfile(c, pid)
			errc <- err
		}()

		go func() {
			// get the conference
			var err error
			conference, err = getConference(c, ckey)
			if err != nil {
				err = errBadRequest(err, "conference does not exist")
			}
			errc <- err
		}()

		// wait and check for errors
		multi := make(appengine.MultiError, 0, 2)
		for i := 0; i < 2; i++ {
			if err := <-errc; err != nil {
				multi = append(multi, err)
			}
		}
		if len(multi) > 0 {
			return multi
		}

		if profile.IsRegistered(conference.WebsafeKey) {
			return ErrRegistered
		}
		if conference.SeatsAvailable <= 0 {
			return ErrNoSeatsAvailable
		}

		// register to the conference
		profile.register(conference.WebsafeKey)
		_, err = datastore.Put(c, pid.key, profile)
		if err != nil {
			return err
		}

		// decrease the available seats
		conference.SeatsAvailable--
		_, err = datastore.Put(c, ckey, conference)
		if err != nil {
			return err
		}

		// update indexation
		return indexConference(c, conference)

	}, &datastore.TransactionOptions{XG: true})

	if err != nil {
		return err
	}

	// clear cache
	_ = deleteCacheNoFilters.Call(c)
	return nil
}

// CancelConference cancels the registration to the specified ConferenceKeyForm.
func (ConferenceAPI) CancelConference(c context.Context, form *ConferenceKeyForm) error {
	pid, err := profileID(c)
	if err != nil {
		return err
	}
	ckey, err := datastore.DecodeKey(form.WebsafeKey)
	if err != nil {
		return errBadRequest(err, "invalid conference key")
	}

	err = datastore.RunInTransaction(c, func(c context.Context) error {
		errc := make(chan error, 2)
		var profile *Profile
		var conference *Conference

		go func() {
			// get the profile
			var err error
			profile, err = getProfile(c, pid)
			errc <- err
		}()

		go func() {
			// get the conference
			var err error
			conference, err = getConference(c, ckey)
			if err != nil {
				err = errBadRequest(err, "conference does not exist")
			}
			errc <- err
		}()

		// wait and check for errors
		multi := make(appengine.MultiError, 0, 2)
		for i := 0; i < 2; i++ {
			if err := <-errc; err != nil {
				multi = append(multi, err)
			}
		}
		if len(multi) > 0 {
			return multi
		}

		if !profile.IsRegistered(conference.WebsafeKey) {
			return ErrNotRegistered
		}

		// unregister from the conference
		profile.unregister(conference.WebsafeKey)
		_, err = datastore.Put(c, pid.key, profile)
		if err != nil {
			return err
		}

		// increase the available seats
		conference.SeatsAvailable++
		_, err = datastore.Put(c, ckey, conference)
		if err != nil {
			return err
		}

		// update indexation
		return indexConference(c, conference)

	}, &datastore.TransactionOptions{XG: true})

	if err != nil {
		return err
	}

	// clear cache
	_ = deleteCacheNoFilters.Call(c)
	return nil
}
