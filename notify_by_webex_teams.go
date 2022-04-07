// notify_by_webex_teams.go
//
// CLI command for sending messages to Cisco Webex rooms.
// Send messages and files to Webex. If *room name* is not found a new room is created.
// Supports on file upload per request.
// by Herwig Grimm (herwig.grimm at aon.at)
//
// required args:
//			-T <Webex bot token>
//			-t <team name>
//			-r <room name>
//			-m <markdown message> | -i
//
// optinal args:
//			-p <proxy server>
//			-f <png filename and path to send>
//			-d <message_id>
//			-a <card_attachment>
//			-i ... use standard input instead of flag -m
//
// example:
//			upload_poc.exe -T <apitoken> -t "Test-Team" -r "INM18/00021" -m "Happy hacking" -f upload_poc.go
//
// doc links:
//			https://developer.webex.com/getting-started.html
//
// changelog:
//				V0.1 (16.05.2018): 	initial release
//				V0.2 (20.05.2018): 	now files (HTTP link) can be send via flag -a
//				V0.3 (25.05.2018): 	complete redesign without 3rd party library (github.com/vallard/spark/)
//					and new file upload function added via flag -f
//				V0.4 (24.11.2019): 	new message delete function via flag -d
//					and card attachment via flag -a. see also https://developer.webex.com/docs/api/guides/cards and https://adaptivecards.io/designer/
//				V0.5 (07.04.2022): new flag -i for reading messages from standard input and new flag description for flag -T
//
// card attachment example:
//				./notify_by_webex_teams -T "<token>" -t "KMP-Test-Team" -r "Allgemein" -m "Test GRH 010" \
//				-a '{ "contentType": "application/vnd.microsoft.card.adaptive", "content": { "type": "AdaptiveCard", "version": "1.0", "body": [ { "type": "TextBlock", "text": "Please enter your comment here: " }, { "type": "Input.Text", "id": "name", "title": "New Input.Toggle", "placeholder": "comment text" } ], "actions": [ { "type": "Action.Submit", "title": "accept", "data": { "answer": "accept " } }, { "type": "Action.Submit", "title": "decline", "data": { "answer": "decline " } } ] } }'
package main

import (
	"bufio"
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
	"net/textproto"
	"net/url"
	"os"
	"path"
	"runtime"
	"strings"
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

type Message struct {
	ID          string    `json:"id"`
	RoomID      string    `json:"roomId"`
	RoomType    string    `json:"roomType"`
	Text        string    `json:"text"`
	Files       []string  `json:"files"`
	PersonID    string    `json:"personId"`
	PersonEmail string    `json:"personEmail"`
	Markdown    string    `json:"markdown"`
	HTML        string    `json:"html"`
	Created     time.Time `json:"created"`
}

var (
	uploadFile      string
	proxyString     string
	markdownMsg     string
	apiToken        string
	teamName        string
	roomName        string
	showVersion     bool
	deleteMessageId string
	cardAttachment  string
	useStdIn        bool
)

const (
	roomsURL    = "https://api.ciscospark.com/v1/rooms"
	messagesURL = "https://api.ciscospark.com/v1/messages"
	version     = "0.5"
)

func init() {
	flag.StringVar(&apiToken, "T", "", "Webex bot token (bot must be member of team and room)")
	flag.StringVar(&teamName, "t", "Developer-Team", "team name")
	flag.StringVar(&roomName, "r", "Room1", "room name")
	flag.StringVar(&uploadFile, "f", "", "PNG filename and path to send")
	flag.StringVar(&markdownMsg, "m", "", "markdown message")
	flag.StringVar(&proxyString, "p", "", "proxy server. format: http://<user>:<password>@<hostname>:<port>")
	flag.StringVar(&deleteMessageId, "d", "", "delete message. provide message id")
	flag.StringVar(&cardAttachment, "a", "", "card attachment -a see https://developer.webex.com/docs/api/guides/cards and https://adaptivecards.io/designer/")
	flag.BoolVar(&showVersion, "V", false, "show version")
	flag.BoolVar(&useStdIn, "i", false, "read message from standard input")
}

func createMessageAndAttachmentsToRoom(markdownMsg, roomID, attachment string) (string, error) {

	b := new(bytes.Buffer)
	b.WriteString(`{"roomId": "`)
	b.WriteString(roomID)
	b.WriteString(`", `)
	b.WriteString(`"markdown": "`)
	b.WriteString(markdownMsg)
	b.WriteString(`", `)
	b.WriteString(`"attachments": [`)
	b.WriteString(attachment)
	b.WriteString(`]`)
	b.WriteString(`}`)

	log.Printf("postData: %s\n", b.String())

	resp, err := webexTeamsRequest(apiToken, proxyString, "POST", messagesURL, nil, b)
	if err != nil {
		return "", err
	}
	log.Printf("createMessageAndAttachmentsToRoom() HTTP status code: %d", resp.StatusCode)
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	resp.Body.Close()

	var m Message
	err = json.Unmarshal(body, &m)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("createMessageAndAttachmentsToRoom() message ID: %s", m.ID)
	log.Printf("createMessageAndAttachmentsToRoom() message created: %s", m.Created)
	// log.Printf("createMessageToRoom body: %s\n", body)
	return "", err
}

func createMessageAndUploadToRoom(markdownMsg, roomID, uploadFile string) (string, error) {

	extraParams := map[string]string{
		"roomId":   roomID,
		"markdown": markdownMsg,
		"roomType": "group",
	}

	log.Printf("file to upload: %s\n", uploadFile)
	request, err := newfileUploadRequest(messagesURL, extraParams, "files", uploadFile)
	// log.Printf("newfileUploadRequest: %+v\n", request)
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

	log.Printf("request.ContentLength %d\n", request.ContentLength)
	// fmt.Printf("request.Header: %#v\n", request.Header)
	resp, err := client.Do(request)
	if err != nil {
		log.Printf("request error\n")
		log.Fatal(err)
	} else {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		resp.Body.Close()
		log.Printf("createMessageAndUploadToRoom() HTTP status code: %d", resp.StatusCode)

		var m Message
		err = json.Unmarshal(body, &m)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("createMessageAndUploadToRoom() message ID: %s", m.ID)
		log.Printf("createMessageAndUploadToRoom() message created: %s", m.Created)
	}
	return "", err
}

func createPngFormFile(w *multipart.Writer, fieldname, filename string) (io.Writer, error) {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, fieldname, filename))
	h.Set("Content-Type", "image/png;")
	return w.CreatePart(h)
}

// Creates a new file upload http request with optional extra params
func newfileUploadRequest(uri string, params map[string]string, fieldname, uploadFile string) (*http.Request, error) {
	buf := new(bytes.Buffer)
	w := multipart.NewWriter(buf)

	fw, err := createPngFormFile(w, fieldname, uploadFile)
	if err != nil {
		log.Println(err)
	}
	fd, err := os.Open(uploadFile)
	if err != nil {
		log.Println(err)
	}
	defer fd.Close()

	_, err = io.Copy(fw, fd)
	if err != nil {
		log.Println(err)
	}

	for key, val := range params {
		err = w.WriteField(key, val)
		if err != nil {
			log.Println(err)
		}
	}

	// Important if you do not close the multipart writer you will not have a
	// terminating boundry
	w.Close()

	req, err := http.NewRequest("POST", uri, buf)
	if err != nil {
		log.Println(err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

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
	log.Printf("webexTeamsRequest() uriAndValues: %s\n", uriAndValues)
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

	log.Printf("roomsURL: %s", roomsURL)
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
		// Files    []string `json:"files"`
	}

	// newMessage := &NewSparkMessage{RoomID: roomID, Markdown: messageText, Files: []string{"https://www.kapsch.net/KapschInternet/media/CarrierCom/PressCorner/Kapsch_Claim_White-Yellow_RGB.png"}}
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
	resp.Body.Close()
	log.Printf("createMessageAndUploadToRoom() HTTP status code: %d", resp.StatusCode)

	var m Message
	err = json.Unmarshal(body, &m)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("createMessageToRoom() message ID: %s", m.ID)
	log.Printf("createMessageToRoom() message created: %s", m.Created)
	// log.Printf("createMessageToRoom body: %s\n", body)
	return "", err
}

func deleteMessage(messageID string) error {
	url := fmt.Sprintf("%s/%s", messagesURL, messageID)
	resp, err := webexTeamsRequest(apiToken, proxyString, "DELETE", url, nil, nil)
	if err != nil {
		return err
	}
	log.Printf("deleteMessage() HTTP status code: %d", resp.StatusCode)
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	flag.Parse()

	lineSeparator := byte('\n')
	if runtime.GOOS == "darwin" {
		lineSeparator = byte('\r')
	}

	if useStdIn {
		reader := bufio.NewReader(os.Stdin)
		for {
			line, err := reader.ReadString(lineSeparator)
			line = strings.TrimSuffix(line, "\r")
			line = strings.TrimSuffix(line, "\r\n")
			markdownMsg += line + "\n"
			if err == io.EOF {
				break
			}
		}
	}

	if showVersion {
		fmt.Printf("%s version: %s\n", path.Base(os.Args[0]), version)
		os.Exit(0)
	}

	if len(deleteMessageId) > 0 {
		err := deleteMessage(deleteMessageId)
		if err != nil {
			log.Fatal(err)
		}
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

	if len(cardAttachment) > 0 {
		_, err := createMessageAndAttachmentsToRoom(markdownMsg, roomID, cardAttachment)
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}

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
