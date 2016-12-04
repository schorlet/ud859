package ud859

import (
	"golang.org/x/net/context"

	"google.golang.org/appengine/datastore"
)

// ConferenceQueryForm collects filters for searching for Conferences.
type ConferenceQueryForm struct {
	Filters []*Filter `json:"filters"`
}

// Filter describes a query restriction.
type Filter struct {
	Field string      `endpoints:"req"`
	Op    string      `endpoints:"req"`
	Value interface{} `endpoints:"req"`
}

// Filter adds a filter to the query.
func (q *ConferenceQueryForm) Filter(field string, op string, value interface{}) *ConferenceQueryForm {
	q.Filters = append(q.Filters, &Filter{field, op, value})
	return q
}

// QueryConferences searches for Conferences with the specified filters.
func (ConferenceAPI) QueryConferences(c context.Context, form *ConferenceQueryForm) (*Conferences, error) {
	// perform search on index
	if len(form.Filters) > 0 {
		return SearchConferences(c, form)
	}

	// get conferences from cache
	conferences := getCacheNoFilters(c)
	if conferences != nil {
		return conferences, nil
	}

	items := make([]*Conference, 0)
	keys, err := datastore.NewQuery("Conference").Order(StartDate).GetAll(c, &items)
	if err != nil {
		return nil, err
	}

	for i := 0; i < len(items); i++ {
		items[i].WebsafeKey = keys[i].Encode()
	}
	conferences = &Conferences{Items: items}

	// cache conferences
	_ = setCacheNoFilters.Call(c, conferences)

	return conferences, nil
}

// ConferencesCreated returns the Conferences created by the current user.
func (ConferenceAPI) ConferencesCreated(c context.Context) (*Conferences, error) {
	pkey, err := profileKey(c)
	if err != nil {
		return nil, err
	}

	// get the conferences whose parent is the profile key
	items := make([]*Conference, 0)
	query := datastore.NewQuery("Conference").Ancestor(pkey).Order(StartDate)

	keys, err := query.GetAll(c, &items)
	if err != nil {
		return nil, err
	}

	for i := 0; i < len(items); i++ {
		items[i].WebsafeKey = keys[i].Encode()
	}

	return &Conferences{Items: items}, nil
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
		items := make([]*Conference, 0)
		return &Conferences{Items: items}, nil
	}

	// get the conference keys
	keys := make([]*datastore.Key, len(profile.Conferences))
	for i, safeKey := range profile.Conferences {
		keys[i], err = datastore.DecodeKey(safeKey)
		if err != nil {
			return nil, err
		}
	}

	// get the conferences
	items := make([]*Conference, len(profile.Conferences))
	err = datastore.GetMulti(c, keys, items)
	if err != nil {
		return nil, err
	}

	for i := 0; i < len(items); i++ {
		items[i].WebsafeKey = keys[i].Encode()
	}

	// TODO: sort by StartDate
	return &Conferences{Items: items}, nil
}
