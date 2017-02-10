// +build go1.7

package ud859_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/go-endpoints/endpoints"
	"github.com/schorlet/ud859"

	"golang.org/x/net/context"

	"google.golang.org/appengine/aetest"
	"google.golang.org/appengine/user"
)

const emailTest = "bob@email"

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
	return c.doAs("", url, v)
}

func (c *client) doID(url string, v interface{}) (*httptest.ResponseRecorder, error) {
	return c.doAs(emailTest, url, v)
}

func (c *client) doAs(email, url string, v interface{}) (*httptest.ResponseRecorder, error) {
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
		r.Header.Set("Authorization", "BEARER "+email)
	}

	// response
	w := httptest.NewRecorder()

	// serve
	c.server.ServeHTTP(w, r)
	return w, nil
}

// authenticator

type testAuthenticator struct{}

func testAuthenticatorFactory() endpoints.Authenticator {
	return testAuthenticator{}
}

func (testAuthenticator) CurrentOAuthClientID(c context.Context, scope string) (string, error) {
	return "YOUR-CLIENT-ID", nil
}

func (testAuthenticator) CurrentOAuthUser(c context.Context, scope string) (*user.User, error) {
	r := endpoints.HTTPRequest(c)
	fields := strings.Fields(r.Header.Get("Authorization"))
	return &user.User{Email: fields[1]}, nil
}

// test

func TestAPI(t *testing.T) {
	endpoints.AuthenticatorFactory = testAuthenticatorFactory

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
		t.Fatalf("got:%d, want:%d", w.Code, http.StatusUnauthorized)
	}

	verifyProfile(c, t, new(ud859.ProfileForm))
}

func saveProfile(c *client, t *testing.T) {
	form := &ud859.ProfileForm{
		DisplayName:  "bob",
		TeeShirtSize: "XL",
	}

	// save profile unauthorized
	w, err := c.do("/ConferenceAPI.SaveProfile", form)
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("got:%d, want:%d", w.Code, http.StatusUnauthorized)
	}

	// save profile
	w, err = c.doID("/ConferenceAPI.SaveProfile", form)
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != http.StatusOK {
		t.Fatalf("got:%d, want:%d", w.Code, http.StatusOK)
	}
	verifyProfile(c, t, form)

	// update profile
	form.TeeShirtSize = "XXL"
	w, err = c.doID("/ConferenceAPI.SaveProfile", form)
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != http.StatusOK {
		t.Fatalf("got:%d, want:%d", w.Code, http.StatusOK)
	}
	verifyProfile(c, t, form)

}

func verifyProfile(c *client, t *testing.T, form *ud859.ProfileForm) {
	// get profile
	w, err := c.doID("/ConferenceAPI.GetProfile", nil)
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != http.StatusOK {
		t.Fatalf("got:%d, want:%d", w.Code, http.StatusOK)
	}

	// decode the profile
	profile := new(ud859.Profile)
	err = json.NewDecoder(w.Body).Decode(profile)
	if err != nil {
		t.Fatal(err)
	}

	// verify the profile
	if profile.Email != "" {
		t.Errorf("got:%s, want:empty", profile.Email)
	}
	if profile.DisplayName != form.DisplayName {
		t.Errorf("got:%s, want:%s", profile.DisplayName, form.DisplayName)
	}
	if profile.TeeShirtSize != form.TeeShirtSize {
		t.Errorf("got:%s, want:%s", profile.TeeShirtSize, form.TeeShirtSize)
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
		t.Fatalf("got:%d, want:%d", w.Code, http.StatusBadRequest)
	}

	// get conference with bad key
	w, err = c.do("/ConferenceAPI.GetConference", &ud859.ConferenceKeyForm{WebsafeKey: "foo"})
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != http.StatusBadRequest {
		t.Fatalf("got:%d, want:%d", w.Code, http.StatusBadRequest)
	}
}

func createConference(c *client, t *testing.T) {
	// sorted by StartDate
	forms := []*ud859.ConferenceForm{
		{
			Name:         "gophercon",
			Description:  "Largest event in the world dedicated to the Go programming language",
			Topics:       []string{"Programming", "Go", "Mountain"},
			City:         "Denver, Colorado",
			StartDate:    "2016-07-11T23:00:00Z",
			EndDate:      "2016-07-13T23:00:00Z",
			MaxAttendees: "10",
		},
		{
			Name:         "dotGo",
			Description:  "The European Go conference",
			Topics:       []string{"Programming", "Go"},
			City:         "Paris",
			StartDate:    "2016-10-10T23:00:00Z",
			EndDate:      "2016-10-10T23:00:00Z",
			MaxAttendees: "1",
		},
	}

	for _, form := range forms {
		// save conference unauthorized
		w, err := c.do("/ConferenceAPI.CreateConference", form)
		if err != nil {
			t.Fatal(err)
		}
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("got:%d, want:%d", w.Code, http.StatusUnauthorized)
		}

		// save conference
		w, err = c.doID("/ConferenceAPI.CreateConference", form)
		if err != nil {
			t.Fatal(err)
		}
		if w.Code != http.StatusOK {
			t.Fatalf("got:%d, want:%d", w.Code, http.StatusOK)
		}

		// decode the conference created
		created := new(ud859.ConferenceCreated)
		err = json.NewDecoder(w.Body).Decode(created)
		if err != nil {
			t.Fatal(err)
		}
		if created.Name != form.Name {
			t.Errorf("got:%s, want:%s", created.Name, form.Name)
		}

		key := &ud859.ConferenceKeyForm{WebsafeKey: created.WebsafeKey}
		verifyConference(c, t, key, form)
	}

	// query conferences created
	w, err := c.doID("/ConferenceAPI.ConferencesCreated", nil)
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != http.StatusOK {
		t.Fatalf("got:%d, want:%d", w.Code, http.StatusOK)
	}

	// decode the conferences
	conferences := new(ud859.Conferences)
	err = json.NewDecoder(w.Body).Decode(conferences)
	if err != nil {
		t.Fatal(err)
	}
	if len(conferences.Items) != 2 {
		t.Errorf("got:%d, want:2", len(conferences.Items))
	}

	for i, conference := range conferences.Items {
		key := &ud859.ConferenceKeyForm{WebsafeKey: conference.WebsafeKey}
		verifyConference(c, t, key, forms[i])
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
		t.Fatalf("got:%d, want:%d", w.Code, http.StatusOK)
	}

	// decode the conference
	conference := new(ud859.Conference)
	err = json.NewDecoder(w.Body).Decode(conference)
	if err != nil {
		t.Fatal(err)
	}

	// verify the conference
	if conference.WebsafeKey == "" {
		t.Error("conference.WebsafeKey is empty")
	}
	if conference.Name != form.Name {
		t.Errorf("got:%s, want:%s", conference.Name, form.Name)
	}
	if conference.Organizer != "bob" {
		t.Errorf("got:%s, want:bob", conference.Organizer)
	}

	startDate := conference.StartDate.Format(time.RFC3339)
	if startDate != form.StartDate {
		t.Errorf("got:%s, want:%s", startDate, form.StartDate)
	}
	endDate := conference.EndDate.Format(time.RFC3339)
	if endDate != form.EndDate {
		t.Errorf("got:%s, want:%s", endDate, form.EndDate)
	}

	tf := strings.Join(form.Topics, ",")
	tc := strings.Join(conference.Topics, ",")
	if tc != tf {
		t.Errorf("got:%s, want:%s", tc, tf)
	}

	if strconv.Itoa(conference.SeatsAvailable) != form.MaxAttendees {
		t.Errorf("got:%d, want:%s", conference.SeatsAvailable, form.MaxAttendees)
	}
	if conference.SeatsAvailable != conference.MaxAttendees {
		t.Errorf("got:%d, want:%d", conference.SeatsAvailable, conference.MaxAttendees)
	}
}

// query

func queryConferences(c *client, t *testing.T) {
	t.Run("Nofilters", withClient(c, queryNofilters))
	t.Run("Invalid", withClient(c, queryInvalid))
	t.Run("Filters", withClient(c, queryFilters))
}

func queryNofilters(c *client, t *testing.T) {
	w, err := c.do("/ConferenceAPI.QueryConferences", nil)
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != http.StatusOK {
		t.Fatalf("got:%d, want:%d", w.Code, http.StatusOK)
	}

	// decode the conferences
	conferences := new(ud859.Conferences)
	err = json.NewDecoder(w.Body).Decode(conferences)
	if err != nil {
		t.Fatal(err)
	}
	if len(conferences.Items) != 2 {
		t.Errorf("got:%d, want:2", len(conferences.Items))
	}
}

func queryInvalid(c *client, t *testing.T) {
	type r ud859.Filter

	tts := []r{
		{ud859.Month, ud859.EQ, "dotGo"},
		{ud859.Month, "OK", 10},
		{ud859.StartDate, ud859.EQ, "dotGo"},
	}

	for _, tt := range tts {
		query := new(ud859.ConferenceQueryForm).
			Filter(tt.Field, tt.Op, tt.Value)

		w, err := c.do("/ConferenceAPI.QueryConferences", query)
		if err != nil {
			t.Fatal(err)
		}
		if w.Code != http.StatusBadRequest {
			t.Fatalf("got:%d, want:%d", w.Code, http.StatusBadRequest)
		}
	}
}

func queryFilters(c *client, t *testing.T) {
	type r ud859.Filter

	tts := []struct {
		restrictions []r
		expected     int
	}{
		{[]r{{ud859.Name, ud859.EQ, "dotGo"}}, 1},
		{[]r{{ud859.Name, ud859.NE, "dotGo"}}, 1},
		{[]r{{ud859.Name, ud859.EQ, "gophercon"}}, 1},
		{[]r{{ud859.Name, ud859.NE, "gophercon"}}, 1},
		{[]r{{ud859.Name, ud859.EQ, "Denver"}}, 0},
		{[]r{{ud859.Name, ud859.NE, "Denver"}}, 2},
		//
		{[]r{{ud859.City, ud859.EQ, "Paris"}}, 1},
		{[]r{{ud859.City, ud859.NE, "Paris"}}, 1},
		{[]r{{ud859.City, ud859.EQ, "London"}}, 0},
		{[]r{{ud859.City, ud859.NE, "London"}}, 2},
		{[]r{{ud859.City, ud859.EQ, "Denver"}}, 1},
		{[]r{{ud859.City, ud859.NE, "Denver"}}, 1},
		{[]r{{ud859.City, ud859.EQ, "Denver, Colorado"}}, 1},
		{[]r{{ud859.City, ud859.NE, "Denver, Colorado"}}, 1},
		//
		{[]r{{ud859.Topics, ud859.EQ, "Go"}}, 2},
		{[]r{{ud859.Topics, ud859.NE, "Go"}}, 0},
		{[]r{{ud859.Topics, ud859.EQ, "Go Programming"}}, 2},
		{[]r{{ud859.Topics, ud859.EQ, "Programming"}}, 2},
		{[]r{{ud859.Topics, ud859.EQ, "Mountain"}}, 1},
		{[]r{{ud859.Topics, ud859.NE, "Mountain"}}, 1},
		{[]r{{ud859.Topics, ud859.EQ, "Dart"}}, 0},
		{[]r{{ud859.Topics, ud859.NE, "Dart"}}, 2},
		//
		{[]r{{ud859.MaxAttendees, ud859.EQ, 1}}, 1},
		{[]r{{ud859.MaxAttendees, ud859.NE, 1}}, 1},
		{[]r{{ud859.MaxAttendees, ud859.GT, 0}}, 2},
		{[]r{{ud859.MaxAttendees, ud859.GT, 1}}, 1},
		{[]r{{ud859.MaxAttendees, ud859.GTE, 1}}, 2},
		{[]r{{ud859.MaxAttendees, ud859.GT, 10}}, 0},
		{[]r{{ud859.MaxAttendees, ud859.LT, 10}}, 1},
		{[]r{{ud859.MaxAttendees, ud859.LTE, 10}}, 2},
		//
		{[]r{{ud859.Month, ud859.EQ, 7}}, 1},
		{[]r{{ud859.Month, ud859.NE, 7}}, 1},
		{[]r{{ud859.Month, ud859.EQ, 10}}, 1},
		{[]r{{ud859.Month, ud859.NE, 10}}, 1},
		{[]r{{ud859.Month, ud859.EQ, 1}}, 0},
		{[]r{{ud859.Month, ud859.NE, 1}}, 2},
		{[]r{{ud859.Month, ud859.GT, 3}}, 2},
		{[]r{{ud859.Month, ud859.GT, 8}}, 1},
		{[]r{{ud859.Month, ud859.LT, 8}}, 1},
		{[]r{{ud859.Month, ud859.LT, 3}}, 0},
		//
		// inequality on multiple fields
		{[]r{{ud859.Month, ud859.GT, 3},
			{ud859.Topics, ud859.NE, "Dart"},
			{ud859.MaxAttendees, ud859.LTE, 10}}, 2},
		{[]r{{ud859.Month, ud859.GT, 3},
			{ud859.Topics, ud859.EQ, "Dart"},
			{ud859.MaxAttendees, ud859.LTE, 10}}, 0},
		//
		{[]r{{ud859.SeatsAvailable, ud859.EQ, 1}}, 1},
		{[]r{{ud859.SeatsAvailable, ud859.NE, 1}}, 1},
		{[]r{{ud859.SeatsAvailable, ud859.GT, 0}}, 2},
		{[]r{{ud859.SeatsAvailable, ud859.GT, 1}}, 1},
		{[]r{{ud859.SeatsAvailable, ud859.GTE, 1}}, 2},
		{[]r{{ud859.SeatsAvailable, ud859.GT, 10}}, 0},
		{[]r{{ud859.SeatsAvailable, ud859.LT, 10}}, 1},
		{[]r{{ud859.SeatsAvailable, ud859.LTE, 10}}, 2},
		//
		{[]r{{ud859.StartDate, ud859.GTE, "2016-10-01T23:00:00Z"},
			{ud859.StartDate, ud859.LTE, "2016-10-31T23:00:00Z"}}, 1},
		{[]r{{ud859.StartDate, ud859.GTE, "2016-01-01T23:00:00Z"}}, 2},
		{[]r{{ud859.StartDate, ud859.GTE, "2017-01-01T23:00:00Z"}}, 0},
		//
		{[]r{{ud859.City, ud859.EQ, "Paris"},
			{ud859.StartDate, ud859.GT, "2016-10-01T23:00:00Z"}}, 1},
		{[]r{{ud859.City, ud859.EQ, "Paris"},
			{ud859.StartDate, ud859.GT, "2016-11-01T23:00:00Z"}}, 0},
		{[]r{{ud859.City, ud859.EQ, "Paris"},
			{ud859.Topics, ud859.EQ, "Go"},
			{ud859.StartDate, ud859.GT, "2016-10-01T23:00:00Z"}}, 1},
		{[]r{{ud859.City, ud859.EQ, "Paris"},
			{ud859.Topics, ud859.EQ, "Go"},
			{ud859.StartDate, ud859.GT, "2016-11-01T23:00:00Z"}}, 0},
		//
		{[]r{{ud859.Topics, ud859.EQ, "Go"},
			{ud859.StartDate, ud859.GT, "2016-01-01T23:00:00Z"}}, 2},
	}

	for _, tt := range tts {
		tt := tt

		t.Run(tt.restrictions[0].Field, func(t *testing.T) {
			t.Parallel()

			// build the query
			query := new(ud859.ConferenceQueryForm)
			for _, r := range tt.restrictions {
				query.Filter(r.Field, r.Op, r.Value)
			}

			w, err := c.do("/ConferenceAPI.QueryConferences", query)
			if err != nil {
				t.Fatal(err)
			}
			if w.Code != http.StatusOK {
				t.Fatalf("got:%d, want:%d", w.Code, http.StatusOK)
			}

			// decode the conferences
			conferences := new(ud859.Conferences)
			err = json.NewDecoder(w.Body).Decode(conferences)
			if err != nil {
				t.Fatal(err)
			}
			if len(conferences.Items) != tt.expected {
				t.Errorf("got:%d, want:%d", len(conferences.Items), tt.expected)
			}
		})
	}
}

// registration

func gotoConferences(c *client, t *testing.T) {
	t.Run("BadConference", withClient(c, gotoUnknown))
	t.Run("NotRegistered", withClient(c, gotoUnRegistered))
	t.Run("Register", withClient(c, gotoRegistration))
	t.Run("Conflict", withClient(c, gotoConflict))
}

func gotoUnknown(c *client, t *testing.T) {
	key := &ud859.ConferenceKeyForm{WebsafeKey: "foo"}

	tts := []struct {
		email  string
		status int
	}{
		{"", http.StatusUnauthorized},
		{emailTest, http.StatusBadRequest},
	}

	for _, tt := range tts {
		// register
		w, err := c.doAs(tt.email, "/ConferenceAPI.GotoConference", key)
		if err != nil {
			t.Fatal(err)
		}
		if w.Code != tt.status {
			t.Errorf("got:%d, want:%d", w.Code, tt.status)
		}

		// unregister
		w, err = c.doAs(tt.email, "/ConferenceAPI.CancelConference", key)
		if err != nil {
			t.Fatal(err)
		}
		if w.Code != tt.status {
			t.Errorf("got:%d, want:%d", w.Code, tt.status)
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
		t.Fatalf("got:%d, want:%d", w.Code, http.StatusOK)
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
			test{emailTest, c.WebsafeKey, http.StatusConflict})
	}

	for _, tt := range tts {
		key := &ud859.ConferenceKeyForm{WebsafeKey: tt.key}

		// unregister
		w, err = c.doAs(tt.email, "/ConferenceAPI.CancelConference", key)
		if err != nil {
			t.Fatal(err)
		}
		if w.Code != tt.status {
			t.Errorf("got:%d, want:%d", w.Code, tt.status)
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
		t.Fatalf("got:%d, want:%d", w.Code, http.StatusOK)
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
		key := &ud859.ConferenceKeyForm{WebsafeKey: conference.WebsafeKey}

		// register
		w, err = c.doID("/ConferenceAPI.GotoConference", key)
		if err != nil {
			t.Fatal(err)
		}
		if w.Code != http.StatusOK {
			t.Errorf("got:%d, want:%d", w.Code, http.StatusOK)
		}

		// register twice
		w, err = c.doID("/ConferenceAPI.GotoConference", key)
		if err != nil {
			t.Fatal(err)
		}
		if w.Code != http.StatusConflict {
			t.Errorf("got:%d, want:%d", w.Code, http.StatusConflict)
		}

		verifyConferencesToAttend(c, t, i+1)
	}

	for i, conference := range conferences.Items {
		key := &ud859.ConferenceKeyForm{WebsafeKey: conference.WebsafeKey}

		// unregister
		w, err = c.doID("/ConferenceAPI.CancelConference", key)
		if err != nil {
			t.Fatal(err)
		}
		if w.Code != http.StatusOK {
			t.Errorf("got:%d, want:%d", w.Code, http.StatusOK)
		}

		// unregister twice
		w, err = c.doID("/ConferenceAPI.CancelConference", key)
		if err != nil {
			t.Fatal(err)
		}
		if w.Code != http.StatusConflict {
			t.Errorf("got:%d, want:%d", w.Code, http.StatusConflict)
		}

		verifyConferencesToAttend(c, t, len(conferences.Items)-i-1)
	}
}

func gotoConflict(c *client, t *testing.T) {
	query := new(ud859.ConferenceQueryForm).
		Filter(ud859.SeatsAvailable, ud859.EQ, 1)

	// query conferences
	w, err := c.do("/ConferenceAPI.QueryConferences", query)
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != http.StatusOK {
		t.Fatalf("got:%d, want:%d", w.Code, http.StatusOK)
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

	for _, conference := range conferences.Items {
		key := &ud859.ConferenceKeyForm{WebsafeKey: conference.WebsafeKey}

		wg := new(sync.WaitGroup)
		wg.Add(2)
		codes := make(chan int, 2)

		for _, user := range []string{"user1", "user2"} {
			go func(email string) {
				defer wg.Done()

				// register user
				w, err := c.doAs(email, "/ConferenceAPI.GotoConference", key)
				if err != nil {
					t.Error(err)
					codes <- 0
				} else {
					codes <- w.Code
				}
			}(user)
		}
		wg.Wait()
		close(codes)

		// expected errors:
		// 	StatusConflict: "ud859: no seats available"
		// 	StatusBadRequest: "API error 2 (datastore_v3: CONCURRENT_TRANSACTION): Concurrency exception."
		ok, nok := <-codes, <-codes
		if (ok*nok != http.StatusOK*http.StatusConflict) &&
			(ok*nok != http.StatusOK*http.StatusBadRequest) {
			t.Fatalf("got:%d,%d", ok, nok)
		}

		// get conference
		w, err = c.do("/ConferenceAPI.GetConference", key)
		if err != nil {
			t.Fatal(err)
		}
		if w.Code != http.StatusOK {
			t.Fatalf("got:%d, want:%d", w.Code, http.StatusOK)
		}

		// decode the conference
		conference2 := new(ud859.Conference)
		err = json.NewDecoder(w.Body).Decode(conference2)
		if err != nil {
			t.Fatal(err)
		}
		if conference2.SeatsAvailable != 0 {
			t.Errorf("got:%d, want:0", conference2.SeatsAvailable)
		}
	}
}

func verifyConferencesToAttend(c *client, t *testing.T, count int) {
	// query conferences to attend
	w, err := c.doID("/ConferenceAPI.ConferencesToAttend", nil)
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != http.StatusOK {
		t.Fatalf("got:%d, want:%d", w.Code, http.StatusOK)
	}

	// decode the conferences
	conferences := new(ud859.Conferences)
	err = json.NewDecoder(w.Body).Decode(conferences)
	if err != nil {
		t.Fatal(err)
	}
	if len(conferences.Items) != count {
		t.Errorf("got:%d, want:%d", len(conferences.Items), count)
	}

	// get the profile
	w, err = c.doID("/ConferenceAPI.GetProfile", nil)
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != http.StatusOK {
		t.Fatalf("got:%d, want:%d", w.Code, http.StatusOK)
	}

	// decode the profile
	profile := new(ud859.Profile)
	err = json.NewDecoder(w.Body).Decode(profile)
	if err != nil {
		t.Fatal(err)
	}

	// verify the profile
	if len(profile.Conferences) != count {
		t.Errorf("got:%d, want:%d", len(profile.Conferences), count)
	}
	for _, conference := range conferences.Items {
		if !profile.IsRegistered(conference.WebsafeKey) {
			t.Errorf("should be registered to:%s", conference.WebsafeKey)
		}
		verifyIndexedConference(c, t, conference)
	}
}

func verifyIndexedConference(c *client, t *testing.T, conference *ud859.Conference) {
	// build the query
	query := new(ud859.ConferenceQueryForm).
		Filter("KEY", ud859.EQ, conference.WebsafeKey)

	w, err := c.do("/ConferenceAPI.QueryConferences", query)
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != http.StatusOK {
		t.Fatalf("got:%d, want:%d", w.Code, http.StatusOK)
	}

	// decode the conferences
	conferences := new(ud859.Conferences)
	err = json.NewDecoder(w.Body).Decode(conferences)
	if err != nil {
		t.Fatal(err)
	}

	if len(conferences.Items) != 1 {
		t.Fatalf("got:%d, want:%d", len(conferences.Items), 1)
	}

	if !reflect.DeepEqual(conferences.Items[0], conference) {
		t.Errorf("got: %+v", conferences.Items[0])
		t.Fatalf("want:%+v", conference)
	}
}
