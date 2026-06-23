// Command seed initializes (or regenerates) the family tree in Postgres.
//
//	go run ./cmd/seed            # create schema + seed if empty
//	go run ./cmd/seed -reseed    # drop person table + reimport (regenerate ids)
package main

import (
	"context"
	"flag"
	"log"

	"github.com/masudur-rahman/kazi-ancestry/internal/config"
	"github.com/masudur-rahman/kazi-ancestry/internal/db"
)

func main() {
	reseed := flag.Bool("reseed", false, "drop the person table and reimport (regenerate ids)")
	flag.Parse()

	ctx := context.Background()
	cfg := config.Load()

	engine, err := db.Connect(cfg)
	if err != nil {
		log.Fatal(err)
	}
	if err := db.Sync(ctx, engine); err != nil {
		log.Fatal(err)
	}

	var n int
	if *reseed {
		n, err = db.Reseed(ctx, engine, cfg.SeedPath)
	} else {
		n, err = db.Seed(ctx, engine, cfg.SeedPath)
	}
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("person table ready: %d people", n)
}
