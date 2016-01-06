package main

import (
	"github.com/Financial-Times/content-api-neo/content"
	"github.com/Financial-Times/up-neoutil-go"
	"github.com/jawher/mow.cli"
	"github.com/jmcvetta/neoism"
	"log"
	"os"
)

func main() {

	app := cli.App("content-api-neo", "A RESTful API for managing Content in neo4j")
	neoURL := app.StringOpt("neo-url", "http://localhost:7474/db/data", "neo4j endpoint URL")
	port := app.IntOpt("port", 8080, "Port to listen on")

	app.Action = func() {
		db, err := neoism.Connect(*neoURL)
		if err != nil {
			log.Fatal(err)
		}

		cr := neoutil.NewBatchWriter(db, 1024)

		engs := map[string]neoutil.NeoEngine{
			"content": content.ContentNeoEngine{cr},
		}

		neoutil.EnsureAllIndexes(db, engs)
		neoutil.RunServer(engs, *port)
	}

	app.Run(os.Args)
}
