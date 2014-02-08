package leaderboard

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"menteslibres.net/gosexy/to"

	"appengine"
	"appengine/datastore"
)

const (
	DSNAME = "Leaderboard"
)

type LeaderboardEntry struct {
	Name  string
	Score uint64
	When  time.Time
}

type Response struct {
	ThisPage   int
	TotalPages int
	Entries    []LeaderboardEntry
}

func dskey(c appengine.Context) *datastore.Key {
	return datastore.NewKey(c, DSNAME, "def", 0, nil)
}

func get_method(w http.ResponseWriter, r *http.Request) {
	page := r.URL.Query().Get("page")
	if page != "" {
		page = "1"
	}

	ipage := int(to.Int64(page))

	c := appengine.NewContext(r)

	q := datastore.NewQuery(DSNAME).Ancestor(dskey(c))

	cnt, err := q.Count(c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	totalpages := cnt / 50
	if cnt%50 != 0 {
		totalpages++
	}

	if ipage > totalpages {
		ipage = totalpages
	}

	if ipage <= 0 {
		ipage = 1
	}

	query := q.Order("-Score").Order("-Date").Limit(50).Offset((ipage - 1) * 50)

	entries := make([]LeaderboardEntry, 0, 50)

	if _, err := query.GetAll(c, &entries); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := Response{
		ThisPage:   ipage,
		TotalPages: totalpages,
		Entries:    entries,
	}

	bte, err := json.Marshal(&resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	fmt.Fprint(w, string(bte))
}

func post_method(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	g := LeaderboardEntry{
		Name:  r.FormValue("name"),
		Score: to.Uint64(r.FormValue("score")),
		When:  time.Now(),
	}

	if g.Score < 0 {
		panic("Bad score")
	}

	key := datastore.NewIncompleteKey(c, DSNAME, dskey(c))

	_, err := datastore.Put(c, key, &g)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, "OK")
}

func init() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				http.Error(w, fmt.Sprintf("Invalid Request: %s", r), http.StatusBadRequest)
				return
			}
		}()
		switch r.Method {
		case "POST":
			post_method(w, r)
		case "GET":
			get_method(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
}
