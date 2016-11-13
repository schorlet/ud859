package ud859

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"google.golang.org/appengine"
	"google.golang.org/appengine/user"
)

func ConferenceHandler(w http.ResponseWriter, r *http.Request) {
	// conference/v1
	//    profile GET POST
	//    conference/:id GET
	//    conference/:id/registration POST DELETE
	//    conference POST
	//    queryConferences_nofilters POST
	//    queryConferences POST
	//    getConferencesCreated POST
	//    getConferencesToAttend GET
	//    announcement GET

	var err error
	var paths = strings.Split(r.URL.Path[1:], "/")
	var is = func(method, path string) bool {
		return r.Method == method && paths[2] == path
	}

	switch {
	case is("GET", "profile"):
		err = handleGetProfile(w, r)
	case is("POST", "profile"):
		err = handlePostProfile(w, r)

	// conference
	case len(paths) == 4 && is("GET", "conference"):
		err = handleGetConference(w, r, paths[3])
	case len(paths) == 5 &&
		(is("POST", "conference") || is("DELETE", "conference")):
		err = handleConferencesRegistration(w, r, paths[3])
	case is("POST", "conference"):
		err = handlePostConference(w, r)

	// query
	case is("POST", "queryConferences_nofilters"):
		err = handleQueryConferences(w, r)
	case is("POST", "queryConferences"):
		err = handleQueryConferencesFilter(w, r)
	case is("POST", "getConferencesCreated"):
		err = handleConferencesCreated(w, r)
	case is("POST", "getConferencesToAttend"):
		err = handleConferencesToAttend(w, r)

	// announcement
	case is("GET", "announcement"):
		err = handleAnnouncement(w, r)
	default:
		http.Error(w, "Bad Request", http.StatusBadRequest)
	}

	if ere, ok := err.(*statusError); ok {
		http.Error(w, ere.Error(), ere.Status)
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// profile

func handleGetProfile(w http.ResponseWriter, r *http.Request) error {
	c := appengine.NewContext(r)

	// get the profile
	profile, err := GetProfile(c)
	if err != nil {
		return err
	}

	// return the profile
	encoder := json.NewEncoder(w)
	if err = encoder.Encode(profile); err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return nil
}

func handlePostProfile(w http.ResponseWriter, r *http.Request) error {
	c := appengine.NewContext(r)
	if u := user.Current(c); u == nil {
		return ErrUnauthorized
	}

	// decode the profile form
	form := new(ProfileForm)
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(form); err != nil {
		return errBadRequest(err, "unable to decode the profile form")
	}

	// save the profile
	if err := SaveProfile(c, form); err != nil {
		return err
	}

	w.WriteHeader(http.StatusCreated)
	return nil
}

func handleConferencesCreated(w http.ResponseWriter, r *http.Request) error {
	c := appengine.NewContext(r)
	if u := user.Current(c); u == nil {
		return ErrUnauthorized
	}

	conferences, err := ConferencesCreated(c)
	if err != nil {
		return err
	}
	return writeConferences(w, conferences)
}

func handleConferencesRegistration(w http.ResponseWriter, r *http.Request, websafeKey string) error {
	c := appengine.NewContext(r)
	if u := user.Current(c); u == nil {
		return ErrUnauthorized
	}

	if r.Method == "POST" {
		return GotoConference(c, websafeKey)
	} else if r.Method == "DELETE" {
		return CancelConference(c, websafeKey)
	}

	http.NotFound(w, r)
	return nil
}

func handleConferencesToAttend(w http.ResponseWriter, r *http.Request) error {
	c := appengine.NewContext(r)
	if u := user.Current(c); u == nil {
		return ErrUnauthorized
	}

	conferences, err := ConferencesToAttend(c)
	if err != nil {
		return err
	}
	return writeConferences(w, conferences)
}

// conference

func handleGetConference(w http.ResponseWriter, r *http.Request, websafeKey string) error {
	c := appengine.NewContext(r)

	conference, err := GetConference(c, websafeKey)
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(w)
	if err = encoder.Encode(conference); err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return nil
}

func handlePostConference(w http.ResponseWriter, r *http.Request) error {
	c := appengine.NewContext(r)
	if u := user.Current(c); u == nil {
		return ErrUnauthorized
	}

	// decode the conference form
	form := new(ConferenceForm)
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(form); err != nil {
		return errBadRequest(err, "unable to decode the conference form")
	}

	// save the conference
	conference, err := CreateConference(c, form)
	if err != nil {
		return err
	}

	w.WriteHeader(http.StatusCreated)
	fmt.Fprint(w, conference.WebsafeKey)
	return nil
}

// query

func handleQueryConferences(w http.ResponseWriter, r *http.Request) error {
	c := appengine.NewContext(r)

	conferences, err := QueryConferences(c)
	if err != nil {
		return err
	}
	return writeConferences(w, conferences)
}

func handleQueryConferencesFilter(w http.ResponseWriter, r *http.Request) error {
	c := appengine.NewContext(r)

	// decode the conference query form
	form := new(ConferenceQueryForm)
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(form); err != nil {
		return errBadRequest(err, "unable to decode the conference query form")
	}

	conferences, err := QueryConferencesFilter(c, form)
	if err != nil {
		return err
	}
	return writeConferences(w, conferences)
}

func writeConferences(w http.ResponseWriter, conferences []*Conference) error {
	encoder := json.NewEncoder(w)
	if err := encoder.Encode(Conferences{Items: conferences}); err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return nil
}

// announcement

func handleAnnouncement(w http.ResponseWriter, r *http.Request) error {
	fmt.Fprintf(w, "%s %s", r.Method, r.URL.Path)
	return nil
}
