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

	w.Header().Set("Content-Type", "application/json")

	switch strings.ToLower(r.Method) {
	case "post":
		storeNewCredentials(credHandler, w, r)
	case "get":
		retrieveCredentials(credHandler, w, r)
	case "patch":
		reissueCypherKey(credHandler, w, r)
	default:
		w.WriteHeader(404)
	}
}

func storeNewCredentials(ch *credentialsHandler, w http.ResponseWriter, r *http.Request) {

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

	pathParts := strings.Split(r.URL.String(), "/")
	if len(pathParts) < 3 {
		w.WriteHeader(400)
		return
	}
	log.Printf("path parts: %v", pathParts[3])
	if len(pathParts) == 0 {
		w.WriteHeader(400)
		w.Write(errorResponse(errors.New("Missing application identifer")))
		return
	}
	var (
		cred           Credentials
		aid            AppIdentifier
		appIdentifiers []AppIdentifier
		keys           []AppKey
	)
	ch.db.Find(&aid, &AppIdentifier{AppName: pathParts[3]})
	if ch.db.NewRecord(aid) {
		w.WriteHeader(404)
		return
	}
	ch.db.Model(&aid).Related(&cred).Related(&keys).Related(&appIdentifiers)
	if ch.db.NewRecord(cred) {
		w.WriteHeader(500)
		w.Write(errorResponse(errors.New("The app identifier was found, but there were no matching credentials.")))
		return
	}
	cred.AppNames = appIdentifiers
	cred.Keys = keys
	w.Write([]byte(cred.String()))
}

func reissueCypherKey(ch *credentialsHandler, w http.ResponseWriter, r *http.Request) {

	pathParts := strings.Split(r.URL.String(), "/")[3:]
	cmd := pathParts[len(pathParts)-1]
	if strings.ToLower(cmd) != "reissue" {
		w.WriteHeader(400)
		return
	}
	appId := pathParts[0]
	log.Printf("app id: %v", appId)
	cred, err := ch.findCredentials(appId)
	if err != nil {
		w.WriteHeader(404)
		w.Write(errorResponse(err))
		return
	}
	cred.GenerateCypherKey()
	ch.db.Save(&cred)
	w.Write([]byte(cred.String()))
}

// App Identifier Handler

type appIdentifierHandler struct {
	db gorm.DB
}

func AppIdentifierHandler(db gorm.DB) http.Handler {
	return &appIdentifierHandler{db}
}

func (aih *appIdentifierHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
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

func (ch *credentialsHandler) findCredentials(appId string) (cred Credentials, err error) {
	var (
		aid            AppIdentifier
		appIdentifiers []AppIdentifier
		keys           []AppKey
	)
	ch.db.Find(&aid, &AppIdentifier{AppName: appId})
	if ch.db.NewRecord(aid) {
		return cred, errors.New("Not Found")
	}
	ch.db.Model(&aid).Related(&cred).Related(&keys).Related(&appIdentifiers)
	if ch.db.NewRecord(cred) {
		return cred, errors.New("Application id exits but credentials are missing or removed")
	}
	cred.AppNames = appIdentifiers
	cred.Keys = keys
	return cred, nil
}
