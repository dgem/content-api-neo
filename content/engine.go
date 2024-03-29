package content

import (
	"encoding/json"
	"github.com/Financial-Times/neo-cypher-runner-go"
	"github.com/jmcvetta/neoism"
	"log"
	"strings"
)

type ContentNeoEngine struct {
	Cr neocypherrunner.CypherRunner
}

func (bnc ContentNeoEngine) DecodeJSON(dec *json.Decoder) (interface{}, string, error) {
	b := Content{}
	err := dec.Decode(&b)
	return b, b.UUID, err
}

func (bnc ContentNeoEngine) SuggestedIndexes() map[string]string {
	return map[string]string{
		"Article": "uuid",
		"Content": "uuid",
		//"Concept": "uuid",  really?
		"Thing": "uuid",
	}
}

func (bnc ContentNeoEngine) Read(identity string) (interface{}, bool, error) {
	panic("not implemented")
}

func (bnc ContentNeoEngine) Write(obj interface{}) error {
	c := obj.(Content)

	if c.Body != "" {
		return bnc.createOrUpdateArticle(c)
	}
	log.Printf("skipping non-article content %v\n", c.UUID)
	return nil
}

func (eng ContentNeoEngine) createOrUpdateArticle(c Content) error {

	p := map[string]interface{}{
		"uuid":                 c.UUID,
		"headline":             c.Title,
		"title":                c.Title,
		"prefLabel":            c.Title,
		"body":                 c.Body,
		"byline":               c.Byline,
		"publishedDate":        c.PublishedDate,
		"publishedDateEpochMs": c.PublishedDate.Unix(),
	}

	var queries []*neoism.CypherQuery

	stmt := `
		MERGE (c:Thing {uuid: {uuid}})
		SET c = {props}
		SET c :Content
		SET c :Article
		`

	if c.MainImage != "" {
		stmt += `
			MERGE (i:Thing {uuid: {iuuid}})
			MERGE (c)-[r:HAS_MAINIMAGE]->(i)
			`
	}

	queries = append(queries, &neoism.CypherQuery{
		Statement: stmt,
		Parameters: map[string]interface{}{
			"uuid":  c.UUID,
			"props": neoism.Props(p),
			"iuuid": c.MainImage,
		},
	})

	for _, b := range c.Brands {
		queries = append(queries, &neoism.CypherQuery{
			Statement: `
				MERGE (c:Content {uuid: {cuuid}})
				MERGE (b:Thing {uuid: {buuid}})
				MERGE (c)-[r:HAS_BRAND]->(b)
			`,
			Parameters: map[string]interface{}{
				"cuuid": c.UUID,
				"buuid": uriToUUID(b.ID),
			},
		})
	}

	return eng.Cr.CypherBatch(queries)
}

func (bnc ContentNeoEngine) Delete(identity string) (bool, error) {
	panic("not implemented")
}

func uriToUUID(uri string) string {
	// TODO: make this more robust
	return strings.Replace(uri, "http://api.ft.com/things/", "", 1)
}
