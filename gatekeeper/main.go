package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
)

const KeymasterURLEnv string = "KEYMASTER_SERVER_URL"

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

	flag.StringVar(&serverFlag, "keymaster", "", fmt.Sprintf("url of keymaster server. overrides %v environment variable if set", KeymasterURLEnv))
	flag.StringVar(&cmdFlag, "cmd", "list", fmt.Sprintf("command: on of %v", validCommands))
	flag.StringVar(&appIdFlag, "appid", "", "application identifer of the credentials you wish to operate on")
	flag.StringVar(&inputFlag, "spec", "", "application credentials spec file [json]")
	flag.StringVar(&outputFlag, "output", "", "output file in which to store credentials. omit for stdout")
	flag.BoolVar(&decryptFlag, "decrypt", false, "decrypt the contents of the retrieved credentials to stdout")
}

func main() {

	flag.Parse()
	keyMasterURL := os.Getenv(KeymasterURLEnv)
	if serverFlag != "" {
		keyMasterURL = serverFlag
	}
	if keyMasterURL == "" {
		panic(fmt.Sprintf("You must supply the url of the keymaster. pass in \"keymaster=URL\" or set \"%v\" in your environment", KeymasterURLEnv))
	}
	fmt.Printf("This is my command: %v\n", cmdFlag)
	switch cmdFlag {
	case "list":
		appIds, err := getAllAppIds(serverFlag)
		if err != nil {
			panic(err)
		}
		fmt.Println("app ids: %v", appIds)
	}
}

func getAllAppIds(keymaster string) (appids [][]string, err error) {

	kmUrl, err := url.Parse(keymaster)
	if err != nil {
		return
	}
	kmUrl.Path = "/api/apps"
	resp, err := http.Get(kmUrl.String())
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
