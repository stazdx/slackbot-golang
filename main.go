package main

import (
	"badger"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	// badger "github.com/dgraph-io/badger/v3"
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

	db, err := badger.Open(badger.DefaultOptions("/tmp/badger"))
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

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
						log.Fatal(err)
					}

				case socketmode.EventTypeSlashCommand:
					command, ok := event.Data.(slack.SlashCommand)
					if !ok {
						log.Printf("Could not type cast the message to a SlashCommand: %v\n", command)
						continue
					}

					payload, err := handleSlashCommand(command, client)
					if err != nil {
						log.Fatal(err)
					}

					socketClient.Ack(*event.Request, payload)

				case socketmode.EventTypeInteractive:
					interaction, ok := event.Data.(slack.InteractionCallback)
					if !ok {
						log.Printf("Could not type cast the message to a Interaction callback: %v\n", interaction)
						continue
					}

					err := handleInteractionEvent(interaction, client, db)
					if err != nil {
						log.Fatal(err)
					}
					socketClient.Ack(*event.Request)
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
		// return handleIsArticleGood(command, client)
		// case "/was-this-article-useful":
		// 	return handleIsArticleGood(command, client)
	}

	return nil, nil
}

func handleIsArticleGood(command slack.SlashCommand, client *slack.Client) (interface{}, error) {
	attachment := slack.Attachment{}

	checkbox := slack.NewCheckboxGroupsBlockElement("answer",
		slack.NewOptionBlockObject("yes", &slack.TextBlockObject{Text: "Yes", Type: slack.MarkdownType}, &slack.TextBlockObject{Text: "Did you Enjoy it?", Type: slack.MarkdownType}),
		slack.NewOptionBlockObject("no", &slack.TextBlockObject{Text: "No", Type: slack.MarkdownType}, &slack.TextBlockObject{Text: "Did you Dislike it?", Type: slack.MarkdownType}),
	)

	accessory := slack.NewAccessory(checkbox)

	attachment.Blocks = slack.Blocks{
		BlockSet: []slack.Block{
			slack.NewSectionBlock(
				&slack.TextBlockObject{
					Type: slack.MarkdownType,
					Text: "Did you think this article was helpful?",
				},
				nil,
				accessory,
			),
		},
	}

	attachment.Text = "Rate the tutorial"
	attachment.Color = "#4af030"
	return attachment, nil
}

func handleInteractionEvent(interaction slack.InteractionCallback, client *slack.Client, db *badger.DB) error {
	switch interaction.Type {
	case slack.InteractionTypeBlockActions:
		for _, action := range interaction.ActionCallback.BlockActions {
			log.Printf("%+v", action)
			log.Println("Selected option: ", action.SelectedOptions)

		}
	case slack.InteractionTypeInteractionMessage:
		for _, action := range interaction.ActionCallback.AttachmentActions {
			log.Printf("%+v", action)
			log.Println("Action: ", action.Name)
			switch action.Name {
			case "actionPenalize":
				err2 := db.Update(func(txn *badger.Txn) error {
					e := badger.NewEntry([]byte("answer3"), []byte("25")).WithMeta(byte(1))
					err2 := txn.SetEntry(e)
					return err2
				})
				return err2
				log.Println("Penalizar!")
			case "actionSave":
				err := db.View(func(txn *badger.Txn) error {
					item, _ := txn.Get([]byte("answer"))

					var valNot, valCopy []byte
					_ = item.Value(func(val []byte) error {
						fmt.Println("The answer is: %s\n", val)
						valCopy = append([]byte{}, val...)
						valNot = val // Do not do this.
						return nil
					})

					fmt.Printf("NEVER do this. %s\n", valNot)

					fmt.Printf("The answer is: %s\n", valCopy)

					valCopy, _ = item.ValueCopy(nil)
					fmt.Printf("The answer is: %s\n", valCopy)

					return nil
				})

				return err
				log.Println("Inocente!")
			}
		}
	default:
	}
	return nil
}

func handleAccuseCommand(command slack.SlashCommand, client *slack.Client) error {

	accusedUserID, found := GetUserIDByStrings(command.Text, "<@", "|")
	if found == false {
		return fmt.Errorf("Quieres acusar a alguien? Etiquétalo!")
	}

	userInfo, err := client.GetUserInfo(accusedUserID)

	attachment := slack.Attachment{}

	attachment.Title = ":rotating_light::rotating_light::rotating_light: ALERTA DE ACUSADO :rotating_light::rotating_light::rotating_light:"
	attachment.ImageURL = userInfo.Profile.Image512
	attachment.ThumbURL = userInfo.Profile.Image512

	attachment.Fallback = "Tú no tienes poderes aquí!"
	attachment.CallbackID = "Penalize"

	attachment.Fields = []slack.AttachmentField{
		{
			Title: ":squirrel: Usuario que acusa",
			Value: fmt.Sprintf("<@%s|%s>", command.UserID, command.UserName),
		}, {
			Title: ":skull_and_crossbones: Usuario acusado",
			Value: fmt.Sprintf("<@%s|%s>", userInfo.ID, userInfo.Name),
		}, {
			Title: ":memo: Declaración",
			Value: command.Text,
		}, {
			Title: ":date: Fecha del delito",
			Value: time.Now().Format("02-01-2020 15:04:05"),
		},
	}

	attachment.Actions = []slack.AttachmentAction{
		{
			Name:  "actionPenalize",
			Text:  "Culpable",
			Value: "Penalize",
			Style: "danger",
			Type:  "button",
			// Confirm: *slack.ConfirmationField{
			// 	Title:       "Ejecutar la acción",
			// 	Text:        "Después de esto no hay vuelta atrás :confused: !",
			// 	OkText:      "Sí",
			// 	DismissText: "No",
			// },
		},
		{
			Name:  "actionSave",
			Text:  "Inocente",
			Value: "Save",
			Style: "default",
			Type:  "button",
		},
	}

	attachment.Text = fmt.Sprintf("A continuación los detalles de la acusación:")
	attachment.Color = "#B22222"

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

func GetUserIDByStrings(str string, startS string, endS string) (result string, found bool) {
	s := strings.Index(str, startS)
	if s == -1 {
		return result, false
	}
	newS := str[s+len(startS):]
	e := strings.Index(newS, endS)
	if e == -1 {
		return result, false
	}
	result = newS[:e]

	return result, true
}
