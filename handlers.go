package keymaster

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jinzhu/gorm"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

var validCommands = map[string]bool{
	"recypher": true,
}

type credentialsHandler struct {
	db gorm.DB
}

type errorMessage struct {
	Message string
}

func CredentialsHandler(db gorm.DB) http.Handler {
	return &credentialsHandler{db}
}

func (credHandler *credentialsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	switch strings.ToLower(r.Method) {
	case "post":
		storeNewCredentials(credHandler, w, r)
	case "get":
		retrieveCredentials(credHandler, w, r)
	default:
		w.WriteHeader(404)
	}
}

func storeNewCredentials(ch *credentialsHandler, w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")

	jsonRequest, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(500)
		w.Write(errorResponse(err))
		return
	}
	var credentials Credentials
	err = json.Unmarshal(jsonRequest, &credentials)
	if err != nil {
		w.WriteHeader(500)
		w.Write(errorResponse(err))
		return
	}
	existingIds := ch.checkExistingAppIds(credentials.AppNames)
	if len(existingIds) > 0 {
		w.WriteHeader(409)
		w.Write(errorResponse(errors.New(fmt.Sprintf("The following app ids exist already: %v", existingIds))))
		return
	}
	credentials.GenerateCypherKey()
	ch.db.Save(&credentials)
	w.WriteHeader(201)
	w.Write([]byte(credentials.String()))
}

func retrieveCredentials(ch *credentialsHandler, w http.ResponseWriter, r *http.Request) {

	log.Println(r.URL.Path)
}

// App Identifier Handler

type appIdentifierHandler struct {
	db gorm.DB
}

func AppIdentifierHandler(db gorm.DB) http.Handler {
	return &appIdentifierHandler{db}
}

func (aih *appIdentifierHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	var allAppIds []AppIdentifier
	aih.db.Order("credentials_id, app_name").Find(&allAppIds)

	var results = make([][]string, 0)
	var lastId int64 = -1
	for _, aid := range allAppIds {
		if aid.CredentialsId != lastId {
			results = append(results, []string{aid.AppName})
			lastId = aid.CredentialsId
		} else {
			group := results[len(results)-1]
			results[len(results)-1] = append(group, aid.AppName)
		}
	}
	bytes, err := json.Marshal(results)
	if err != nil {
		w.WriteHeader(500)
		w.Write(errorResponse(err))
		log.Printf("Error processing request: %v", err)
		return
	}
	w.WriteHeader(200)
	w.Write(bytes)
}

// Private helper methods

func errorResponse(e error) (response []byte) {

	response, _ = json.Marshal(errorMessage{e.Error()})
	return
}

func (ch *credentialsHandler) checkExistingAppIds(appIds []AppIdentifier) (existing []AppIdentifier) {

	idstrs := make([]string, len(appIds))
	for idx, aid := range appIds {
		idstrs[idx] = aid.AppName
	}
	ch.db.Where("app_name in (?)", idstrs).Find(&existing)
	return
}
