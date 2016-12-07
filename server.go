// Package ud859 is an implementation of the udacity course at http://udacity.com/course/ud859.
package ud859

import "github.com/GoogleCloudPlatform/go-endpoints/endpoints"

const clientID = "YOUR-CLIENT-ID"

var (
	scopes    = []string{endpoints.EmailScope}
	clientIds = []string{clientID, endpoints.APIExplorerClientID}
	audiences = []string{clientID}
)

// ConferenceAPI defines the conferences management API.
type ConferenceAPI struct{}

func init() {
	server := endpoints.NewServer("")
	if err := RegisterConferenceAPI(server); err != nil {
		panic(err)
	}
	server.HandleHTTP(nil)
}

// RegisterConferenceAPI adds the ConferenceAPI to the server.
func RegisterConferenceAPI(server *endpoints.Server) error {
	api, err := server.RegisterService(
		new(ConferenceAPI), "conference", "v1", "Conference Central", true)
	if err != nil {
		return err
	}

	register := func(orig, name, method, path string) *endpoints.MethodInfo {
		info := api.MethodByName(orig).Info()
		info.Name, info.HTTPMethod, info.Path = name, method, path
		return info
	}
	login := func(orig, name, method, path string) {
		info := register(orig, name, method, path)
		info.Scopes, info.ClientIds, info.Audiences = scopes, clientIds, audiences
	}

	// profile
	login("GetProfile", "getProfile", "GET", "profile")
	login("SaveProfile", "saveProfile", "POST", "profile")

	// conference
	register("GetConference", "getConference", "GET", "conference/{websafeConferenceKey}")
	login("CreateConference", "createConference", "POST", "conference")

	// query conferences
	login("ConferencesCreated", "getConferencesCreated", "POST", "getConferencesCreated")
	login("ConferencesToAttend", "getConferencesToAttend", "GET", "getConferencesToAttend")
	register("QueryConferences", "queryConferences", "POST", "queryConferences")

	// registration
	login("GotoConference", "registerForConference", "POST", "conference/{websafeConferenceKey}/registration")
	login("CancelConference", "unregisterFromConference", "DELETE", "conference/{websafeConferenceKey}/registration")

	return nil
}
