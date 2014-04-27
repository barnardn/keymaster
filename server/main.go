package main

import (
	"bitbucket.org/barnardn/keymaster"
	"fmt"
	"github.com/jinzhu/gorm"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"net/http"
	"os"
)

var (
	db gorm.DB
)

const (
	databaseFilename string = "./keymaster-store.sqlite3"
)

func main() {

	var err error
	var createTables bool = false
	if _, err = os.Stat(databaseFilename); err != nil {
		createTables = true
	}
	db, err = gorm.Open("sqlite3", databaseFilename)
	if err != nil {
		log.Fatal(err)
	}
	if createTables == true {
		db.CreateTable(keymaster.AppIdentifier{})
		db.CreateTable(keymaster.AppKey{})
		db.CreateTable(keymaster.Credentials{})
	}

	http.HandleFunc("/", rootHandler)
	mux := http.NewServeMux()
	mux.Handle("/api/credentials", keymaster.CredentialsHandler(db))
	mux.Handle("/api/credentials/", keymaster.CredentialsHandler(db))
	mux.Handle("/api/apps", keymaster.AppIdentifierHandler(db))
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Dump out API instructions when accessing root page")
}
