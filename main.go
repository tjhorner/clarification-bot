package main

import (
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/jinzhu/gorm"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

var clRegex = regexp.MustCompile("(?i)(what|huh)\\??")
var lastMessages = make(map[int64]string)
var db *gorm.DB

func shouldClarify(message string) bool {
	if len(strings.Split(message, " ")) >= 3 {
		return false
	}

	return clRegex.MatchString(message)
}

func escape(text string) string {
	text = strings.ReplaceAll(text, "&", "&amp;")
	text = strings.ReplaceAll(text, "<", "&lt;")
	text = strings.ReplaceAll(text, ">", "&gt;")

	return text
}

var optOutCache = make(map[int]bool) // true = opt out, false = opt in

func userOptedOut(user *tgbotapi.User) bool {
	if optedOut, ok := optOutCache[user.ID]; ok {
		return optedOut
	}

	var count int
	db.Model(&User{}).Where(&User{TelegramID: user.ID, OptedOut: true}).Count(&count)

	optedOut := count > 0
	optOutCache[user.ID] = optedOut

	return optedOut
}

func optUser(user *tgbotapi.User, in bool) {
	var dbUser User
	db.FirstOrCreate(&dbUser, &User{TelegramID: user.ID})

	optOutCache[user.ID] = !in
	dbUser.OptedOut = !in
	db.Save(&dbUser)
}

func clarify(update tgbotapi.Update, text string) *tgbotapi.MessageConfig {
	if len(text) >= 500 || userOptedOut(update.Message.From) {
		return nil
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "<b>"+strings.ToUpper(text)+"</b>")
	msg.ReplyToMessageID = update.Message.MessageID
	msg.ParseMode = tgbotapi.ModeHTML

	return &msg
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}

	return value
}

func main() {
	tok := os.Getenv("CLARIFICATION_BOT_TOKEN")
	if tok == "" {
		log.Fatalln("Please provide a bot token in the CLARIFICATION_BOT_TOKEN")
	}

	debug := getEnv("CLARIFICATION_BOT_DEBUG", "") != ""
	dbPath := getEnv("CLARIFICATION_BOT_DB_PATH", "./dev.db3")

	dbase, err := gorm.Open("sqlite3", dbPath)
	defer dbase.Close()

	dbase.AutoMigrate(&User{})

	db = dbase // global var

	bot, err := tgbotapi.NewBotAPI(tok)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = debug

	upd := tgbotapi.NewUpdate(0)
	upd.Timeout = 60

	updates, err := bot.GetUpdatesChan(upd)

	for update := range updates {
		if update.Message == nil || update.Message.Text == "" {
			continue
		}

		if update.Message.Text == "/clarify off" {
			optUser(update.Message.From, false)

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Opted out of clarification.")
			msg.ReplyToMessageID = update.Message.MessageID
			bot.Send(msg)

			continue
		} else if update.Message.Text == "/clarify on" {
			optUser(update.Message.From, true)

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Opted in to clarification.")
			msg.ReplyToMessageID = update.Message.MessageID
			bot.Send(msg)

			continue
		} else if update.Message.Text == "/clarify" {
			optedOut := userOptedOut(update.Message.From)

			opted := "IN"
			if optedOut {
				opted = "OUT"
			}

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Use this command to opt out of the bot clarifying when you prompt it to. Send either `/clarify on` or `/clarify off` to toggle. You are currently opted *"+opted+"*.")
			msg.ReplyToMessageID = update.Message.MessageID
			msg.ParseMode = tgbotapi.ModeMarkdown
			bot.Send(msg)

			continue
		}

		var msg *tgbotapi.MessageConfig

		if update.Message.ReplyToMessage != nil && update.Message.ReplyToMessage.Text != "" && !shouldClarify(update.Message.ReplyToMessage.Text) && shouldClarify(update.Message.Text) {
			msg = clarify(update, update.Message.ReplyToMessage.Text)
		} else if last, ok := lastMessages[update.Message.Chat.ID]; ok && !shouldClarify(last) && shouldClarify(update.Message.Text) {
			msg = clarify(update, last)
		}

		if msg != nil {
			bot.Send(*msg)
		}

		lastMessages[update.Message.Chat.ID] = update.Message.Text
	}
}
