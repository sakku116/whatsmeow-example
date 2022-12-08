package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/lib/pq"
	"github.com/mdp/qrterminal"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
	_ "modernc.org/sqlite"
)

var client *whatsmeow.Client

func eventHandler(evt interface{}) {
	// handle incoming event. such message to current logged user

	switch v := evt.(type) {
	case *events.Message:
		// if !v.Info.IsFromMe {
		if v.Message.GetConversation() != "" {
			fmt.Printf("MESSAGE RECIEVED from %s!: %s\n", v.Info.Sender.User, v.Message.GetConversation())
		}
		// }
	}
}

func sendMessage(to string, body string) {
	client.SendMessage(
		context.Background(),
		types.JID{
			User:   to,
			Server: types.DefaultUserServer,
		},
		"",
		&waProto.Message{
			Conversation: proto.String(body),
		},
	)
}

func main() {
	// Make sure to import sqlite driver
	// database, err := sqlstore.New("postgres", "host=localhost user=postgres password=root dbname=whatsmeow port=5432 sslmode=disable", nil)
	database, err := sqlstore.New("sqlite", "file:whatsmeow.db?_foreign_keys=on", nil)
	if err != nil {
		panic(err)
	}

	// get session device
	deviceStore, err := database.GetFirstDevice()
	if err != nil {
		panic(err)
	}

	logger := waLog.Stdout("Whatmeow Client:", "INFO", true)
	client = whatsmeow.NewClient(deviceStore, logger)

	// incoming event handler
	client.AddEventHandler(eventHandler)

	// authentication
	if client.Store.ID == nil {
		// No ID stored, new login
		qr_chan, _ := client.GetQRChannel(context.Background())
		err = client.Connect()
		if err != nil {
			panic(err)
		}
		for evt := range qr_chan {
			if evt.Event == "code" {
				// render qr in terminal
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
	} else {
		// Already logged in, just connect
		err = client.Connect()
		if err != nil {
			panic(err)
		}
	}

	// Listen to Ctrl+C (you can also do something else that prevents the program from exiting)
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	client.Disconnect()
}
