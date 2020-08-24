package main

import (
	"fmt"
	"github.com/hugh-hefnerd/mongoBackup/backup"
	"github.com/hugh-hefnerd/mongoBackup/providers"
	flag "github.com/spf13/pflag"
	"go.mongodb.org/mongo-driver/mongo"
	"log"
)

var (
	command    string
	host       string
	port       string
	dbName     string
	username   string
	password   string
	backupName string
	mc         *mongo.Client
	provider   *backup.MongoProvider
	text       []byte
	key        []byte
)

func init() {
	flag.StringVar(&command, "command", "", "Choose from dump, restore, or query")
	flag.StringVar(&host, "host", "localhost", "Input hostname: defaults to localhost")
	flag.StringVar(&port, "port", "27107", "Input port: defaults to 27107")
	flag.StringVar(&dbName, "db", "", "Database name required: defaults to test")
	flag.StringVar(&username, "user", "", "Username to connect with")
	flag.StringVar(&password, "pass", "", "Password for database")
	flag.StringVar(&backupName, "backupname", "", "Backup path for database")
	flag.Parse()
}

func main() {
	provider = backup.NewMongoClient(host, port, dbName, username, password, backupName)
	cmd := command
	switch cmd {
	case string(providers.CommandDump):
		log.Println("Dump the db")
		provider.DbBackup()
	case string(providers.CommandRestore):
		log.Println("Restore the db")
		provider.DbRestore()
	case string(providers.CommandQuery):
		log.Println("Query for backups")
		provider.DbBackupQuery()
	default:
		fmt.Println("A command is required: dump, restore, or query")
	}
	log.Printf("The Mongo %s command has completed.", cmd)
}
