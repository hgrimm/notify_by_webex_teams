// notify_by_webex_teams.go
//
// CLI command for sending messages to Cisco Webex Teams rooms.
// Send messages and files to Webex Teams. If *room name* is not found a new room is created.
// Supports on file upload per request.
// by Herwig Grimm (herwig.grimm at aon.at)
//
// required args:
//			-T <Webex Teams API token>
//			-t <team name>
//			-r <room name>
//			-m <markdown message>
//
// optinal args:
//			-p <proxy server>
//			-f <filename and path to send>
//
// example:
//			upload_poc.exe -T <apitoken> -t "KMP-Test-Team" -r "INM18/00021" -m "Happy hacking" -f upload_poc.go
//
// doc links:
//			https://developer.webex.com/getting-started.html
//
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"time"
)

type roomsResp struct {
	Items []struct {
		ID           string    `json:"id"`
		Title        string    `json:"title"`
		Type         string    `json:"type"`
		IsLocked     bool      `json:"isLocked"`
		LastActivity time.Time `json:"lastActivity"`
		TeamID       string    `json:"teamId,omitempty"`
		CreatorID    string    `json:"creatorId"`
		Created      time.Time `json:"created"`
	} `json:"items"`
}

type SparkRoom struct {
	Id           string
	Title        string
	Type         string
	IsLocked     bool
	LastActivity time.Time
	CreatorId    string
	Created      time.Time
	TeamId       string
}

var (
	uploadFile  string
	proxyString string
	markdownMsg string
	apiToken    string
	teamName    string
	roomName    string
	showVersion bool
)

const (
	roomsURL    = "https://api.ciscospark.com/v1/rooms"
	messagesURL = "https://api.ciscospark.com/v1/messages"
	version     = "0.2"
)

func init() {
	flag.StringVar(&apiToken, "T", "", "Webex Teams API token")
	flag.StringVar(&teamName, "t", "KMP-Developer-Team", "team name")
	flag.StringVar(&roomName, "r", "Room1", "room name")
	flag.StringVar(&uploadFile, "f", "", "filename and path to send")
	flag.StringVar(&markdownMsg, "m", "", "markdown message")
	flag.StringVar(&proxyString, "p", "", "proxy server. format: http://<user>:<password>@<hostname>:<port>")
}

func createMessageAndUploadToRoom(markdownMsg, roomID, uploadFile string) (string, error) {

	extraParams := map[string]string{
		"roomId":   roomID,
		"markdown": markdownMsg,
		"roomType": "group",
	}

	log.Printf("file to upload: %s\n", uploadFile)
	request, err := newfileUploadRequest(messagesURL, extraParams, "files", uploadFile)
	if err != nil {
		return "", err
	}

	client := &http.Client{}
	if len(proxyString) > 0 {
		proxyURL, err := url.Parse(proxyString)
		if err != nil {
			log.Fatal(err)
		}
		tr := &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
			// TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client.Transport = tr
	}

	authBearer := fmt.Sprintf("Bearer %s", apiToken)
	request.Header.Add("Authorization", authBearer)
	// fmt.Printf("request.Header: %#v\n", request.Header)
	resp, err := client.Do(request)
	if err != nil {
		log.Fatal(err)
	} else {
		body := &bytes.Buffer{}
		_, err := body.ReadFrom(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		resp.Body.Close()
		log.Printf("createMessageAndUploadToRoom() HTTP status code: %d", resp.StatusCode)
	}
	return "", err
}

// Creates a new file upload http request with optional extra params
func newfileUploadRequest(uri string, params map[string]string, paramName, path string) (*http.Request, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(paramName, filepath.Base(path))
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, file)

	for key, val := range params {
		_ = writer.WriteField(key, val)
	}
	err = writer.Close()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", uri, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req, err
}

func webexTeamsRequest(apiToken string,
	proxyString string,
	method string,
	baseURL string,
	values url.Values,
	buf io.Reader) (*http.Response, error) {

	var resp *http.Response

	authBearer := fmt.Sprintf("Bearer %s", apiToken)

	uriAndValues := fmt.Sprintf("%s?%s", baseURL, values.Encode())
	req, err := http.NewRequest(method, uriAndValues, buf)

	client := &http.Client{}
	if len(proxyString) > 0 {
		proxyURL, err := url.Parse(proxyString)
		if err != nil {
			return resp, err
		}
		tr := &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
			// TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client.Transport = tr
	}

	req.Header.Add("Content-Type", "application/json; charset=utf-8")
	req.Header.Add("Authorization", authBearer)

	resp, err = client.Do(req)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

func getTeamIDByName(name string) (string, error) {
	queryValues := url.Values{}
	queryValues.Add("type", "group")

	resp, err := webexTeamsRequest(apiToken, proxyString, "GET", roomsURL, queryValues, nil)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("getTeamIDByName() HTTP status code: %d", resp.StatusCode)
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var rr roomsResp
	err = json.Unmarshal(body, &rr)
	if err != nil {
		log.Fatal(err)
	}

	for _, v := range rr.Items {
		if v.Title == name {
			return v.TeamID, nil
		}
	}
	errMessage := fmt.Sprintf("No room with name: %s was found\n", name)
	return "", errors.New(errMessage)
}

func createRoomAndGetRoom(teamID string, name string) (string, error) {
	queryValues := url.Values{}
	queryValues.Add("teamId", teamID)
	queryValues.Add("type", "group")

	resp, err := webexTeamsRequest(apiToken, proxyString, "GET", roomsURL, queryValues, nil)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("createRoomAndGetRoom() HTTP status code: %d", resp.StatusCode)
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var rr roomsResp
	err = json.Unmarshal(body, &rr)
	if err != nil {
		log.Fatal(err)
	}

	for _, v := range rr.Items {
		if v.Title == name {
			return v.ID, nil
		}
	}

	log.Printf("room name >>%s<< not found\n", name)

	roomID, err := createRoom(name, teamID)
	if err != nil {
		log.Fatal(err)
	}
	return roomID, nil
}

func createRoom(roomTitle, teamID string) (string, error) {

	var nr SparkRoom

	type NewSparkRoom struct {
		TeamID string `json:"teamId"`
		Title  string `json:"title"`
	}

	newRoom := &NewSparkRoom{TeamID: teamID, Title: roomTitle}

	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(newRoom)
	fmt.Printf("createRoom json: %s\n", b.String())
	// bytes, err := s.PostRequest(RoomsUrl, b, "")
	resp, err := webexTeamsRequest(apiToken, proxyString, "POST", roomsURL, nil, b)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("createRoom() HTTP status code: %d", resp.StatusCode)
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	if err != nil {
		return nr.Id, err
	}
	err = json.Unmarshal(body, &nr)
	return nr.Id, err
}

func createMessageToRoom(messageText, roomID string) (string, error) {

	type NewSparkMessage struct {
		RoomID   string `json:"roomId"`
		Markdown string `json:"markdown"`
	}

	newMessage := &NewSparkMessage{RoomID: roomID, Markdown: messageText}

	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(newMessage)

	resp, err := webexTeamsRequest(apiToken, proxyString, "POST", messagesURL, nil, b)
	if err != nil {
		return "", err
	}
	log.Printf("createMessageToRoom() HTTP status code: %d", resp.StatusCode)
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	log.Printf("createMessageToRoom body: %s\n", body)
	return "", err
}

func main() {
	flag.Parse()

	if showVersion {
		fmt.Printf("%s version: %s\n", path.Base(os.Args[0]), version)
		os.Exit(0)
	}

	teamID, err := getTeamIDByName(teamName)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("teamID: %s\n", teamID)

	roomID, err := createRoomAndGetRoom(teamID, roomName)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("roomID: %s\n", roomID)

	if len(uploadFile) > 0 {
		_, err = createMessageAndUploadToRoom(markdownMsg, roomID, uploadFile)
		if err != nil {
			log.Fatal(err)
		}

	} else {
		_, err = createMessageToRoom(markdownMsg, roomID)
		if err != nil {
			log.Fatal(err)
		}
	}

}
