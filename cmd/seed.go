package cmd

import (
	"context"
	"log"

	"github.com/masudur-rahman/kazi-ancestry/configs"
	"github.com/masudur-rahman/kazi-ancestry/services/all"

	"github.com/spf13/cobra"
)

var reseed bool

// seedCmd initializes (or regenerates) the family tree in Postgres.
var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Initialize the family tree in Postgres (or -reseed to regenerate ids)",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		if err := configs.InitiateDatabaseConnection(ctx); err != nil {
			log.Fatalln(err)
		}

		svc := all.GetServices().Person
		path := configs.KaziConfig.SeedPath

		var (
			n   int
			err error
		)
		if reseed {
			n, err = svc.Reseed(path)
		} else {
			n, err = svc.Seed(path)
		}
		if err != nil {
			log.Fatalln(err)
		}
		log.Printf("person table ready: %d people", n)
	},
}

func init() {
	seedCmd.Flags().BoolVar(&reseed, "reseed", false, "drop existing rows and reimport (regenerate ids)")
	rootCmd.AddCommand(seedCmd)
}
