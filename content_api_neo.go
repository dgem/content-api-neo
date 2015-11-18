package main

import (
	"encoding/json"
	"fmt"
	"github.com/Financial-Times/up-neoutil-go"
	"github.com/gorilla/mux"
	"github.com/jmcvetta/neoism"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
)

func main() {

	neoURL := os.Getenv("NEO_URL")
	if neoURL == "" {
		log.Println("no $NEO_URL set, defaulting to local")
		neoURL = "http://localhost:7474/db/data"
	}
	log.Printf("connecting to %s\n", neoURL)

	var err error
	db, err = neoism.Connect(neoURL)
	if err != nil {
		panic(err)
	}

	neoutil.EnsureIndexes(db, map[string]string{
		"Content": "uuid",
		"Article": "uuid",
		"Image":   "uuid",
		"Brand":   "uuid",
	})

	port := 8080

	m := mux.NewRouter()
	http.Handle("/", m)

	m.HandleFunc("/content/{uuid}", writeHandler).Methods("PUT")
	m.HandleFunc("/content/", allWriteHandler).Methods("PUT")

	go func() {
		log.Printf("listening on %d", port)
		if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
			log.Printf("web stuff failed: %v\n", err)
		}
	}()

	bw = neoutil.NewBatchWriter(db)

	// wait for ctrl-c
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	close(bw.WriteQueue)
	<-bw.Closed
	println("exiting")

}

var db *neoism.Database

var bw *neoutil.BatchWriter

func allWriteHandler(w http.ResponseWriter, r *http.Request) {

	dec := json.NewDecoder(r.Body)

	for {
		var o content
		err := dec.Decode(&o)
		if err == io.ErrUnexpectedEOF {
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		bw.WriteQueue <- toQueries(o)
	}

	w.WriteHeader(http.StatusAccepted)
}

func writeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uuid := vars["uuid"]

	var m content
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&m)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if m.UUID != uuid {
		fmt.Printf("%v\n", m)
		http.Error(w, fmt.Sprintf("id does not match: '%v' '%v'", m.UUID, uuid), http.StatusBadRequest)
		return
	}

	bw.WriteQueue <- toQueries(m)

	w.WriteHeader(http.StatusAccepted)
}

func toQueries(c content) []*neoism.CypherQuery {
	if c.Body != "" {
		return toQueriesArticle(c)
	}
	log.Printf("skipping non-article content %v\n", c.UUID)
	return []*neoism.CypherQuery{}
}

func toQueriesArticle(m content) []*neoism.CypherQuery {
	props := toProps(m)

	var queries []*neoism.CypherQuery

	queries = append(queries, &neoism.CypherQuery{
		Statement: `
			MERGE (c:Content {uuid: {uuid}})
			SET c = {props}
			SET c :Article
		`,
		Parameters: map[string]interface{}{
			"uuid":  m.UUID,
			"props": props,
		},
	})

	for _, b := range m.Brands {
		queries = append(queries, &neoism.CypherQuery{
			Statement: `
				MERGE (c:Content {uuid: {cuuid}})
				MERGE (b:Concept {uuid: {buuid}})
				MERGE (c)-[r:HAS_BRAND]->(b)
				SET b :Brand
			`,
			Parameters: map[string]interface{}{
				"cuuid": m.UUID,
				"buuid": uriToUUID(b.ID),
			},
		})
	}

	if m.MainImage != "" {
		queries = append(queries, &neoism.CypherQuery{
			Statement: `
				MERGE (c:Content {uuid: {cuuid}})
				MERGE (i:Content {uuid: {iuuid}})
				MERGE (c)-[r:HAS_MAINIMAGE]->(i)
				SET i :Image
			`,
			Parameters: map[string]interface{}{
				"cuuid": m.UUID,
				"iuuid": m.MainImage,
			},
		})
	}
	return queries
}

func uriToUUID(uri string) string {
	// TODO: make this more robust
	return strings.Replace(uri, "http://api.ft.com/things/", "", 1)
}

func toProps(m content) neoism.Props {
	p := map[string]interface{}{
		"uuid":          m.UUID,
		"headline":      m.Title,
		"title":         m.Title,
		"prefLabel":     m.Title,
		"body":          m.Body,
		"byline":        m.Byline,
		"publishedDate": m.PublishedDate,
	}

	return neoism.Props(p)
}
