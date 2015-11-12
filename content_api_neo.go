package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/jmcvetta/neoism"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"
)

func main() {

	neoUrl := os.Getenv("NEO_URL")
	if neoUrl == "" {
		log.Println("no $NEO_URL set, defaulting to local")
		neoUrl = "http://localhost:7474/db/data"
	}
	log.Printf("connecting to %s\n", neoUrl)

	var err error
	db, err = neoism.Connect(neoUrl)
	if err != nil {
		panic(err)
	}

	ensureIndexes(db)

	writeQueue = make(chan content, 2048)

	port := 8080

	m := mux.NewRouter()
	http.Handle("/", m)

	m.HandleFunc("/content/{uuid}", idWriteHandler).Methods("PUT")
	m.HandleFunc("/content/", allWriteHandler).Methods("PUT")

	go func() {
		log.Printf("listening on %d", port)
		if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
			log.Printf("web stuff failed: %v\n", err)
		}
	}()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		orgWriteLoop()
		wg.Done()
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	// wait for ctrl-c
	<-c
	close(writeQueue)
	wg.Wait()
	println("exiting")

}

func ensureIndexes(db *neoism.Database) {
	ensureIndex(db, "Content", "uuid")
	ensureIndex(db, "Article", "uuid")
}

func ensureIndex(db *neoism.Database, label string, prop string) {
	indexes, err := db.Indexes(label)
	if err != nil {
		panic(err)
	}
	for _, ind := range indexes {
		if len(ind.PropertyKeys) == 1 && ind.PropertyKeys[0] == prop {
			return
		}
	}
	if _, err := db.CreateIndex(label, prop); err != nil {
		panic(err)
	}
}

var db *neoism.Database

var writeQueue chan content

func orgWriteLoop() {
	var qs []*neoism.CypherQuery

	timer := time.NewTimer(1 * time.Second)

	defer log.Printf("write loop exited")
	for {
		select {
		case o, ok := <-writeQueue:
			if !ok {
				return
			}
			for _, q := range toQueries(o) {
				qs = append(qs, q)
			}
			if len(qs) < 1024 {
				timer.Reset(1 * time.Second)
				continue
			}
		case <-timer.C:
		}
		if len(qs) > 0 {
			fmt.Printf("writing batch of %d\n", len(qs))
			err := db.CypherBatch(qs)
			if err != nil {
				panic(err)
			}
			fmt.Printf("wrote batch of %d\n", len(qs))
			qs = qs[0:0]
			timer.Stop()
		}
	}
}

func allWriteHandler(w http.ResponseWriter, r *http.Request) {

	dec := json.NewDecoder(r.Body)

	for {
		var o content
		err := dec.Decode(&o)
		if err == io.ErrUnexpectedEOF {
			println("eof")
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		writeQueue <- o
	}

	w.WriteHeader(http.StatusAccepted)
}

func idWriteHandler(w http.ResponseWriter, r *http.Request) {
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

	writeQueue <- m

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
				MERGE (b:Brand {uuid: {buuid}})
				MERGE (c)-[r:IS_BRANDED]->(b)
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
				MERGE (b:Image {uuid: {iuuid}})
				MERGE (c)-[r:HAS_MAINIMAGE]->(i)
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
		"uuid":      m.UUID,
		"headline":  m.Title,
		"title":     m.Title,
		"prefLabel": m.Title,
		"body":      m.Body,
		"byline":    m.Byline,
	}

	return neoism.Props(p)
}
