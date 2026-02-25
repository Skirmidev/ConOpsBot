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
	viper.SetConfigName("config")
	viper.SetConfigType("json")
	viper.AddConfigPath(".")

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

	b.Start(ctx)
}

func handler(ctx context.Context, b *bot.Bot, update *models.Update) {
	var C config
	err := viper.Unmarshal(&C)
	if err != nil {
		fmt.Printf("unable to decode into struct, %v", err)
	}

	if update.Message != nil {

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
			var topicID = getTopicFromUser(C, strconv.FormatInt(update.Message.Chat.ID, 10))
			if topicID != 0 {
				//user has messaged before
			} else {
				createTopic(ctx, b, update)
			}
			ForwardMessageToTopic(ctx, b, update, topicID)
		}
	}

	if update.EditedMessage != nil {

	}

	//all other updates, can debug here
}

type config struct {
	ConOpsChat string
	BotToken   string
	Topics     map[string]int `mapstructure:"Topics"`
	Users      map[int]string `mapstructure:"Users"`
}

func getTopicFromUser(C config, user string) int {
	return C.Topics[user]
}

func getUserFromTopic(C config, topic int) string {
	return C.Users[topic]
}

func createTopic(ctx context.Context, b *bot.Bot, update *models.Update) {
	var createdTopic, _ = b.CreateForumTopic(ctx, &bot.CreateForumTopicParams{
		ChatID: viper.GetString("conopschat"),
		Name:   update.Message.From.FirstName + " " + update.Message.From.LastName,
	})
	viper.Set("Topics."+strconv.FormatInt(update.Message.Chat.ID, 10), createdTopic.MessageThreadID)
	viper.Set("Users."+strconv.Itoa(createdTopic.MessageThreadID), strconv.FormatInt(update.Message.Chat.ID, 10))
	viper.WriteConfig()
}

func ForwardMessageToTopic(ctx context.Context, b *bot.Bot, update *models.Update, topicID int) {
	b.ForwardMessage(ctx, &bot.ForwardMessageParams{
		ChatID:          viper.GetString("conopschat"),
		MessageThreadID: topicID,
		FromChatID:      update.Message.Chat.ID,
		MessageID:       update.Message.ID,
	})
}

func ForwardMessageFromTopic(ctx context.Context, b *bot.Bot, update *models.Update, topicID int, targetUser string) {
	//
	b.ForwardMessage(ctx, &bot.ForwardMessageParams{
		ChatID:     targetUser,
		FromChatID: viper.GetString("conopschat"),
		MessageID:  update.Message.ID,
	})
}
