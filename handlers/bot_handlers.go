package handlers

import (
	"log"
	"strconv"
	"strings"

	"TokarTgBot/db"
	"TokarTgBot/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var good models.Good

var UserStates = make(map[int64]models.UserState, 128)

var UserRights = make(map[int64]string)

var CommonStartKeyboard = tgbotapi.NewInlineKeyboardMarkup(
	tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("Найти товар", "find"),
	),
)

var AdminStartKeyboard = tgbotapi.NewInlineKeyboardMarkup(
	tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("Найти товар", "find"),
	),
	tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("Добавить товар", "add"),
		tgbotapi.NewInlineKeyboardButtonData("Удалить товар", "delete"),
	),
)

func InitUserRights() error {
	return db.FillPermissionMap(UserRights)
}

func IsAdmin(id int64) bool {
	right, ok := UserRights[id]
	if ok && right == "admin" {
		return true
	}
	return false
}

func HandleAdd(message *tgbotapi.Message) tgbotapi.MessageConfig {
	phase := UserStates[message.Chat.ID].Phase

	switch phase {
	case 0:
		state := UserStates[message.Chat.ID]
		state.Phase = 1
		UserStates[message.Chat.ID] = state
		good.Name = message.Text
		return tgbotapi.NewMessage(message.Chat.ID, "Добавляем товар. Шаг 2:\nВведите количество товара")
	case 1:
		amount, err := strconv.ParseInt(message.Text, 10, 64)
		if err != nil {
			return tgbotapi.NewMessage(message.Chat.ID, "Введенное значение не является целым числом. Пожалуйста, введите новое значение")
		}
		good.Amount = amount
		if err := db.InsertIntoGoods(good); err != nil {
			return tgbotapi.NewMessage(message.Chat.ID, "Возникла ошибка. Пожалуйста, введите количество товара снова либо попробуйте начать заного")
		}
		delete(UserStates, message.Chat.ID)
		return tgbotapi.NewMessage(message.Chat.ID, "Товар успешно добавлен")
	default:
		delete(UserStates, message.Chat.ID)
		return tgbotapi.NewMessage(message.Chat.ID, "Возникла ошибка. Попробуйте снова")
	}
}

func HandleDelete(message *tgbotapi.Message) tgbotapi.MessageConfig {
	order, parseErr := strconv.ParseInt(message.Text, 10, 64)
	if parseErr != nil {
		return tgbotapi.NewMessage(message.Chat.ID, "Введенное значение не является целым числом. Пожалуйста, введите новое значение")
	}

	if dbErr := db.DeleteFromGoods(order); dbErr != nil {
		return tgbotapi.NewMessage(message.Chat.ID, "Возникла ошибка. Возможно, товара не существует. Пожалуйста, введите новое значение")
	}

	delete(UserStates, message.Chat.ID)
	return tgbotapi.NewMessage(message.Chat.ID, "Товар успешно удалён")
}

func HandleFind(message *tgbotapi.Message) tgbotapi.MessageConfig {
	goods, err := db.FindInGoods(message.Text)
	if err != nil {
		return tgbotapi.NewMessage(message.Chat.ID, "Возникла ошибка. Пожалуйста, введите новое значение")
	} else if len(goods) == 0 {
		return tgbotapi.NewMessage(message.Chat.ID, "Товаров не найдено. Пожалуйста, введите новое значение")
	}

	var sb strings.Builder
	sb.WriteString("Найденные товары:\n")
	sb.WriteString(goods)
	return tgbotapi.NewMessage(message.Chat.ID, sb.String())
}

func HandleMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message) tgbotapi.MessageConfig {
	switch {
	case message.Text == "/start":
		if IsAdmin(message.Chat.ID) {
			messageConfig := tgbotapi.NewMessage(message.Chat.ID, "Приветствую в панели администратора!")
			messageConfig.ReplyMarkup = AdminStartKeyboard
			return messageConfig
		} else {
			messageConfig := tgbotapi.NewMessage(message.Chat.ID, "Приветствую!")
			messageConfig.ReplyMarkup = CommonStartKeyboard
			return messageConfig
		}
	case UserStates[message.Chat.ID].State == "adding":
		return HandleAdd(message)
	case UserStates[message.Chat.ID].State == "deleting":
		return HandleDelete(message)
	case UserStates[message.Chat.ID].State == "finding":
		return HandleFind(message)
	default:
		return tgbotapi.NewMessage(message.Chat.ID, "Неизвестная команда")
	}
}

func HandleQuery(bot *tgbotapi.BotAPI, cQuery *tgbotapi.CallbackQuery) tgbotapi.MessageConfig {
	switch cQuery.Data {
	case "add":
		good = models.Good{}
		UserStates[cQuery.Message.Chat.ID] = models.UserState{State: "adding"}
		_, err := bot.Request(tgbotapi.NewCallback(cQuery.ID, ""))
		if err != nil {
			log.Println("callback error:", err)
		}
		return tgbotapi.NewMessage(cQuery.Message.Chat.ID, "Добавляем товар. Шаг 1:\nВведите название товара")
	case "delete":
		good = models.Good{}
		UserStates[cQuery.Message.Chat.ID] = models.UserState{State: "deleting"}
		_, err := bot.Request(tgbotapi.NewCallback(cQuery.ID, ""))
		if err != nil {
			log.Println("callback error:", err)
		}
		return tgbotapi.NewMessage(cQuery.Message.Chat.ID, "Удаляем товары. Введите порядковый номер товара")
	case "find":
		good = models.Good{}
		UserStates[cQuery.Message.Chat.ID] = models.UserState{State: "finding"}
		_, err := bot.Request(tgbotapi.NewCallback(cQuery.ID, ""))
		if err != nil {
			log.Println("callback error:", err)
		}
		return tgbotapi.NewMessage(cQuery.Message.Chat.ID, "Ищем товары. Введите название товара")
	default:
		_, err := bot.Request(tgbotapi.NewCallback(cQuery.ID, ""))
		if err != nil {
			log.Println("callback error:", err)
		}
		return tgbotapi.NewMessage(cQuery.Message.Chat.ID, "Неизвестная команда")
	}
}

func HandleUpdate(bot *tgbotapi.BotAPI, update tgbotapi.Update) tgbotapi.MessageConfig {
	message := update.Message
	cQuery := update.CallbackQuery

	if message != nil {
		return HandleMessage(bot, message)
	} else if cQuery != nil {
		return HandleQuery(bot, cQuery)
	}

	return tgbotapi.NewMessage(update.Message.Chat.ID, "Вообще белиберда какая-то")
}
