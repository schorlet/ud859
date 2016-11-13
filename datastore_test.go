// +build go1.7

package ud859_test

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/schorlet/ud859"

	"golang.org/x/net/context"

	"google.golang.org/appengine"
	"google.golang.org/appengine/aetest"
)

type instanceFunc func(aetest.Instance, *testing.T)

func withInstance(inst aetest.Instance, fn instanceFunc) func(*testing.T) {
	return func(t *testing.T) {
		fn(inst, t)
	}
}

type ctxFunc func(string) context.Context
type contextFunc func(ctxFunc, *testing.T)

func withContext(inst aetest.Instance, fn contextFunc) func(*testing.T) {
	return func(t *testing.T) {
		ctx := func(email string) context.Context {

			r, err := inst.NewRequest("GET", "/", nil)
			if err != nil {
				_ = inst.Close()
				t.Fatal(err)
			}
			if email != "" {
				r.Header.Set("X-AppEngine-User-Email", email)
			}
			return appengine.NewContext(r)
		}

		fn(ctx, t)
	}
}

func TestConference(t *testing.T) {
	// start the development server
	inst, err := aetest.NewInstance(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer inst.Close()

	t.Run("profileWithoutUser", withContext(inst, profileWithoutUser))
	t.Run("profileFirstTime", withContext(inst, profileFirstTime))
	t.Run("profileCreate", withContext(inst, profileCreate))
	t.Run("profileUpdate", withContext(inst, profileUpdate))

	t.Run("conferenceCreateWithoutUser", withContext(inst, conferenceCreateWithoutUser))
	t.Run("conferenceCreate", withContext(inst, conferenceCreate))
	t.Run("conferencesQuery", withInstance(inst, conferencesQuery))
	t.Run("conferenceGoto", withContext(inst, conferenceGoto))
}

// profile

func profileWithoutUser(ctx ctxFunc, t *testing.T) {
	cAny := ctx("")

	profile, err := ud859.GetProfile(cAny)
	if err != ud859.ErrUnauthorized {
		t.Errorf("want:%v, got:%v", ud859.ErrUnauthorized, err)
	}
	if profile != nil {
		t.Errorf("want:nil, got:%v", profile)
	}
}

func profileFirstTime(ctx ctxFunc, t *testing.T) {
	c := ctx("ud859@udacity.com")

	profile, err := ud859.GetProfile(c)
	if err != nil {
		t.Errorf("want:nil, got:%v", err)
	}
	if profile == nil {
		t.Error("want:profile, got:nil")
	}
}

func profileCreate(ctx ctxFunc, t *testing.T) {
	c := ctx("ud859@udacity.com")
	cAny := ctx("")

	// create a profile form
	form := &ud859.ProfileForm{
		DisplayName:  "ud859",
		TeeShirtSize: ud859.SIZE_L,
	}

	err := ud859.SaveProfile(cAny, form)
	if err != ud859.ErrUnauthorized {
		t.Errorf("want:%v, got:%v", ud859.ErrUnauthorized, err)
	}

	err = ud859.SaveProfile(c, form)
	if err != nil {
		t.Fatal(err)
	}

	// verify the profile
	profile, err := ud859.GetProfile(c)
	if err != nil {
		t.Fatal(err)
	}

	verifyProfile(t, profile, form)
}

func profileUpdate(ctx ctxFunc, t *testing.T) {
	c := ctx("ud859@udacity.com")

	// create a profile form
	form := &ud859.ProfileForm{
		DisplayName:  "udacity 859",
		TeeShirtSize: ud859.SIZE_XL,
	}
	err := ud859.SaveProfile(c, form)
	if err != nil {
		t.Fatal(err)
	}

	// verify the profile
	profile, err := ud859.GetProfile(c)
	if err != nil {
		t.Fatal(err)
	}

	verifyProfile(t, profile, form)
}

func verifyProfile(t *testing.T, profile *ud859.Profile, form *ud859.ProfileForm) {
	if profile.Email != "ud859@udacity.com" {
		t.Errorf("want:%s, got:%s", "ud859@udacity.com", profile.Email)
	}
	verifyWebProfile(t, profile, form)
}

func verifyWebProfile(t *testing.T, profile *ud859.Profile, form *ud859.ProfileForm) {
	if profile.DisplayName != form.DisplayName {
		t.Errorf("want:%s, got:%s", form.DisplayName, profile.DisplayName)
	}
	if profile.TeeShirtSize != form.TeeShirtSize {
		t.Errorf("want:%d, got:%d", form.TeeShirtSize, profile.TeeShirtSize)
	}
}

// conference

func conferenceCreateWithoutUser(ctx ctxFunc, t *testing.T) {
	cAny := ctx("")

	form := ud859.ConferenceForm{
		Name:         "dotGo",
		Description:  "The European Go conference",
		Topics:       "Programming,Go",
		City:         "Paris",
		StartDate:    "2016-10-10",
		EndDate:      "2016-10-10",
		MaxAttendees: "200",
	}

	// create the conference
	conference, err := ud859.CreateConference(cAny, &form)
	if err != ud859.ErrUnauthorized {
		t.Errorf("want:%v, got:%v", ud859.ErrUnauthorized, err)
	}
	if conference != nil {
		t.Errorf("want:nil, got:%v", conference)
	}
}

func conferenceCreate(ctx ctxFunc, t *testing.T) {
	c := ctx("ud859@udacity.com")
	cAny := ctx("")

	forms := []*ud859.ConferenceForm{
		{
			Name:         "dotGo",
			Description:  "The European Go conference",
			Topics:       "Programming,Go",
			City:         "Paris",
			StartDate:    "2016-10-10",
			EndDate:      "2016-10-10",
			MaxAttendees: "1",
		},
		{
			Name:         "gophercon",
			Description:  "Largest event in the world dedicated to the Go programming language",
			Topics:       "Programming,Go",
			City:         "Denver, Colorado",
			StartDate:    "2016-07-11",
			EndDate:      "2016-07-13",
			MaxAttendees: "2",
		},
	}

	for _, form := range forms {
		// create the conference
		conference, err := ud859.CreateConference(c, form)
		if err != nil {
			t.Fatal(err)
		}
		verifyConference(t, conference, form)

		// verify the conference
		conference, err = ud859.GetConference(c, conference.WebsafeKey)
		if err != nil {
			t.Fatal(err)
		}
		verifyConference(t, conference, form)

		// verify the conference
		conference, err = ud859.GetConference(cAny, conference.WebsafeKey)
		if err != nil {
			t.Fatal(err)
		}
		verifyConference(t, conference, form)
	}
}

func verifyConference(t *testing.T, conference *ud859.Conference, form *ud859.ConferenceForm) {
	if conference.ID == 0 {
		t.Error("conference.ID is empty")
	}
	verifyWebConference(t, conference, form)
}

func verifyWebConference(t *testing.T, conference *ud859.Conference, form *ud859.ConferenceForm) {
	if conference.WebsafeKey == "" {
		t.Error("conference.WebsafeKey is empty")
	}
	if conference.Name != form.Name {
		t.Errorf("want:%s, got:%s", form.Name, conference.Name)
	}

	startDate := conference.StartDate.Format(ud859.TimeFormat)
	if startDate != form.StartDate {
		t.Errorf("want:%s, got:%s", form.StartDate, startDate)
	}
	endDate := conference.EndDate.Format(ud859.TimeFormat)
	if endDate != form.EndDate {
		t.Errorf("want:%s, got:%s", form.EndDate, endDate)
	}

	topics := strings.Join(conference.Topics, ",")
	if topics != form.Topics {
		t.Errorf("want:%s, got:%s", conference.Topics, topics)
	}
	if strconv.Itoa(conference.SeatsAvailable) != form.MaxAttendees {
		t.Errorf("want:%s, got:%d", form.MaxAttendees, conference.SeatsAvailable)
	}
}

// conferencesQuery

func conferencesQuery(inst aetest.Instance, t *testing.T) {
	// run all sub-tests in parallel
	t.Run("Nofilters", withContext(inst, queryNofilters))
	t.Run("City", withContext(inst, queryCity))
	t.Run("Topics", withContext(inst, queryTopics))
	t.Run("MaxAttendees", withContext(inst, queryMaxAttendees))
	t.Run("StartDate", withContext(inst, queryStartDate))
	t.Run("Created", withContext(inst, queryCreated))
	t.Run("CreatedUser1", withContext(inst, queryCreatedUser1))
}

func queryNofilters(ctx ctxFunc, t *testing.T) {
	t.Parallel()
	c := ctx("ud859@udacity.com")

	conferences, err := ud859.QueryConferences(c)
	if err != nil {
		t.Fatal(err)
	}
	if len(conferences) != 2 {
		t.Fatalf("want:2, got:%d", len(conferences))
	}
}

func queryCity(ctx ctxFunc, t *testing.T) {
	t.Parallel()
	c := ctx("ud859@udacity.com")

	query := new(ud859.ConferenceQueryForm).
		Filter(ud859.City, ud859.EQ, "Paris")

	conferences, err := ud859.QueryConferencesFilter(c, query)
	if err != nil {
		t.Fatal(err)
	}
	if len(conferences) != 1 {
		t.Fatalf("want:1, got:%d", len(conferences))
	}
}

func queryTopics(ctx ctxFunc, t *testing.T) {
	t.Parallel()
	c := ctx("ud859@udacity.com")

	query := new(ud859.ConferenceQueryForm).
		Filter(ud859.Topics, ud859.GTE, "Go").
		Filter(ud859.Topics, ud859.LTE, "Go")

	conferences, err := ud859.QueryConferencesFilter(c, query)
	if err != nil {
		t.Fatal(err)
	}
	if len(conferences) != 2 {
		t.Fatalf("want:2, got:%d", len(conferences))
	}
}

func queryMaxAttendees(ctx ctxFunc, t *testing.T) {
	t.Parallel()
	c := ctx("ud859@udacity.com")

	// 1 < MaxAttendees
	query := new(ud859.ConferenceQueryForm).
		Filter(ud859.MaxAttendees, ud859.GT, 1)

	conferences, err := ud859.QueryConferencesFilter(c, query)
	if err != nil {
		t.Fatal(err)
	}
	if len(conferences) != 1 {
		t.Fatalf("want:1, got:%d", len(conferences))
	}

	// 1 < MaxAttendees < 10
	query = new(ud859.ConferenceQueryForm).
		Filter(ud859.MaxAttendees, ud859.GT, 1).
		Filter(ud859.MaxAttendees, ud859.LT, 10)

	conferences, err = ud859.QueryConferencesFilter(c, query)
	if err != nil {
		t.Fatal(err)
	}
	if len(conferences) != 1 {
		t.Fatalf("want:1, got:%d", len(conferences))
	}
}

func queryStartDate(ctx ctxFunc, t *testing.T) {
	t.Parallel()
	c := ctx("ud859@udacity.com")

	// 2016-10-01 <= Start <= 2016-10-31
	gte, _ := time.Parse(ud859.TimeFormat, "2016-10-01")
	lte, _ := time.Parse(ud859.TimeFormat, "2016-10-31")

	query := new(ud859.ConferenceQueryForm).
		Filter(ud859.StartDate, ud859.GTE, gte).
		Filter(ud859.StartDate, ud859.LTE, lte)

	conferences, err := ud859.QueryConferencesFilter(c, query)
	if err != nil {
		t.Fatal(err)
	}
	if len(conferences) != 1 {
		t.Fatalf("want:1, got:%d", len(conferences))
	}

	// 2016-01-01 <= Start
	gte, _ = time.Parse(ud859.TimeFormat, "2016-01-01")

	query = new(ud859.ConferenceQueryForm).
		Filter(ud859.StartDate, ud859.GTE, gte)

	conferences, err = ud859.QueryConferencesFilter(c, query)
	if err != nil {
		t.Fatal(err)
	}
	if len(conferences) != 2 {
		t.Fatalf("want:2, got:%d", len(conferences))
	}

	// 2017-01-01 <= Start
	gte, _ = time.Parse(ud859.TimeFormat, "2017-01-01")

	query = new(ud859.ConferenceQueryForm).
		Filter(ud859.StartDate, ud859.GTE, gte)

	conferences, err = ud859.QueryConferencesFilter(c, query)
	if err != nil {
		t.Fatal(err)
	}
	if len(conferences) != 0 {
		t.Fatalf("want:0, got:%d", len(conferences))
	}
}

func queryCreated(ctx ctxFunc, t *testing.T) {
	t.Parallel()
	c := ctx("ud859@udacity.com")

	conferences, err := ud859.ConferencesCreated(c)
	if err != nil {
		t.Fatal(err)
	}
	if len(conferences) != 2 {
		t.Fatalf("want:2, got:%d", len(conferences))
	}
}

func queryCreatedUser1(ctx ctxFunc, t *testing.T) {
	t.Parallel()
	c := ctx("user1@udacity.com")

	conferences, err := ud859.ConferencesCreated(c)
	if err != nil {
		t.Fatal(err)
	}
	if len(conferences) != 0 {
		t.Fatalf("want:0, got:%d", len(conferences))
	}
}

// conferenceGoto

func conferenceGoto(ctx ctxFunc, t *testing.T) {
	c := ctx("ud859@udacity.com")

	conferences, err := ud859.QueryConferences(c)
	if err != nil {
		t.Fatal(err)
	}
	if len(conferences) == 0 {
		t.Fatal("conferences slice is empty")
	}
	// pick the dotGo conference
	conferenceID := conferences[0].ID
	websafeKey := conferences[0].WebsafeKey

	// cancel registration when not registered
	err = ud859.CancelConference(c, websafeKey)
	if err != ud859.ErrNotRegistered {
		t.Fatalf("want:%v, got:%v", ud859.ErrNotRegistered, err)
	}

	// register to the conference
	err = ud859.GotoConference(c, websafeKey)
	if err != nil {
		t.Fatal(err)
	}

	// register twice to the conference
	err = ud859.GotoConference(c, websafeKey)
	if err != ud859.ErrRegistered {
		t.Fatalf("want:%v, got:%v", ud859.ErrRegistered, err)
	}

	// register with another user
	c2 := ctx("other@udacity.com")
	err = ud859.GotoConference(c2, websafeKey)
	if err != ud859.ErrNoSeatsAvailable {
		t.Fatalf("want:%v, got:%v", ud859.ErrNoSeatsAvailable, err)
	}

	// verify the registration tho the conference
	profile, err := ud859.GetProfile(c)
	if err != nil {
		t.Fatal(err)
	}
	if !profile.Registered(conferenceID) {
		t.Errorf("want:registered to %d", conferenceID)
	}

	// verify the conferences to attend
	conferences, err = ud859.ConferencesToAttend(c)
	if err != nil {
		t.Fatal(err)
	}
	if len(conferences) != 1 {
		t.Errorf("want:1, got:%d", len(conferences))
	}
	if conferences[0].ID != conferenceID {
		t.Errorf("want:%d, got:%d", conferenceID, conferences[0].ID)
	}

	// cancel registration
	err = ud859.CancelConference(c, websafeKey)
	if err != nil {
		t.Fatal(err)
	}

	// verify there is no conference to attend
	profile, err = ud859.GetProfile(c)
	if err != nil {
		t.Fatal(err)
	}
	if profile.Registered(conferenceID) {
		t.Errorf("want: not registered to %d", conferenceID)
	}

	// verify there is no conference to attend
	conferences, err = ud859.ConferencesToAttend(c)
	if err != nil {
		t.Fatal(err)
	}
	if len(conferences) != 0 {
		t.Errorf("want:0, got:%d", len(conferences))
	}
}
