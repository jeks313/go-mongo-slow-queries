package mongoslow

import (
	_ "embed"
	"encoding/json"
	"net/http"
	"text/template"
)

//go:embed html/queries.html
var queriesHTML string

// SlowQueryHandler will output the current running query list
func SlowQueryHandler(slow *MongoSlow) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		json.NewEncoder(w).Encode(slow.runningQueries)
	}
}

// HistoryQueryHandler will dump the ring buffer of historical slow queries
func HistoryQueryHandler(slow *MongoSlow) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		var queries []*Query
		slow.history.Do(func(p interface{}) {
			if p != nil {
				queries = append(queries, p.(*Query))
			}
		})
		json.NewEncoder(w).Encode(queries)
	}
}

// RunningQueryTableHandler will output the running queries in a datatable
func RunningQueryTableHandler(slow *MongoSlow) func(w http.ResponseWriter, r *http.Request) {
	t := template.Must(template.New("table").Parse(queriesHTML))
	return func(w http.ResponseWriter, r *http.Request) {
		var queries []*Query
		for _, query := range slow.runningQueries {
			queries = append(queries, query)
		}
		w.Header().Set("content-type", "text/html")
		j, _ := json.Marshal(queries)
		t.Execute(w, string(j))
	}
}

// HistoryQueryTableHandler will output the running queries in a datatable
func HistoryQueryTableHandler(slow *MongoSlow) func(w http.ResponseWriter, r *http.Request) {
	t := template.Must(template.New("table").Parse(queriesHTML))
	return func(w http.ResponseWriter, r *http.Request) {
		var queries []*Query
		slow.history.Do(func(p interface{}) {
			if p != nil {
				queries = append(queries, p.(*Query))
			}
		})
		w.Header().Set("content-type", "text/html")
		j, _ := json.Marshal(queries)
		t.Execute(w, string(j))
	}
}
