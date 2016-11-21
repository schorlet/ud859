package ud859

import (
	"fmt"

	"golang.org/x/net/context"

	"google.golang.org/appengine/datastore"
)

// Filter adds a filter to the query.
func (q *ConferenceQueryForm) Filter(field string, op string, value interface{}) *ConferenceQueryForm {
	q.Filters = append(q.Filters, &Filter{field, op, value})
	return q
}

// CheckFilters verifies that the inequality filter applys only on the same field.
func (q *ConferenceQueryForm) CheckFilters() error {
	var found bool

	for _, filter := range q.Filters {
		if filter.Op != EQ {
			if found && filter.Field != q.inequalityFilter.Field {
				return errBadRequest(nil, "only one inequality filter is allowed")
			}

			found = true
			q.inequalityFilter = filter
		}
	}
	return nil
}

// Query returns the query to apply to the datastore.
func (q ConferenceQueryForm) Query() (*datastore.Query, error) {
	err := q.CheckFilters()
	if err != nil {
		return nil, err
	}

	query := datastore.NewQuery("Conference")

	if q.inequalityFilter != nil {
		// order first by the inequality filter
		query = query.Order(string(q.inequalityFilter.Field))
	}
	query = query.Order("Name")

	for _, filter := range q.Filters {
		query = query.Filter(
			fmt.Sprintf("%s %s", filter.Field, filter.Op), filter.Value)
	}

	return query, nil
}

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
	pkey, err := profileKey(c)
	if err != nil {
		return nil, err
	}

	// get the conferences whose parent is the profile key
	conferences := make([]*Conference, 0)
	query := datastore.NewQuery("Conference").Ancestor(pkey).Order("Name")

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

// ConferencesToAttend returns the Conferences to addend by the current user.
func (ConferenceAPI) ConferencesToAttend(c context.Context) (*Conferences, error) {
	pid, err := profileID(c)
	if err != nil {
		return nil, err
	}

	// get the profile
	profile, err := getProfile(c, pid)
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
		keys[i] = conferenceKey(c, conferenceID, pid.key)
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
