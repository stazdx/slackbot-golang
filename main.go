package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

func main() {
	godotenv.Load(".env")

	token := os.Getenv("SLACK_AUTH_TOKEN")
	appToken := os.Getenv("SLACK_APP_TOKEN")

	client := slack.New(token, slack.OptionDebug(true), slack.OptionAppLevelToken(appToken))

	socketClient := socketmode.New(
		client,
		socketmode.OptionDebug(true),

		socketmode.OptionLog(log.New(os.Stdout, "socketmode: ", log.Lshortfile|log.LstdFlags)),
	)

	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	go func(ctx context.Context, client *slack.Client, socketClient *socketmode.Client) {
		for {
			select {
			case <-ctx.Done():
				log.Println("Shutting Down!!")
				return
			case event := <-socketClient.Events:
				switch event.Type {
				case socketmode.EventTypeEventsAPI:
					eventsAPIEvent, ok := event.Data.(slackevents.EventsAPIEvent)
					if !ok {
						log.Printf("Could not type cast the event to the EventsAPIEvent: %v\n", event)
						continue
					}
					socketClient.Ack(*event.Request)

					err := handleEventMessage(eventsAPIEvent, client)
					if err != nil {
						// Replace with actual err handeling
						log.Fatal(err)
					}

				case socketmode.EventTypeSlashCommand:
					// Just like before, type cast to the correct event type, this time a SlashEvent
					command, ok := event.Data.(slack.SlashCommand)
					if !ok {
						log.Printf("Could not type cast the message to a SlashCommand: %v\n", command)
						continue
					}

					// handleSlashCommand will take care of the command
					payload, err := handleSlashCommand(command, client)
					if err != nil {
						log.Fatal(err)
					}
					// Dont forget to acknowledge the request and send the payload
					socketClient.Ack(*event.Request, payload)
				}
			}
		}
	}(ctx, client, socketClient)

	socketClient.Run()
}

func handleEventMessage(event slackevents.EventsAPIEvent, client *slack.Client) error {
	switch event.Type {
	case slackevents.CallbackEvent:
		innerEvent := event.InnerEvent
		switch ev := innerEvent.Data.(type) {
		case *slackevents.AppMentionEvent:
			// log.Println(ev)
			err := handleAppMentionEvent(ev, client)
			if err != nil {
				return err
			}
		}
	default:
		return errors.New("unsupported event type")
	}
	return nil
}

func handleSlashCommand(command slack.SlashCommand, client *slack.Client) (interface{}, error) {

	switch command.Command {
	case "/acuso":
		return nil, handleAccuseCommand(command, client)
		// case "/was-this-article-useful":
		// 	return handleIsArticleGood(command, client)
	}

	return nil, nil
}

func GetAccusedUser(command slack.SlashCommand, client *slack.Client) (*slack.User, error) {

	// fmt.Println(userProfile)

	// result, err := client.GetUsers()
	// if err != nil {
	// 	println(err)
	// }

	// for _, v := range result {
	// 	if command.Text != nil {
	// 		if strings.Contains(command.Text, v.ID) {
	// 			if v.Profile.DisplayName != nil {

	// 			}
	// 		}
	// 	}
	// }
	// command.Text = "A quién acusas? Etiquétalo!"

	return nil, nil
}

func handleAccuseCommand(command slack.SlashCommand, client *slack.Client) error {
	// The Input is found in the text field so
	// Create the attachment and assigned based on the message

	// accused = GetAccusedUser(command slack.SlashCommand, client *slack.Client)
	accusedUserID := strings.TrimLeft(strings.TrimRight(command.Text, "|"), "<@")

	fmt.Println("======== ID =========", accusedUserID)

	params := new(slack.GetUserProfileParameters)

	userProfile, err := client.GetUserProfile(params{accusedUserID})

	fmt.Println("======== ID =========", userProfile)

	fmt.Println("\n \n +++++++++++++++++++ \n \n", command, "\n \n +++++++++++++++++++ \n \n")
	attachment := slack.Attachment{}
	// Add Some default context like user who mentioned the bot
	attachment.Fields = []slack.AttachmentField{
		{
			Title: "Usuario que acusa",
			Value: command.UserName,
		}, {
			Title: "Usuario acusado",
			Value: "test",
		}, {
			Title: "Fecha del crimen",
			Value: time.Now().String(),
		},
	}

	attachment.Text = fmt.Sprintf("Gracias por acusar %s : \n %s", "@"+command.UserName, command.Text)
	attachment.Color = "#4af030"

	_, _, err = client.PostMessage(command.ChannelID, slack.MsgOptionAttachments(attachment))
	if err != nil {
		return fmt.Errorf("failed to post message: %w", err)
	}
	return nil
}

func handleAppMentionEvent(event *slackevents.AppMentionEvent, client *slack.Client) error {
	user, err := client.GetUserInfo(event.User)
	if err != nil {
		return err
	}

	// fmt.Println("---------- \n", user, "---------- \n")

	text := strings.ToLower(event.Text)

	attachment := slack.Attachment{}
	attachment.Fields = []slack.AttachmentField{
		{
			Title: "Date",
			Value: time.Now().String(),
		}, {
			Title: "User",
			Value: user.Name,
		},
	}
	if strings.Contains(text, "hola") {
		attachment.Text = fmt.Sprintf("Hola %s", user.Name)
		attachment.Pretext = "Hola Peiger!"
		attachment.Color = "#5DADE2"
	} else {
		attachment.Text = fmt.Sprintf("En qué te puedo ayudar %s?", user.Name)
		attachment.Pretext = "Qué tal todo! " + user.Name
		attachment.Color = "#7D3C98"
	}

	_, _, err = client.PostMessage(event.Channel, slack.MsgOptionAttachments(attachment))
	if err != nil {
		return fmt.Errorf("failed to post message: %w", err)
	}
	return nil
}
