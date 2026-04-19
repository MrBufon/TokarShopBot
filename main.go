package main

import (
	"log"
	"os"

	"github.com/MrBufon/TokarShopBot/db"
	"github.com/MrBufon/TokarShopBot/handlers"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	if err := db.Init(); err != nil {
		log.Fatal(err)
	}

	log.Println("DB connected:", db.DB != nil)

	botToken := os.Getenv("BOT_TOKEN")

	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatal("Bot error:", err)
	}

	bot.Debug = true
	log.Printf("Bot authorized as @%s", bot.Self.UserName)

	if err := handlers.InitUserRights(); err != nil {
		log.Fatal(err)
	}

	const workerCount = 5

	updatesCh := make(chan tgbotapi.Update, 100)

	for range workerCount {
		go func() {
			for update := range updatesCh {
				if update.Message == nil && update.CallbackQuery == nil {
					continue
				}

				bot.Send(handlers.HandleUpdate(bot, update))
			}
		}()
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		updatesCh <- update
	}
}
