package main

import (
	"bitbucket.org/barnardn/keymaster"
	"bytes"
	"crypto/aes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/nu7hatch/gouuid"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
)

const (
	KeymasterURLEnv                 string = "KEYMASTER_SERVER_URL"
	KeymasterCredentialsAPIEndpoint string = "/api/credentials"
	KeymasterAppListingAPIEndpoint  string = "/api/apps"
)

var (
	serverFlag    string
	cmdFlag       string
	appIdFlag     string
	inputFlag     string
	outputFlag    string
	decryptFlag   bool
	validCommands []string = []string{"add", "update", "list", "get", "reissue"}
)

func init() {

	flag.StringVar(&serverFlag, "keymaster", "", fmt.Sprintf("Full of keymaster server. overrides %v environment variable if set", KeymasterURLEnv))
	flag.StringVar(&cmdFlag, "cmd", "list", fmt.Sprintf("Where command is one of %v", validCommands))
	flag.StringVar(&appIdFlag, "appid", "", "The application identifer of the credentials you wish to operate on")
	flag.StringVar(&inputFlag, "spec", "", "The application credentials spec file [json]")
	flag.StringVar(&outputFlag, "output", "", "Output file in which to store credentials. omit for stdout")
	flag.BoolVar(&decryptFlag, "decrypt", false, "Decrypt the contents of the retrieved credentials to stdout")
}

func main() {

	flag.Parse()
	kmURL, err := keymasterURL()
	if err != nil {
		panic(fmt.Sprintf("You must supply the url of the keymaster. pass in \"keymaster=URL\" or set \"%v\" in your environment", KeymasterURLEnv))
	}
	switch cmdFlag {
	case "list":
		appIds, err := getAllAppIds(kmURL)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		displayAppIdList(appIds)
	case "get":
		cred, err := getCredentials(kmURL, appIdFlag)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		handleOutput(cred, outputFlag)
	case "reissue":
		cred, err := reissueCypherKey(kmURL, appIdFlag)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		handleOutput(cred, outputFlag)
	case "add":
		cred, err := editCredentials(kmURL, inputFlag, "POST", "")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		handleOutput(cred, outputFlag)
	case "update":
		cred, err := editCredentials(kmURL, inputFlag, "PUT", appIdFlag)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		handleOutput(cred, outputFlag)
	default:
		fmt.Println("Invalid command %s", cmdFlag)
		os.Exit(1)
	}
}

func getAllAppIds(keymaster *url.URL) (appids [][]string, err error) {

	keymaster.Path = KeymasterAppListingAPIEndpoint
	resp, err := http.Get(keymaster.String())
	if err != nil {
		return
	}
	defer resp.Body.Close()
	jsonBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	err = json.Unmarshal(jsonBytes, &appids)
	return
}

func getCredentials(keymaster *url.URL, appid string) (credentials string, err error) {

	if appid == "" {
		err = errors.New("You must supply an app id")
		return
	}
	keymaster.Path = fmt.Sprintf("%s/%s", KeymasterCredentialsAPIEndpoint, appid)
	resp, err := http.Get(keymaster.String())
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()
	jsonBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode != 200 {
		if resp.StatusCode == 400 {
			err = errors.New("Bad Request")
		} else {
			err = jsonError(jsonBytes, resp.StatusCode)
		}
		return
	}
	credentials = string(jsonBytes[:])
	return
}

func reissueCypherKey(keymaster *url.URL, appId string) (credentials string, err error) {

	keymaster.Path = fmt.Sprintf("%s/%s/%s", KeymasterCredentialsAPIEndpoint, appId, "reissue")
	request, _ := http.NewRequest("PATCH", keymaster.String(), nil)
	request.Header.Add("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	jsonBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode != 200 {
		if resp.StatusCode == 400 {
			err = errors.New("Bad Request")
		} else {
			err = jsonError(jsonBytes, resp.StatusCode)
		}
		return
	}
	credentials = string(jsonBytes[:])
	return
}

func editCredentials(keymaster *url.URL, credFilename string, cmd string, appid string) (credentials string, err error) {

	if credFilename == "" {
		err = errors.New("Missing input credentials json file")
		return
	}
	inbytes, err := ioutil.ReadFile(credFilename)
	if err != nil {
		return
	}
	// valid json check.
	var f interface{}
	err = json.Unmarshal(inbytes, &f)
	if err != nil {
		return
	}
	keymaster.Path = KeymasterCredentialsAPIEndpoint
	if cmd == "PUT" {
		if appid == "" {
			err = errors.New("Application id requried for update: -appid application-id")
			return
		}
		keymaster.Path = fmt.Sprintf("%s/%s", KeymasterCredentialsAPIEndpoint, appid)
	}
	request, _ := http.NewRequest(cmd, keymaster.String(), bytes.NewBuffer(inbytes))
	request.Header.Add("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	jsonBytes, err := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != 201 && resp.StatusCode != 200 {
		err = jsonError(jsonBytes, resp.StatusCode)
		return
	}
	credentials = string(jsonBytes[:])
	return
}

// * helper methods *

func keymasterURL() (serverURL *url.URL, err error) {

	l := os.Getenv(KeymasterURLEnv)
	if serverFlag != "" {
		l = serverFlag
	}
	serverURL, err = url.Parse(l)
	return
}

func displayAppIdList(appids [][]string) {

	if len(appids) == 0 {
		addr, _ := keymasterURL()
		fmt.Printf("There were no credentials sets found at the specified address: %s", addr)
		return
	}

	for setIndex, idSet := range appids {

		fmt.Printf("--= App Identifers In Credentials Set %d =--\n", setIndex)
		for idIndex, appid := range idSet {
			fmt.Printf("\t%d  %s\n", idIndex, appid)
		}
		fmt.Printf("\n")

	}
}

func jsonError(jsonBody []byte, statusCode int) (retError error) {

	var f interface{}
	err := json.Unmarshal(jsonBody, &f)
	if err != nil {
		panic(err)
	}
	jsonDict := f.(map[string]interface{})
	fmt.Printf("%v", jsonDict)
	if jsonDict["Message"] == nil {
		retError = fmt.Errorf("Server returned HTTP status code: %d", statusCode)
		return
	}
	msg := jsonDict["Message"].(string)
	retError = fmt.Errorf("%d %s", statusCode, msg)
	return
}

func handleOutput(s string, outFilename string) {

	if outFilename == "" {
		fmt.Printf("%v\n", s)
	} else {
		err := ioutil.WriteFile(outFilename, []byte(s), 0666)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
	if decryptFlag {
		decryptCredentials(s)
	}
}

func decryptCredentials(cred string) {

	var f interface{}
	_ = json.Unmarshal([]byte(cred), &f)

	dict := f.(map[string]interface{})
	cryptText64 := dict["cypherText"].(string)
	cryptKey := dict["cypherKey"].(string)

	aesKey, _ := uuid.ParseHex(cryptKey)
	cryptText, _ := base64.StdEncoding.DecodeString(cryptText64)
	plainBuf := make([]byte, len(cryptText))
	_ = keymaster.DecryptAESCBC(plainBuf, []byte(cryptText[:]), aesKey[:], []byte(aesKey[:aes.BlockSize]))
	fmt.Printf("%s\n", string(plainBuf[:]))
}
