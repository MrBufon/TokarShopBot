package handlers

import (
	"log"
	"strconv"
	"strings"

	"github.com/MrBufon/TokarShopBot/collections"
	"github.com/MrBufon/TokarShopBot/db"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var UserGoods = collections.NewMutexMap[int64, collections.Good]()

var UserStates = collections.NewMutexMap[int64, collections.UserState](128)

var UserRights = collections.NewMutexMap[int64, string]()

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

var BackKeyboard = tgbotapi.NewInlineKeyboardMarkup(
	tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("Назад", "back"),
	),
)

var StopFindKeyboard = tgbotapi.NewInlineKeyboardMarkup(
	tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("Прекратить поиск", "stopFind"),
	),
)

func InitUserRights() error {
	return db.FillPermissionMap(UserRights)
}

func IsAdmin(id int64) bool {
	right, ok := UserRights.Get(id)
	if ok && right == "admin" {
		return true
	}
	return false
}

func HandleAdd(message *tgbotapi.Message) tgbotapi.MessageConfig {
	userState, _ := UserStates.Get(message.Chat.ID)
	phase := userState.Phase

	switch phase {
	case 0:
		state, _ := UserStates.Get(message.Chat.ID)
		state.Phase = 1
		UserStates.Set(message.Chat.ID, state)
		UserGoods.Set(message.Chat.ID, collections.Good{Name: message.Text})
		return tgbotapi.NewMessage(message.Chat.ID, "Добавляем товар. Шаг 2:\nВведите количество товара")
	case 1:
		amount, err := strconv.ParseInt(message.Text, 10, 64)
		if err != nil {
			return tgbotapi.NewMessage(message.Chat.ID, "Введенное значение не является целым числом. Пожалуйста, введите новое значение")
		}
		good, _ := UserGoods.Get(message.Chat.ID)
		good.Amount = amount
		if err := db.InsertIntoGoods(good); err != nil {
			return tgbotapi.NewMessage(message.Chat.ID, "Возникла ошибка. Пожалуйста, введите количество товара снова либо попробуйте начать заного")
		}

		UserStates.Delete(message.Chat.ID)
		UserGoods.Delete(message.Chat.ID)

		msg := tgbotapi.NewMessage(message.Chat.ID, "Товар успешно добавлен")
		if IsAdmin(message.Chat.ID) {
			msg.ReplyMarkup = AdminStartKeyboard
		} else {
			msg.ReplyMarkup = CommonStartKeyboard
		}

		return msg
	default:
		UserStates.Delete(message.Chat.ID)
		UserGoods.Delete(message.Chat.ID)
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

	UserStates.Delete(message.Chat.ID)

	msg := tgbotapi.NewMessage(message.Chat.ID, "Товар успешно удалён")
	if IsAdmin(message.Chat.ID) {
		msg.ReplyMarkup = AdminStartKeyboard
	} else {
		msg.ReplyMarkup = CommonStartKeyboard
	}

	return msg
}

func HandleFind(message *tgbotapi.Message) tgbotapi.MessageConfig {
	goodsStr, err := db.FindInGoods(message.Text)
	msg := tgbotapi.NewMessage(message.Chat.ID, "")
	msg.ReplyMarkup = StopFindKeyboard

	if err != nil {
		msg.Text = "Возникла ошибка. Пожалуйста, введите новое значение"
		return msg
	} else if len(goodsStr) == 0 {
		msg.Text = "Товаров не найдено. Пожалуйста, введите новое значение"
		return msg
	}

	var sb strings.Builder
	sb.WriteString("Найденные товары:\n")
	sb.WriteString(goodsStr)
	msg.Text = sb.String()
	return msg
}

func HandleMessage(message *tgbotapi.Message) tgbotapi.MessageConfig {
	userState, ok := UserStates.Get(message.Chat.ID)
	state := userState.State
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
	case message.Text == "/cancel":
		if ok {
			UserStates.Delete(message.Chat.ID)
			UserGoods.Delete(message.Chat.ID)
			return tgbotapi.NewMessage(message.Chat.ID, "Операция завершена досрочно")
		}
		return tgbotapi.NewMessage(message.Chat.ID, "Вы не находитесь в процессе выполнения операции")
	case state == "adding":
		return HandleAdd(message)
	case state == "deleting":
		return HandleDelete(message)
	case state == "finding":
		return HandleFind(message)
	default:
		return tgbotapi.NewMessage(message.Chat.ID, "Неизвестная команда")
	}
}

func HandleQuery(bot *tgbotapi.BotAPI, cQuery *tgbotapi.CallbackQuery) tgbotapi.MessageConfig {
	switch cQuery.Data {
	case "add":
		if !IsAdmin(cQuery.Message.Chat.ID) {
			return tgbotapi.NewMessage(cQuery.Message.Chat.ID, "Нет прав")
		}

		UserGoods.Delete(cQuery.Message.Chat.ID)
		UserStates.Set(cQuery.Message.Chat.ID, collections.UserState{State: "adding", Phase: 0})
		_, err := bot.Request(tgbotapi.NewCallback(cQuery.ID, ""))
		if err != nil {
			log.Println("callback error:", err)
		}
		return tgbotapi.NewMessage(cQuery.Message.Chat.ID, "Добавляем товар. Шаг 1:\nВведите название товара")
	case "delete":
		if !IsAdmin(cQuery.Message.Chat.ID) {
			return tgbotapi.NewMessage(cQuery.Message.Chat.ID, "Нет прав")
		}

		UserGoods.Delete(cQuery.Message.Chat.ID)
		UserStates.Set(cQuery.Message.Chat.ID, collections.UserState{State: "deleting", Phase: 0})
		_, err := bot.Request(tgbotapi.NewCallback(cQuery.ID, ""))
		if err != nil {
			log.Println("callback error:", err)
		}
		return tgbotapi.NewMessage(cQuery.Message.Chat.ID, "Удаляем товары. Введите порядковый номер товара")
	case "find":
		UserGoods.Delete(cQuery.Message.Chat.ID)
		UserStates.Set(cQuery.Message.Chat.ID, collections.UserState{State: "finding", Phase: 0})
		_, err := bot.Request(tgbotapi.NewCallback(cQuery.ID, ""))
		if err != nil {
			log.Println("callback error:", err)
		}

		msg := tgbotapi.NewMessage(cQuery.Message.Chat.ID, "Ищем товары. Введите название товара")
		msg.ReplyMarkup = StopFindKeyboard
		return msg
	case "back":
		return tgbotapi.NewMessage(cQuery.Message.Chat.ID, "Неизвестная команда")
	case "stopFind":
		_, err := bot.Request(tgbotapi.NewCallback(cQuery.ID, ""))
		if err != nil {
			log.Println("callback error:", err)
		}

		msg := tgbotapi.NewMessage(cQuery.Message.Chat.ID, "")

		if userState, ok := UserStates.Get(cQuery.Message.Chat.ID); ok {
			if userState.State != "finding" {
				msg.Text = "Вы находитесь в процессе выполнения другой операции"
				return msg
			}
			UserStates.Delete(cQuery.Message.Chat.ID)

			if IsAdmin(cQuery.Message.Chat.ID) {
				msg.ReplyMarkup = AdminStartKeyboard
			} else {
				msg.ReplyMarkup = CommonStartKeyboard
			}
			msg.Text = "Операция поиска завершена"
			return msg
		}

		if IsAdmin(cQuery.Message.Chat.ID) {
			msg.ReplyMarkup = AdminStartKeyboard
		} else {
			msg.ReplyMarkup = CommonStartKeyboard
		}
		msg.Text = "Вы не находитесь в процессе выполнения операции"
		return msg
	default:
		UserStates.Delete(cQuery.Message.Chat.ID)
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
		return HandleMessage(message)
	} else if cQuery != nil {
		return HandleQuery(bot, cQuery)
	}

	return tgbotapi.NewMessage(0, "Неизвестный тип update")
}
