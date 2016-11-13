package ud859

import (
	"net/http"
)

func init() {
	http.HandleFunc("/conference/v1/", ConferenceHandler)
}
