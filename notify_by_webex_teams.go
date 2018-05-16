// 	file: notify_by_webex_teams.go
// 	Version 0.1 (16.05.2018)
//
// CLI command for sending messages to Cisco Webex Teams rooms
// by Herwig Grimm (herwig.grimm at aon.at)
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"

	"github.com/vallard/spark"
)

var (
	apiToken    string
	teamName    string
	roomName    string
	msgText     string
	showVersion bool
)

func init() {
	flag.StringVar(&apiToken, "T", "", "Webex Teams API token")
	flag.StringVar(&teamName, "t", "KMP-Developer-Team", "team name")
	flag.StringVar(&roomName, "r", "Room1", "new room name")
	flag.StringVar(&msgText, "m", "test message (markdown format)", "messages text")
	flag.BoolVar(&showVersion, "V", false, "print program version")
}

const (
	roomsURL = "https://api.ciscospark.com/v1/rooms"
	version  = "0.1"
)

func main() {

	flag.Parse()

	if showVersion {
		fmt.Printf("%s version: %s\n", path.Base(os.Args[0]), version)
		os.Exit(0)
	}

	if len(apiToken) == 0 {
		log.Fatalf("missing Webex Teams API token (flag -T)\n")
	}

	s := spark.New(apiToken)

	// Get the room ID of the room name
	room, err := s.GetRoomWithName(teamName)

	if err != nil {
		log.Fatalf("cannot find team >>%s<<: %s", teamName, err)
	}

	// check if roomName exists
	newRoom, err := s.GetRoomWithName(roomName)

	if err != nil {
		newRoom, err = s.CreateRoom(roomName, room.TeamId)
		if err != nil {
			log.Fatalf("cannot create room >>%s<<: %s", roomName, err)
		}
	}

	// Create the message we want to send
	m := spark.Message{
		RoomId:   newRoom.Id,
		Markdown: msgText,
	}

	// Post the message to the room
	_, err = s.CreateMessage(m)

	if err != nil {
		log.Fatalf("cannot create message >>%s<<: %s", roomName, err)
	}

}
