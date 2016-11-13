// +build go1.7

package ud859_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/schorlet/ud859"

	"google.golang.org/appengine/aetest"
)

type clientFunc func(string, string, string, interface{}) *httptest.ResponseRecorder
type requestFunc func(clientFunc, *testing.T)

func withClient(inst aetest.Instance, fn requestFunc) func(*testing.T) {
	return func(t *testing.T) {
		client := func(email, method, url string, v interface{}) *httptest.ResponseRecorder {
			var body io.Reader

			if v != nil {
				b, err := json.Marshal(v)
				if err != nil {
					t.Fatal(err)
				}
				body = bytes.NewReader(b)
			}

			r, err := inst.NewRequest(method, url, body)
			if err != nil {
				_ = inst.Close()
				t.Fatal(err)
			}
			if email != "" {
				r.Header.Set("X-AppEngine-User-Email", email)
			}

			w := httptest.NewRecorder()
			ud859.ConferenceHandler(w, r)
			return w
		}

		fn(client, t)
	}
}

func TestHandler(t *testing.T) {
	// start the development server
	// options := &aetest.Options{StronglyConsistentDatastore: true}
	inst, err := aetest.NewInstance(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer inst.Close()

	t.Run("getProfile", withClient(inst, getProfile))
	t.Run("postProfile", withClient(inst, postProfile))

	t.Run("getConference", withClient(inst, getConference))
	t.Run("postConference", withClient(inst, postConference))
	t.Run("conferencesWebQuery", withInstance(inst, conferencesWebQuery))
	t.Run("conferenceWebGoto", withClient(inst, conferenceWebGoto))
}

// profile

func getProfile(do clientFunc, t *testing.T) {
	w := do("", "GET", "/conference/v1/profile", nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want:%d, got:%d", http.StatusUnauthorized, w.Code)
	}

	w = do("ud859@udacity.com", "GET", "/conference/v1/profile", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("want:%d, got:%d", http.StatusOK, w.Code)
	}

	// decode the profile
	profile := new(ud859.Profile)
	decoder := json.NewDecoder(w.Body)
	err := decoder.Decode(profile)
	if err != nil {
		t.Fatal(err)
	}

	// verify the profile
	if profile.Email != "" {
		t.Errorf("want:empty, got:%s", profile.Email)
	}
	form := new(ud859.ProfileForm)
	verifyWebProfile(t, profile, form)
}

func postProfile(do clientFunc, t *testing.T) {
	form := &ud859.ProfileForm{
		DisplayName:  "ud859",
		TeeShirtSize: ud859.SIZE_XL,
	}
	w := do("ud859@udacity.com", "POST", "/conference/v1/profile", form)
	if w.Code != http.StatusCreated {
		t.Fatalf("want:%d, got:%d", http.StatusCreated, w.Code)
	}

	w = do("ud859@udacity.com", "GET", "/conference/v1/profile", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("want:%d, got:%d", http.StatusOK, w.Code)
	}

	// decode the profile
	profile := new(ud859.Profile)
	decoder := json.NewDecoder(w.Body)
	err := decoder.Decode(profile)
	if err != nil {
		t.Fatal(err)
	}

	// verify the profile
	if profile.Email != "" {
		t.Errorf("want:empty, got:%s", profile.Email)
	}
	verifyWebProfile(t, profile, form)
}

// conference

func getConference(do clientFunc, t *testing.T) {
	w := do("", "GET", "/conference/v1/conference", nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want:%d, got:%d", http.StatusBadRequest, w.Code)
	}
	w = do("", "GET", "/conference/v1/conference/foo", nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want:%d, got:%d", http.StatusBadRequest, w.Code)
	}
}

func postConference(do clientFunc, t *testing.T) {
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
		// try to create the conference without a user
		w := do("", "POST", "/conference/v1/conference", form)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("want:%d, got:%d", http.StatusUnauthorized, w.Code)
		}

		// create the conference
		w = do("ud859@udacity.com", "POST", "/conference/v1/conference", form)
		if w.Code != http.StatusCreated {
			t.Fatalf("want:%d, got:%d", http.StatusCreated, w.Code)
		}

		// get the conference
		websafeKey := w.Body.String()
		if websafeKey == "" {
			t.Fatal("websafeKey is empty")
		}
		w = do("", "GET", "/conference/v1/conference/"+websafeKey, nil)
		if w.Code != http.StatusOK {
			t.Fatalf("want:%d, got:%d", http.StatusOK, w.Code)
		}

		// decode the conference
		conference := new(ud859.Conference)
		decoder := json.NewDecoder(w.Body)
		err := decoder.Decode(conference)
		if err != nil {
			t.Fatal(err)
		}

		// verify the conference
		if conference.ID != 0 {
			t.Errorf("want:empty, got:%d", conference.ID)
		}
		verifyWebConference(t, conference, form)
	}
}

// conferencesQuery

func conferencesWebQuery(inst aetest.Instance, t *testing.T) {
	// run all sub-tests in parallel
	t.Run("Nofilters", withClient(inst, webQueryNofilters))
	t.Run("City", withClient(inst, webQueryCity))
	t.Run("Topics", withClient(inst, webQueryTopics))
	t.Run("MaxAttendees", withClient(inst, webQueryMaxAttendees))
	t.Run("StartDate", withClient(inst, webQueryStartDate))
	t.Run("Created", withClient(inst, webQueryCreated))
	t.Run("CreatedUser1", withClient(inst, webQueryCreatedUser1))
}

func webQueryNofilters(do clientFunc, t *testing.T) {
	t.Parallel()

	w := do("", "POST", "/conference/v1/queryConferences_nofilters", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("want:%d, got:%d", http.StatusOK, w.Code)
	}

	// decode the conferences
	conferences := new(ud859.Conferences)
	decoder := json.NewDecoder(w.Body)
	err := decoder.Decode(conferences)
	if err != nil {
		t.Fatal(err)
	}
	if len(conferences.Items) != 2 {
		t.Errorf("want:2, got:%d", len(conferences.Items))
	}
}

func webQueryCity(do clientFunc, t *testing.T) {
	t.Parallel()

	// City = Paris
	query := new(ud859.ConferenceQueryForm).
		Filter(ud859.City, ud859.EQ, "Paris")

	w := do("", "POST", "/conference/v1/queryConferences", query)
	if w.Code != http.StatusOK {
		t.Fatalf("want:%d, got:%d", http.StatusOK, w.Code)
	}

	// decode the conferences
	conferences := new(ud859.Conferences)
	decoder := json.NewDecoder(w.Body)
	err := decoder.Decode(conferences)
	if err != nil {
		t.Fatal(err)
	}
	if len(conferences.Items) != 1 {
		t.Errorf("want:1, got:%d", len(conferences.Items))
	}

	// City = London
	query = new(ud859.ConferenceQueryForm).
		Filter(ud859.City, ud859.EQ, "London")

	w = do("", "POST", "/conference/v1/queryConferences", query)
	if w.Code != http.StatusOK {
		t.Fatalf("want:%d, got:%d", http.StatusOK, w.Code)
	}

	// decode the conferences
	conferences = new(ud859.Conferences)
	decoder = json.NewDecoder(w.Body)
	err = decoder.Decode(conferences)
	if err != nil {
		t.Fatal(err)
	}
	if len(conferences.Items) != 0 {
		t.Errorf("want:0, got:%d", len(conferences.Items))
	}
}

func webQueryTopics(do clientFunc, t *testing.T) {
	t.Parallel()

	query := new(ud859.ConferenceQueryForm).
		Filter(ud859.Topics, ud859.GTE, "Go").
		Filter(ud859.Topics, ud859.LTE, "Go")

	w := do("", "POST", "/conference/v1/queryConferences", query)
	if w.Code != http.StatusOK {
		t.Fatalf("want:%d, got:%d", http.StatusOK, w.Code)
	}

	// decode the conferences
	conferences := new(ud859.Conferences)
	decoder := json.NewDecoder(w.Body)
	err := decoder.Decode(conferences)
	if err != nil {
		t.Fatal(err)
	}
	if len(conferences.Items) != 2 {
		t.Errorf("want:2, got:%d", len(conferences.Items))
	}
}

func webQueryMaxAttendees(do clientFunc, t *testing.T) {
	t.Parallel()

	// 1 < MaxAttendees
	query := new(ud859.ConferenceQueryForm).
		Filter(ud859.MaxAttendees, ud859.GT, 1)

	w := do("", "POST", "/conference/v1/queryConferences", query)
	if w.Code != http.StatusOK {
		t.Fatalf("want:%d, got:%d", http.StatusOK, w.Code)
	}

	// decode the conferences
	conferences := new(ud859.Conferences)
	decoder := json.NewDecoder(w.Body)
	err := decoder.Decode(conferences)
	if err != nil {
		t.Fatal(err)
	}
	if len(conferences.Items) != 1 {
		t.Logf("%+v", conferences)
		t.Errorf("want:1, got:%d", len(conferences.Items))
	}

	// MaxAttendees < 10
	query = new(ud859.ConferenceQueryForm).
		Filter(ud859.MaxAttendees, ud859.LT, 10)

	w = do("", "POST", "/conference/v1/queryConferences", query)
	if w.Code != http.StatusOK {
		t.Fatalf("want:%d, got:%d", http.StatusOK, w.Code)
	}

	// decode the conferences
	conferences = new(ud859.Conferences)
	decoder = json.NewDecoder(w.Body)
	err = decoder.Decode(conferences)
	if err != nil {
		t.Fatal(err)
	}
	if len(conferences.Items) != 2 {
		t.Errorf("want:2, got:%d", len(conferences.Items))
	}
}

func webQueryStartDate(do clientFunc, t *testing.T) {
	t.Parallel()

	// 2016-10-01 <= Start <= 2016-10-31
	query := new(ud859.ConferenceQueryForm).
		Filter(ud859.StartDate, ud859.GTE, "2016-10-01").
		Filter(ud859.StartDate, ud859.LTE, "2016-10-31")

	w := do("", "POST", "/conference/v1/queryConferences", query)
	if w.Code != http.StatusOK {
		t.Fatalf("want:%d, got:%d", http.StatusOK, w.Code)
	}

	// decode the conferences
	conferences := new(ud859.Conferences)
	decoder := json.NewDecoder(w.Body)
	err := decoder.Decode(conferences)
	if err != nil {
		t.Fatal(err)
	}
	if len(conferences.Items) != 1 {
		t.Errorf("want:1, got:%d", len(conferences.Items))
		t.Logf("%+v", conferences)
	}

	// 2016-01-01 <= Start
	query = new(ud859.ConferenceQueryForm).
		Filter(ud859.StartDate, ud859.GTE, "2016-01-01")

	w = do("", "POST", "/conference/v1/queryConferences", query)
	if w.Code != http.StatusOK {
		t.Fatalf("want:%d, got:%d", http.StatusOK, w.Code)
	}

	// decode the conferences
	conferences = new(ud859.Conferences)
	decoder = json.NewDecoder(w.Body)
	err = decoder.Decode(conferences)
	if err != nil {
		t.Fatal(err)
	}
	if len(conferences.Items) != 2 {
		t.Errorf("want:2, got:%d", len(conferences.Items))
		t.Logf("%+v", conferences)
	}

	// 2017-01-01 <= Start
	query = new(ud859.ConferenceQueryForm).
		Filter(ud859.StartDate, ud859.GTE, "2017-01-01")

	w = do("", "POST", "/conference/v1/queryConferences", query)
	if w.Code != http.StatusOK {
		t.Fatalf("want:%d, got:%d", http.StatusOK, w.Code)
	}

	// decode the conferences
	conferences = new(ud859.Conferences)
	decoder = json.NewDecoder(w.Body)
	err = decoder.Decode(conferences)
	if err != nil {
		t.Fatal(err)
	}
	if len(conferences.Items) != 0 {
		t.Errorf("want:0, got:%d", len(conferences.Items))
	}
}

func webQueryCreated(do clientFunc, t *testing.T) {
	t.Parallel()

	w := do("ud859@udacity.com", "POST", "/conference/v1/getConferencesCreated", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("want:%d, got:%d", http.StatusOK, w.Code)
	}

	// decode the conferences
	conferences := new(ud859.Conferences)
	decoder := json.NewDecoder(w.Body)
	err := decoder.Decode(conferences)
	if err != nil {
		t.Fatal(err)
	}
	if len(conferences.Items) != 2 {
		t.Errorf("want:2, got:%d", len(conferences.Items))
	}
}

func webQueryCreatedUser1(do clientFunc, t *testing.T) {
	t.Parallel()

	w := do("user1@udacity.com", "POST", "/conference/v1/getConferencesCreated", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("want:%d, got:%d", http.StatusOK, w.Code)
	}

	// decode the conferences
	conferences := new(ud859.Conferences)
	decoder := json.NewDecoder(w.Body)
	err := decoder.Decode(conferences)
	if err != nil {
		t.Fatal(err)
	}
	if len(conferences.Items) != 0 {
		t.Errorf("want:0, got:%d", len(conferences.Items))
	}
}

// conferenceWebGoto

func conferenceWebGoto(do clientFunc, t *testing.T) {
	// cancel foo registration
	w := do("", "DELETE", "/conference/v1/conference/foo/regitration", nil)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("want:%d, got:%d", http.StatusUnauthorized, w.Code)
	}

	// register foo conference
	w = do("", "POST", "/conference/v1/conference/foo/regitration", nil)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("want:%d, got:%d", http.StatusUnauthorized, w.Code)
	}

	// query all conferences
	w = do("", "POST", "/conference/v1/queryConferences_nofilters", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("want:%d, got:%d", http.StatusOK, w.Code)
	}

	// decode the conferences
	conferences := new(ud859.Conferences)
	decoder := json.NewDecoder(w.Body)
	err := decoder.Decode(conferences)
	if err != nil {
		t.Fatal(err)
	}
	if len(conferences.Items) != 2 {
		t.Errorf("want:2, got:%d", len(conferences.Items))
	}

	// pick the dotGo conference
	conferenceID := conferences.Items[0].ID
	websafeKey := conferences.Items[0].WebsafeKey

	// cancel registration when not registered
	w = do("", "DELETE", "/conference/v1/conference/"+websafeKey+"/regitration", nil)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("want:%d, got:%d", http.StatusUnauthorized, w.Code)
	}

	// cancel registration when registered
	w = do("ud859@udacity.com", "DELETE", "/conference/v1/conference/"+websafeKey+"/regitration", nil)
	if w.Code != http.StatusForbidden {
		t.Errorf("want:%d, got:%d", http.StatusForbidden, w.Code)
	}

	// register to the conference when not registered
	w = do("", "POST", "/conference/v1/conference/"+websafeKey+"/regitration", nil)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("want:%d, got:%d", http.StatusUnauthorized, w.Code)
	}

	// register to the conference when registered
	w = do("ud859@udacity.com", "POST", "/conference/v1/conference/"+websafeKey+"/regitration", nil)
	if w.Code != http.StatusOK {
		t.Errorf("want:%d, got:%d", http.StatusOK, w.Code)
	}

	// register twice to the conference
	w = do("ud859@udacity.com", "POST", "/conference/v1/conference/"+websafeKey+"/regitration", nil)
	if w.Code != http.StatusForbidden {
		t.Errorf("want:%d, got:%d", http.StatusForbidden, w.Code)
	}

	// register with another user
	w = do("other@udacity.com", "POST", "/conference/v1/conference/"+websafeKey+"/regitration", nil)
	if w.Code != http.StatusForbidden {
		t.Errorf("want:%d, got:%d", http.StatusForbidden, w.Code)
	}

	// verify the conferences to attend
	w = do("ud859@udacity.com", "POST", "/conference/v1/getConferencesToAttend", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("want:%d, got:%d", http.StatusOK, w.Code)
	}

	// decode the conferences
	conferences = new(ud859.Conferences)
	decoder = json.NewDecoder(w.Body)
	err = decoder.Decode(conferences)
	if err != nil {
		t.Fatal(err)
	}
	if len(conferences.Items) != 1 {
		t.Errorf("want:1, got:%d", len(conferences.Items))
	}
	if conferences.Items[0].ID != conferenceID {
		t.Errorf("want:%d, got:%d", conferenceID, conferences.Items[0].ID)
	}

	// cancel registration
	w = do("ud859@udacity.com", "DELETE", "/conference/v1/conference/"+websafeKey+"/regitration", nil)
	if w.Code != http.StatusOK {
		t.Errorf("want:%d, got:%d", http.StatusOK, w.Code)
	}

	// verify there is no conference to attend
	w = do("ud859@udacity.com", "POST", "/conference/v1/getConferencesToAttend", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("want:%d, got:%d", http.StatusOK, w.Code)
	}

	// decode the conferences
	conferences = new(ud859.Conferences)
	decoder = json.NewDecoder(w.Body)
	err = decoder.Decode(conferences)
	if err != nil {
		t.Fatal(err)
	}
	if len(conferences.Items) != 0 {
		t.Errorf("want:0, got:%d", len(conferences.Items))
	}
}
