package main

import (
	"bitbucket.org/barnardn/keymaster"
	"flag"
	"fmt"
	"github.com/jinzhu/gorm"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"net/http"
	"os"
)

var (
	db          gorm.DB
	dbPathFlag  string
	logfileFlag string
)

const (
	databaseFilename string = "./keymaster-store.sqlite3"
	logPath          string = "./keymaster-server.log"
)

func init() {

	flag.StringVar(&dbPathFlag, "dbpath", databaseFilename, "path to the keymaster sqlite3 file")
	flag.StringVar(&logfileFlag, "logpath", logPath, "path to the keymaster server log file")

}

func main() {

	flag.Parse()

	var dbpath = databaseFilename
	if dbPathFlag != "" {
		dbpath = databaseFilename
	}
	lp := logPath
	if logfileFlag != "" {
		lp = logfileFlag
	}
	logf, ferr := os.OpenFile(lp, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if ferr != nil {
		panic(ferr)
	}
	logger := log.New(logf, "", log.LstdFlags)

	var err error
	var createTables bool = false
	if _, err = os.Stat(dbpath); err != nil {
		createTables = true
	}
	db, err = gorm.Open("sqlite3", dbpath)
	if err != nil {
		log.Fatal(err)
	}
	if createTables == true {
		db.CreateTable(keymaster.AppIdentifier{})
		db.CreateTable(keymaster.AppKey{})
		db.CreateTable(keymaster.Credentials{})
	}

	logger.Println("keymaster started")
	http.HandleFunc("/", rootHandler)
	mux := http.NewServeMux()
	mux.Handle("/api/credentials", keymaster.CredentialsHandler(db, logger))
	mux.Handle("/api/credentials/", keymaster.CredentialsHandler(db, logger))
	mux.Handle("/api/apps", keymaster.AppIdentifierHandler(db, logger))
	logger.Fatal(http.ListenAndServe(":8080", mux))

}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Dump out API instructions when accessing root page")
}
