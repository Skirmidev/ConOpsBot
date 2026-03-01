package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"strconv"

	"github.com/spf13/viper"
)

func main() {
	//viper config
	viper.SetConfigName("config.json")
	viper.SetConfigType("json")
	viper.AddConfigPath("./config")

	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}
	var C config
	err = viper.Unmarshal(&C)
	if err != nil {
		fmt.Printf("unable to decode into struct, %v", err)
	}

	//telegram bot
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	opts := []bot.Option{
		bot.WithDefaultHandler(handler),
	}

	b, err := bot.New(viper.GetString("bottoken"), opts...)
	if nil != err {
		// panics for the sake of simplicity.
		// you should handle this error properly in your code.
		panic(err)
	}

	//command handlers
	b.RegisterHandler(bot.HandlerTypeMessageText, "/block", bot.MatchTypeExact, blockHandler)

	b.Start(ctx)
}

func handler(ctx context.Context, b *bot.Bot, update *models.Update) {
	var C config
	err := viper.Unmarshal(&C)
	if err != nil {
		fmt.Printf("unable to decode into struct, %v", err)
	}

	//edge cases
	if update.MessageReaction != nil {
		//copy the reaction
		//need to link the ID of the original post to the ID of the forwarded - might not be possible
	}
	if update.EditedMessage != nil {
		//edited message, include both old and new messages
		//editedmessage has no information about the original message :/
		if strconv.FormatInt(update.Message.Chat.ID, 10) == C.ConOpsChat {
			//from conops chat
			var topicID = update.Message.MessageThreadID
			if topicID == 0 {
				//in General thread - ignore
			} else {
				//in a user specific thread
				SendMessageToTopic(ctx, b, getTopicFromUser(C, strconv.FormatInt(update.EditedMessage.Chat.ID, 10)), "This bot does not support editing messages - if you need to clarify a detail, please send a new message")
			}
		} else {
			SendMessageToUser(ctx, b, strconv.FormatInt(update.EditedMessage.Chat.ID, 10), "This bot does not support editing messages - if you need to clarify a detail, please send a new message")
		}
	}

	//core forwarding logic
	if update.Message != nil {
		if update.Message.Text != "" || update.Message.Sticker != nil || update.Message.Photo != nil {
			if strconv.FormatInt(update.Message.Chat.ID, 10) == C.ConOpsChat {
				//this is from the conops chat
				//check if a topic exists
				var topicID = update.Message.MessageThreadID
				if topicID == 0 {
					//in General thread - ignore
				} else {
					//in a user specific thread
					var targetUser = getUserFromTopic(C, topicID)
					ForwardMessageFromTopic(ctx, b, update, topicID, targetUser)
				}

			} else {
				//this is from a user, get the topic ID
				//TODO: check if user is blocked
				if isUserBlocked(C, strconv.FormatInt(update.Message.Chat.ID, 10)) {
					//user is b locked - do nothing at this time. TODO: log later?
				} else {
					var topicID = getTopicFromUser(C, strconv.FormatInt(update.Message.Chat.ID, 10))
					if topicID != 0 {
						//user has messaged before
					} else {
						topicID = createTopic(ctx, b, update)
					}
					ForwardMessageToTopic(ctx, b, update, topicID)
				}
			}
		}
		//TODO: fix removing the topic edit message
		// if update.Message.ForumTopicEdited != nil {
		// 	//delete the message saying a forum topic was edited
		// 	b.DeleteMessage(ctx, &bot.DeleteMessageParams{
		// 		ChatID:    viper.GetString("conopschat"),
		// 		MessageID: update.Message.ID,
		// 	})
		// }
	}

	//if the message has an image, do we  need to do something different?

	//all other updates, can debug here
}

type config struct {
	ConOpsChat string
	BotToken   string
	Topics     map[string]int `mapstructure:"Topics"`
	Users      map[int]string `mapstructure:"Users"`
	Blocklist  map[string]int `mapstructure:"Blocklist"`
}

func getTopicFromUser(C config, user string) int {
	return C.Topics[user]
}

func getUserFromTopic(C config, topic int) string {
	return C.Users[topic]
}

func isUserBlocked(C config, user string) bool {
	_, ok := C.Blocklist[user]
	return ok
}

func createTopic(ctx context.Context, b *bot.Bot, update *models.Update) int {
	var createdTopic, _ = b.CreateForumTopic(ctx, &bot.CreateForumTopicParams{
		ChatID: viper.GetString("conopschat"),
		Name:   update.Message.From.FirstName + " " + update.Message.From.LastName,
	})
	viper.Set("Topics."+strconv.FormatInt(update.Message.Chat.ID, 10), createdTopic.MessageThreadID)
	viper.Set("Users."+strconv.Itoa(createdTopic.MessageThreadID), strconv.FormatInt(update.Message.Chat.ID, 10))
	viper.WriteConfig()
	return createdTopic.MessageThreadID
}

func ForwardMessageToTopic(ctx context.Context, b *bot.Bot, update *models.Update, topicID int) {
	b.ForwardMessage(ctx, &bot.ForwardMessageParams{
		ChatID:          viper.GetString("conopschat"),
		MessageThreadID: topicID,
		FromChatID:      update.Message.Chat.ID,
		MessageID:       update.Message.ID,
	})
	//set topic icon to eyeballs
	SetTopicIcon(ctx, b, topicID, false)
}

func ForwardMessageFromTopic(ctx context.Context, b *bot.Bot, update *models.Update, topicID int, targetUser string) {
	b.ForwardMessage(ctx, &bot.ForwardMessageParams{
		ChatID:     targetUser,
		FromChatID: viper.GetString("conopschat"),
		MessageID:  update.Message.ID,
	})
	//set topic icon to checkmark
	SetTopicIcon(ctx, b, topicID, true)
}

func SetTopicIcon(ctx context.Context, b *bot.Bot, topicID int, responded bool) {
	if responded {
		b.EditForumTopic(ctx, &bot.EditForumTopicParams{
			ChatID:            viper.GetString("conopschat"),
			MessageThreadID:   topicID,
			IconCustomEmojiID: "5237699328843200968",
		})
	} else {
		b.EditForumTopic(ctx, &bot.EditForumTopicParams{
			ChatID:            viper.GetString("conopschat"),
			MessageThreadID:   topicID,
			IconCustomEmojiID: "5417915203100613993",
		})
	}

}

func SendMessageToTopic(ctx context.Context, b *bot.Bot, topicID int, message string) {
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:          viper.GetString("conopschat"),
		MessageThreadID: topicID,
		Text:            message,
	})
}

func SendMessageToUser(ctx context.Context, b *bot.Bot, targetUser string, message string) {
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: targetUser,
		Text:   message,
	})
}

//hi, welcome to ConOps! This chat is your gateway to the ConOps team. If you haven't already, please send a message with your query now. Please also let us know your badge number to help us support you.
//The ConOps desk is currently closed - this chat is not being actively monitored, we make no guarantee that your query will be answered.

func blockHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	//first, check if this is in a thread
	//if it is, close the thread, add user to blocklist
	var C config
	err := viper.Unmarshal(&C)
	if err != nil {
		fmt.Printf("unable to decode into struct, %v", err)
	}

	if update.Message != nil {
		if strconv.FormatInt(update.Message.Chat.ID, 10) == C.ConOpsChat {
			var topicID = update.Message.MessageThreadID
			if topicID == 0 {
				//in General thread - ignore
			} else {
				//in a user specific thread
				//TODO: add to a blocklist in the config
				viper.Set("Blocklist."+getUserFromTopic(C, topicID), topicID)
			}
		}
	}
}

// func unblockHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
// 	//first, check if this is in a thread
// 	//if it is, close the thread, add user to blocklist
// 	var C config
// 	err := viper.Unmarshal(&C)
// 	if err != nil {
// 		fmt.Printf("unable to decode into struct, %v", err)
// 	}

// 	if update.Message != nil {
// 		if strconv.FormatInt(update.Message.Chat.ID, 10) == C.ConOpsChat {
// 			var topicID = update.Message.MessageThreadID
// 			if topicID == 0 {
// 				//in General thread - ignore
// 			} else {
// 				//in a user specific thread
// 				//TODO: add to a blocklist in the config
// 				//viper.Re("Blocklist."+getUserFromTopic(C, topicID), topicID)
// 			}
// 		}
// 	}
// }

func activateHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	//first, check if this is in a thread
	//if it is, close the thread, add user to blocklist
	var C config
	err := viper.Unmarshal(&C)
	if err != nil {
		fmt.Printf("unable to decode into struct, %v", err)
	}

	if update.Message != nil {
		if strconv.FormatInt(update.Message.Chat.ID, 10) == C.ConOpsChat {
			var topicID = update.Message.MessageThreadID
			if topicID == 0 {
				//in General thread
				//set config 'online' to true
			} else {
				//in a thread - ignore
			}
		}
	}
}

func deactivateHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	//first, check if this is in a thread
	//if it is, close the thread, add user to blocklist
	var C config
	err := viper.Unmarshal(&C)
	if err != nil {
		fmt.Printf("unable to decode into struct, %v", err)
	}

	if update.Message != nil {
		if strconv.FormatInt(update.Message.Chat.ID, 10) == C.ConOpsChat {
			var topicID = update.Message.MessageThreadID
			if topicID == 0 {
				//in General thread - ignore
			} else {
				//in a user specific thread
				//TODO: add to a blocklist in the config
			}
		}
	}
}
