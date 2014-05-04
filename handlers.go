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

type credentialsHandler struct {
	db     gorm.DB
	logger *log.Logger
}

type errorMessage struct {
	Message string
}

func CredentialsHandler(db gorm.DB, logger *log.Logger) http.Handler {

	return &credentialsHandler{db, logger}
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
	case "put":
		replaceCredentials(credHandler, w, r)
	default:
		w.WriteHeader(404)
	}
}

func storeNewCredentials(ch *credentialsHandler, w http.ResponseWriter, r *http.Request) {

	jsonRequest, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(500)
		w.Write(errorResponse(err))
		ch.logger.Println(err)
		return
	}
	var credentials Credentials
	err = json.Unmarshal(jsonRequest, &credentials)
	if err != nil {
		w.WriteHeader(500)
		w.Write(errorResponse(err))
		ch.logger.Println(err)
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

	if len(pathParts) == 0 {
		w.WriteHeader(400)
		w.Write(errorResponse(errors.New("Missing application identifer")))
		return
	}

	var cred Credentials
	_, err := cred.FindByAppIdentifier(ch.db, pathParts[3])
	if err != nil {
		w.WriteHeader(404)
		w.Write(errorResponse(err))
		return
	}
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
	var cred Credentials
	_, err := cred.FindByAppIdentifier(ch.db, appId)
	if err != nil {
		w.WriteHeader(404)
		w.Write(errorResponse(err))
		return
	}
	cred.GenerateCypherKey()
	ch.db.Save(&cred)
	w.Write([]byte(cred.String()))
}

func replaceCredentials(ch *credentialsHandler, w http.ResponseWriter, r *http.Request) {

	pathParts := strings.Split(r.URL.String(), "/")
	if len(pathParts) < 3 {
		w.WriteHeader(400)
		return
	}

	if len(pathParts) == 0 {
		w.WriteHeader(400)
		w.Write(errorResponse(errors.New("Missing application identifer")))
		return
	}

	var oldCred, newCred Credentials
	_, err := oldCred.FindByAppIdentifier(ch.db, pathParts[3])
	if err != nil {
		w.WriteHeader(404)
		w.Write(errorResponse(err))
		return
	}
	jsonRequest, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(500)
		w.Write(errorResponse(err))
		ch.logger.Println(err)
		return
	}
	err = json.Unmarshal(jsonRequest, &newCred)
	if err != nil {
		w.WriteHeader(400)
		w.Write(errorResponse(err))
		return
	}
	newCred.CypherKey = oldCred.CypherKey
	ch.db.Where("credentials_id = ?", oldCred.Id).Delete(AppIdentifier{})
	ch.db.Where("credentials_id = ?", oldCred.Id).Delete(AppKey{})
	ch.db.Delete(&oldCred)
	ch.db.Save(&newCred)
	w.Write([]byte(newCred.String()))
}

// App Identifier Handler

type appIdentifierHandler struct {
	db     gorm.DB
	logger *log.Logger
}

func AppIdentifierHandler(db gorm.DB, logger *log.Logger) http.Handler {
	return &appIdentifierHandler{db, logger}
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
		aih.logger.Println(err)
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
