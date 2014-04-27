package keymaster_test

import (
	"bitbucket.org/barnardn/keymaster"
	"encoding/json"
	"fmt"
	"github.com/jinzhu/gorm"
	_ "github.com/mattn/go-sqlite3"
	"io/ioutil"
	"log"
	"os"
	"testing"
)

var (
	database gorm.DB
)

const storeFilename string = "./keymaster-test.sqlite3"

func init() {
	var err error
	os.Remove(storeFilename)
	database, err = gorm.Open("sqlite3", storeFilename)
	if err != nil {
		panic(fmt.Sprintf("Unable to open database %v", err))
	}
	database.CreateTable(keymaster.AppIdentifier{})
	database.CreateTable(keymaster.AppKey{})
	database.CreateTable(keymaster.Credentials{})
}

func TestImportJson(t *testing.T) {

	rawJson, err := ioutil.ReadFile("test-import.json")
	if err != nil {
		t.Errorf("Unable to read json file: ", err)
	}
	var credentials keymaster.Credentials
	err = json.Unmarshal(rawJson, &credentials)
	if err != nil {
		t.Errorf("Json upackaging error: ", err)
	}
	credentials.GenerateCypherKey()
	database.Save(&credentials)

	fmt.Println(credentials.String())
}

func TestFindAllAppnames(t *testing.T) {

	log.Println("TestFindAllAppnames")
	defer log.Println("/TestFindAllAppnames")

	var appKeys []keymaster.AppKey
	database.Find(&appKeys)
	if len(appKeys) != 3 {
		t.Errorf("Expected two keys, got %d", len(appKeys))
	}
}

func TestGetCredentials(t *testing.T) {

	log.Println("TestGetCredentials")
	defer log.Println("/TestGetCredentials")

	var cred keymaster.Credentials
	database.Find(&cred, 1)
	fmt.Println("%+v", cred)
	var names []keymaster.AppIdentifier
	database.Model(&cred).Related(&names)
}

func TestFindSpecificAppName(t *testing.T) {

	log.Println("TestFindSpecificAppName")
	defer log.Println("/TestFindSpecificAppName")

	var appIdentifier keymaster.AppIdentifier
	database.Find(&appIdentifier, keymaster.AppIdentifier{AppName: "com.clamdango.appname"})
	var credentials keymaster.Credentials
	database.Model(&appIdentifier).Related(&credentials)
}
