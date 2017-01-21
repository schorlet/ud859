package ud859

import (
	"net/http"
	"reflect"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/search"
)

func init() {
	http.HandleFunc("/clean_index", cleanIndex)
}

func cleanIndex(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	index, err := search.Open("Conference")
	if err != nil {
		log.Errorf(c, "could not open index: %v", err)
		return
	}

	it := index.List(c, nil)
	for {
		doc := new(conferenceDoc)

		id, err := it.Next(doc)
		if err != nil {
			break
		}

		key, err := datastore.DecodeKey(string(doc.WebsafeKey))
		if err != nil {
			if erd := index.Delete(c, id); erd != nil {
				log.Errorf(c, "could not delete document: %v", erd)
			}
			continue
		}

		conference, err := getConference(c, key)
		if err != nil {
			if erd := index.Delete(c, id); erd != nil {
				log.Errorf(c, "could not delete document %v", erd)
			}
			continue
		}

		if !reflect.DeepEqual(fromConferenceDoc(doc), conference) {
			doc := fromConference(conference)
			_, erp := index.Put(c, conference.WebsafeKey, doc)
			if erp != nil {
				log.Errorf(c, "could not update document %v", erp)
			}
		}
	}

	// clear cache
	_ = deleteCacheNoFilters.Call(c)
}
