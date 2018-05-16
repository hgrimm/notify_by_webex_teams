package main

import (
	"flag"
	"log"

	"github.com/vallard/spark"
)

var (
	apiToken string
	teamName string
	roomName string
	msgText  string
)

func init() {
	flag.StringVar(&apiToken, "T", "", "Webex Teams API token")
	flag.StringVar(&teamName, "t", "KMP-Developer-Team", "team name")
	flag.StringVar(&roomName, "r", "Incident x", "new room name")
	flag.StringVar(&msgText, "m", "test message", "messages text")
}

const (
	roomsURL = "https://api.ciscospark.com/v1/rooms"
)

func main() {


	flag.Parse()

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
