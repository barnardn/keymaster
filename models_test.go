package keymaster_test

import (
	"bitbucket.org/barnardn/keymaster"
	"crypto/aes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/jinzhu/gorm"
	_ "github.com/mattn/go-sqlite3"
	"github.com/nu7hatch/gouuid"
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

func TestFindNothing(t *testing.T) {
	log.Println("TestFindSpecificAppName")
	defer log.Println("/TestFindSpecificAppName")

	var appIdentifier keymaster.AppIdentifier
	database.Find(&appIdentifier, keymaster.AppIdentifier{AppName: "com.gogle.stuff"})
	if database.NewRecord(appIdentifier) {
		fmt.Printf("This is the empty thing I found: %v", appIdentifier)
	} else {
		t.Errorf("this thing was found %v", appIdentifier)
	}

}

func TestCryptoStuff(t *testing.T) {

	log.Println("TestCryptoStuff")
	defer log.Println("/TestCryptoStuff")

	var cred keymaster.Credentials
	found, err := cred.FindByAppIdentifier(database, "com.clamdango.primary")
	if err != nil {
		t.Errorf("Error  ", err)
	}
	if !found {
		t.Errorf("Hey, no credentials found")
	}
	// fmt.Printf("Encrypted credentials: %+v", cred.String())

	var f interface{}
	err = json.Unmarshal([]byte(cred.String()[:]), &f)
	dict := f.(map[string]interface{})
	cryptText64 := dict["cypherText"].(string)
	cryptKey := dict["cypherKey"].(string)

	aesKey, _ := uuid.ParseHex(cryptKey)
	cryptText, err := base64.StdEncoding.DecodeString(cryptText64)
	plainBuf := make([]byte, len(cryptText))
	err = keymaster.DecryptAESCFB(plainBuf, []byte(cryptText[:]), aesKey[:], []byte(aesKey[:aes.BlockSize]))
	fmt.Printf("decrypted: %v", string(plainBuf[:]))
}
