package main

import (
	"time"
)

// TODO: this is really just article right now.
type content struct {
	Body   string `json:"body"`
	Brands []struct {
		ID string `json:"id"`
	} `json:"brands"`
	Byline   string `json:"byline"`
	Comments struct {
		Enabled bool `json:"enabled"`
	} `json:"comments"`
	Description interface{} `json:"description"`
	Identifiers []struct {
		Authority       string `json:"authority"`
		IdentifierValue string `json:"identifierValue"`
	} `json:"identifiers"`
	MainImage        string    `json:"mainImage"`
	PublishReference string    `json:"publishReference"`
	PublishedDate    time.Time `json:"publishedDate"`
	Title            string    `json:"title"`
	UUID             string    `json:"uuid"`
}
