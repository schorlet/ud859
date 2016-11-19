// +build go1.7

package ud859_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/schorlet/ud859"

	"github.com/GoogleCloudPlatform/go-endpoints/endpoints"
	"google.golang.org/appengine/aetest"
)

type client struct {
	server *endpoints.Server
	inst   aetest.Instance
}

type testFunc func(*client, *testing.T)

func withClient(c *client, fn testFunc) func(*testing.T) {
	return func(t *testing.T) {
		fn(c, t)
	}
}

func (c *client) do(url string, v interface{}) (*httptest.ResponseRecorder, error) {
	return c.doID("", url, v)
}

func (c *client) doID(email, url string, v interface{}) (*httptest.ResponseRecorder, error) {
	// payload
	var body io.Reader
	if v != nil {
		b, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(b)

	} else {
		body = strings.NewReader("{}")
	}

	// request
	r, err := c.inst.NewRequest("POST", url, body)
	if err != nil {
		_ = c.inst.Close()
		return nil, err
	}
	if email != "" {
		r.Header.Set("X-AppEngine-User-Email", email)
	}

	// response
	w := httptest.NewRecorder()

	// serve
	c.server.ServeHTTP(w, r)
	return w, nil
}

// test

func TestAPI(t *testing.T) {
	server := endpoints.NewServer("")
	if err := ud859.RegisterConferenceAPI(server); err != nil {
		t.Fatal(err)
	}

	inst, err := aetest.NewInstance(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer inst.Close()

	c := &client{server, inst}

	t.Run("GetProfile", withClient(c, getProfile))
	t.Run("SaveProfile", withClient(c, saveProfile))

	t.Run("GetConference", withClient(c, getConference))
	t.Run("CreateConference", withClient(c, createConference))

	t.Run("QueryConferences", withClient(c, queryConferences))

	t.Run("Registration", withClient(c, gotoConferences))
}

// profile

func getProfile(c *client, t *testing.T) {
	// get profile unauthorized
	w, err := c.do("/ConferenceAPI.GetProfile", nil)
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want:%d, got:%d", http.StatusUnauthorized, w.Code)
	}

	verifyProfile(c, t, new(ud859.ProfileForm))
}

func saveProfile(c *client, t *testing.T) {
	form := &ud859.ProfileForm{
		DisplayName:  "ud859",
		TeeShirtSize: ud859.SizeXL,
	}

	// save profile unauthorized
	w, err := c.do("/ConferenceAPI.SaveProfile", form)
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want:%d, got:%d", http.StatusUnauthorized, w.Code)
	}

	// save profile
	w, err = c.doID("ud859@udacity.com", "/ConferenceAPI.SaveProfile", form)
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != http.StatusOK {
		t.Fatalf("want:%d, got:%d", http.StatusOK, w.Code)
	}
	verifyProfile(c, t, form)

	// update profile
	form.TeeShirtSize = ud859.SizeXXL
	w, err = c.doID("ud859@udacity.com", "/ConferenceAPI.SaveProfile", form)
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != http.StatusOK {
		t.Fatalf("want:%d, got:%d", http.StatusOK, w.Code)
	}
	verifyProfile(c, t, form)

}

func verifyProfile(c *client, t *testing.T, form *ud859.ProfileForm) {
	// get profile
	w, err := c.doID("ud859@udacity.com", "/ConferenceAPI.GetProfile", nil)
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != http.StatusOK {
		t.Fatalf("want:%d, got:%d", http.StatusOK, w.Code)
	}

	// decode the profile
	profile := new(ud859.Profile)
	err = json.NewDecoder(w.Body).Decode(profile)
	if err != nil {
		t.Fatal(err)
	}

	// verify the profile
	if profile.Email != "" {
		t.Errorf("want:empty, got:%s", profile.Email)
	}
	if profile.DisplayName != form.DisplayName {
		t.Errorf("want:%s, got:%s", form.DisplayName, profile.DisplayName)
	}
	if profile.TeeShirtSize != form.TeeShirtSize {
		t.Errorf("want:%d, got:%d", form.TeeShirtSize, profile.TeeShirtSize)
	}
}

// conference

func getConference(c *client, t *testing.T) {
	// get conference without key
	w, err := c.do("/ConferenceAPI.GetConference", nil)
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want:%d, got:%d", http.StatusBadRequest, w.Code)
	}

	// get conference with bad key
	w, err = c.do("/ConferenceAPI.GetConference", &ud859.ConferenceKeyForm{"foo"})
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want:%d, got:%d", http.StatusBadRequest, w.Code)
	}
}

func createConference(c *client, t *testing.T) {
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
			Topics:       "Programming,Go,Mountain",
			City:         "Denver, Colorado",
			StartDate:    "2016-07-11",
			EndDate:      "2016-07-13",
			MaxAttendees: "10",
		},
	}

	for _, form := range forms {
		// save conference unauthorized
		w, err := c.do("/ConferenceAPI.CreateConference", form)
		if err != nil {
			t.Fatal(err)
		}
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("want:%d, got:%d", http.StatusUnauthorized, w.Code)
		}

		// save conference
		w, err = c.doID("ud859@udacity.com", "/ConferenceAPI.CreateConference", form)
		if err != nil {
			t.Fatal(err)
		}
		if w.Code != http.StatusOK {
			t.Fatalf("want:%d, got:%d", http.StatusOK, w.Code)
		}

		// decode the conference key
		key := new(ud859.ConferenceKeyForm)
		err = json.NewDecoder(w.Body).Decode(key)
		if err != nil {
			t.Fatal(err)
		}

		verifyConference(c, t, key, form)
	}

	// query conferences created
	w, err := c.doID("ud859@udacity.com", "/ConferenceAPI.ConferencesCreated", nil)
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != http.StatusOK {
		t.Fatalf("want:%d, got:%d", http.StatusOK, w.Code)
	}

	// decode the conferences
	conferences := new(ud859.Conferences)
	err = json.NewDecoder(w.Body).Decode(conferences)
	if err != nil {
		t.Fatal(err)
	}
	if len(conferences.Items) != 2 {
		t.Errorf("want:2, got:%d", len(conferences.Items))
	}
}

func verifyConference(c *client, t *testing.T,
	key *ud859.ConferenceKeyForm, form *ud859.ConferenceForm) {

	// get conference
	w, err := c.do("/ConferenceAPI.GetConference", key)
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != http.StatusOK {
		t.Fatalf("want:%d, got:%d", http.StatusOK, w.Code)
	}

	// decode the conference
	conference := new(ud859.Conference)
	err = json.NewDecoder(w.Body).Decode(conference)
	if err != nil {
		t.Fatal(err)
	}

	// verify the conference
	if conference.ID != 0 {
		t.Errorf("want:0, got:%d", conference.ID)
	}
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

// query

func queryConferences(c *client, t *testing.T) {
	t.Run("Nofilters", withClient(c, queryNofilters))
	t.Run("Filters", withClient(c, queryFilters))
}

func queryNofilters(c *client, t *testing.T) {
	w, err := c.do("/ConferenceAPI.QueryConferences", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("want:%d, got:%d", http.StatusOK, w.Code)
	}

	// decode the conferences
	conferences := new(ud859.Conferences)
	err = json.NewDecoder(w.Body).Decode(conferences)
	if err != nil {
		t.Fatal(err)
	}
	if len(conferences.Items) != 2 {
		t.Errorf("want:2, got:%d", len(conferences.Items))
	}
}

func queryFilters(c *client, t *testing.T) {
	type r struct {
		field, operator string
		value           interface{}
	}
	tts := []struct {
		restrictions []r
		expected     int
	}{
		{[]r{{ud859.City, ud859.EQ, "Paris"}}, 1},
		{[]r{{ud859.City, ud859.EQ, "London"}}, 0},
		{[]r{{ud859.City, ud859.EQ, "Denver"}}, 0},
		{[]r{{ud859.City, ud859.EQ, "Denver, Colorado"}}, 1},
		//
		{[]r{{ud859.Topics, ud859.EQ, "Go"}}, 2},
		{[]r{{ud859.Topics, ud859.EQ, "Programming"}}, 2},
		{[]r{{ud859.Topics, ud859.EQ, "Mountain"}}, 1},
		{[]r{{ud859.Topics, ud859.EQ, "Dart"}}, 0},
		//
		{[]r{{ud859.MaxAttendees, ud859.GT, 0}}, 2},
		{[]r{{ud859.MaxAttendees, ud859.GT, 1}}, 1},
		{[]r{{ud859.MaxAttendees, ud859.GTE, 1}}, 2},
		{[]r{{ud859.MaxAttendees, ud859.GT, 10}}, 0},
		{[]r{{ud859.MaxAttendees, ud859.LT, 10}}, 1},
		{[]r{{ud859.MaxAttendees, ud859.LTE, 10}}, 2},
		//
		{[]r{{ud859.StartDate, ud859.GTE, "2016-10-01"},
			{ud859.StartDate, ud859.LTE, "2016-10-31"}}, 1},
		{[]r{{ud859.StartDate, ud859.GTE, "2016-01-01"}}, 2},
		{[]r{{ud859.StartDate, ud859.GTE, "2017-01-01"}}, 0},
		//
		{[]r{{ud859.City, ud859.EQ, "Paris"},
			{ud859.StartDate, ud859.GT, "2016-10-01"}}, 1},
		{[]r{{ud859.City, ud859.EQ, "Paris"},
			{ud859.StartDate, ud859.GT, "2016-11-01"}}, 0},
		{[]r{{ud859.City, ud859.EQ, "Paris"},
			{ud859.Topics, ud859.EQ, "Go"},
			{ud859.StartDate, ud859.GT, "2016-10-01"}}, 1},
		{[]r{{ud859.City, ud859.EQ, "Paris"},
			{ud859.Topics, ud859.EQ, "Go"},
			{ud859.StartDate, ud859.GT, "2016-11-01"}}, 0},
	}

	for _, tt_donotuse := range tts {
		tt := tt_donotuse

		t.Run(tt.restrictions[0].field, func(t *testing.T) {
			t.Parallel()

			query := new(ud859.ConferenceQueryForm)
			for _, r := range tt.restrictions {
				query.Filter(r.field, r.operator, r.value)
			}

			w, err := c.do("/ConferenceAPI.QueryConferences", query)
			if w.Code != http.StatusOK {
				t.Logf("%v", w.Body)
				t.Fatalf("want:%d, got:%d", http.StatusOK, w.Code)
			}

			// decode the conferences
			conferences := new(ud859.Conferences)
			err = json.NewDecoder(w.Body).Decode(conferences)
			if err != nil {
				t.Fatal(err)
			}
			if len(conferences.Items) != tt.expected {
				t.Errorf("want:%d, got:%d", tt.expected, len(conferences.Items))
			}
		})
	}
}

// registration

func gotoConferences(c *client, t *testing.T) {
	t.Run("BadConference", withClient(c, gotoUnknown))
	t.Run("NotRegistered", withClient(c, gotoUnRegistered))
	t.Run("Register", withClient(c, gotoRegistration))
}

func gotoUnknown(c *client, t *testing.T) {
	key := &ud859.ConferenceKeyForm{"foo"}

	tts := []struct {
		email  string
		status int
	}{
		{"", http.StatusUnauthorized},
		{"ud859@udacity.com", http.StatusBadRequest},
	}

	for _, tt := range tts {
		// register
		w, err := c.doID(tt.email, "/ConferenceAPI.GotoConference", key)
		if err != nil {
			t.Fatal(err)
		}
		if w.Code != tt.status {
			t.Errorf("want:%d, got:%d", tt.status, w.Code)
		}

		// unregister
		w, err = c.doID(tt.email, "/ConferenceAPI.CancelConference", key)
		if err != nil {
			t.Fatal(err)
		}
		if w.Code != tt.status {
			t.Errorf("want:%d, got:%d", tt.status, w.Code)
		}
	}
}

func gotoUnRegistered(c *client, t *testing.T) {
	// query conferences
	w, err := c.do("/ConferenceAPI.QueryConferences", nil)
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != http.StatusOK {
		t.Fatalf("want:%d, got:%d", http.StatusOK, w.Code)
	}

	// decode the conferences
	conferences := new(ud859.Conferences)
	err = json.NewDecoder(w.Body).Decode(conferences)
	if err != nil {
		t.Fatal(err)
	}
	if len(conferences.Items) == 0 {
		t.Fatal("want:>0, got:0")
	}

	type test struct {
		email  string
		key    string
		status int
	}
	var tts []test

	for _, c := range conferences.Items {
		tts = append(tts,
			test{"", c.WebsafeKey, http.StatusUnauthorized})
		tts = append(tts,
			test{"ud859@udacity.com", c.WebsafeKey, http.StatusConflict})
	}

	for _, tt := range tts {
		key := &ud859.ConferenceKeyForm{tt.key}

		// unregister
		w, err = c.doID(tt.email, "/ConferenceAPI.CancelConference", key)
		if err != nil {
			t.Fatal(err)
		}
		if w.Code != tt.status {
			t.Errorf("want:%d, got:%d", tt.status, w.Code)
		}
	}
}

func gotoRegistration(c *client, t *testing.T) {
	// query conferences
	w, err := c.do("/ConferenceAPI.QueryConferences", nil)
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != http.StatusOK {
		t.Fatalf("want:%d, got:%d", http.StatusOK, w.Code)
	}

	// decode the conferences
	conferences := new(ud859.Conferences)
	err = json.NewDecoder(w.Body).Decode(conferences)
	if err != nil {
		t.Fatal(err)
	}
	if len(conferences.Items) == 0 {
		t.Fatal("want:>0, got:0")
	}

	for i, conference := range conferences.Items {
		key := &ud859.ConferenceKeyForm{conference.WebsafeKey}

		// register
		w, err = c.doID("ud859@udacity.com", "/ConferenceAPI.GotoConference", key)
		if err != nil {
			t.Fatal(err)
		}
		if w.Code != http.StatusOK {
			t.Errorf("want:%d, got:%d", http.StatusOK, w.Code)
		}

		// register twice
		w, err = c.doID("ud859@udacity.com", "/ConferenceAPI.GotoConference", key)
		if err != nil {
			t.Fatal(err)
		}
		if w.Code != http.StatusConflict {
			t.Errorf("want:%d, got:%d", http.StatusConflict, w.Code)
		}

		verifyConferencesToAttend(c, t, i+1)

		// user2 try to register
		w, err = c.doID("user2@udacity.com", "/ConferenceAPI.GotoConference", key)
		if err != nil {
			t.Fatal(err)
		}
		if conference.SeatsAvailable == 1 {
			if w.Code != http.StatusConflict {
				t.Errorf("want:%d, got:%d", http.StatusConflict, w.Code)
			}
		} else if w.Code != http.StatusOK {
			t.Errorf("want:%d, got:%d", http.StatusOK, w.Code)
		}
	}

	for i, conference := range conferences.Items {
		key := &ud859.ConferenceKeyForm{conference.WebsafeKey}

		// unregister
		w, err = c.doID("ud859@udacity.com", "/ConferenceAPI.CancelConference", key)
		if err != nil {
			t.Fatal(err)
		}
		if w.Code != http.StatusOK {
			t.Errorf("want:%d, got:%d", http.StatusOK, w.Code)
		}

		// unregister twice
		w, err = c.doID("ud859@udacity.com", "/ConferenceAPI.CancelConference", key)
		if err != nil {
			t.Fatal(err)
		}
		if w.Code != http.StatusConflict {
			t.Errorf("want:%d, got:%d", http.StatusConflict, w.Code)
		}

		verifyConferencesToAttend(c, t, len(conferences.Items)-i-1)
	}

	verifyConferencesToAttend(c, t, 0)
}

func verifyConferencesToAttend(c *client, t *testing.T, count int) {
	// query conferences to attend
	w, err := c.doID("ud859@udacity.com", "/ConferenceAPI.ConferencesToAttend", nil)
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != http.StatusOK {
		t.Fatalf("want:%d, got:%d", http.StatusOK, w.Code)
	}

	// decode the conferences
	conferences := new(ud859.Conferences)
	err = json.NewDecoder(w.Body).Decode(conferences)
	if err != nil {
		t.Fatal(err)
	}
	if len(conferences.Items) != count {
		t.Errorf("want:%d, got:%d", count, len(conferences.Items))
	}
}
