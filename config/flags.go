package config

import (
	"flag"
	"io"
)

func ParseArgs(w io.Writer, args []string) (Config, *flag.FlagSet, int, error) {
	c := Config{}
	fs := flag.NewFlagSet("config", flag.ContinueOnError)

	fs.SetOutput(w)
	fs.StringVar(&c.ConfigFileLocation, "config", "", "The file to load for configuration variables")

	// Database
	fs.StringVar(&c.Database.Host, "db.host", "", "The PostgreSQL hostname for the indexer db")
	fs.StringVar(&c.Database.Database, "db.database", "", "The PostgreSQL database for the indexer db")
	fs.StringVar(&c.Database.Port, "db.port", "5432", "The PostgreSQL port for the indexer db")
	fs.StringVar(&c.Database.Password, "db.password", "", "The PostgreSQL user password for the indexer db")
	fs.StringVar(&c.Database.User, "db.user", "", "The PostgreSQL user for the indexer db")

	// Client
	fs.StringVar(&c.Client.Model, "client.model", "", "The client deployment model (commercial or not)")

	// Service
	var svcPort int
	fs.IntVar(&svcPort, "port", 8080, "the port the UI client will be served from")

	err := fs.Parse(args)
	if err != nil {
		return c, fs, svcPort, err
	}

	return c, fs, svcPort, nil
}
