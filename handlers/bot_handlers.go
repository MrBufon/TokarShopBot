package handlers

import (
	"fmt"
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
	tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("Редактировать товар", "edit"),
	),
)

var BackKeyboard = tgbotapi.NewInlineKeyboardMarkup(
	tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("Назад", "back"),
	),
)

var BackAndOkKeyboard = tgbotapi.NewInlineKeyboardMarkup(
	tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("Назад", "back"),
		tgbotapi.NewInlineKeyboardButtonData("Ок", "ok"),
	),
)

var BackAndNextAndOkKeyboard = tgbotapi.NewInlineKeyboardMarkup(
	tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("Назад", "back"),
		tgbotapi.NewInlineKeyboardButtonData("Далее", "next"),
	),
	tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("Ок", "ok"),
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

	msg := tgbotapi.NewMessage(message.From.ID, "")

	switch phase {
	case 0:
		state, _ := UserStates.Get(message.Chat.ID)
		state.Phase = 1
		UserStates.Set(message.Chat.ID, state)
		UserGoods.Set(message.Chat.ID, collections.Good{Name: message.Text})

		msg.ReplyMarkup = BackKeyboard
		msg.Text = "Добавляем товар. Шаг 2:\nВведите количество товара"
		return msg
	case 1:
		msg := tgbotapi.NewMessage(message.From.ID, "")

		amount, err := strconv.ParseInt(message.Text, 10, 64)
		if err != nil {
			msg.Text = "Введенное значение не является целым числом. Пожалуйста, введите новое значение"
			msg.ReplyMarkup = BackKeyboard
			return msg
		}

		if amount < 0 {
			msg.Text = "Количество товара не можнт быть отрицательным числом. Пожалуйста, введите новое значение"
			msg.ReplyMarkup = BackKeyboard
			return msg
		}

		good, _ := UserGoods.Get(message.Chat.ID)
		good.Amount = amount
		UserGoods.Set(message.From.ID, good)

		state, _ := UserStates.Get(message.Chat.ID)
		state.Phase = 2
		UserStates.Set(message.Chat.ID, state)

		var sb strings.Builder
		sb.WriteString("Будет добавлен следующий товар:\n")
		sb.WriteString(fmt.Sprintf("Наименование: %s | Количество: %d\n", good.Name, good.Amount))

		msg.Text = sb.String()
		msg.ReplyMarkup = BackAndOkKeyboard

		return msg
	case 2:
		good, _ := UserGoods.Get(message.Chat.ID)

		msg := tgbotapi.NewMessage(message.From.ID, "")

		var sb strings.Builder
		sb.WriteString("Выберите один из вариантов: \"Назад\" или \"Ок\"\n")
		sb.WriteString("В случае выбора \"Ок\" будет добавлен следующий товар:\n")
		sb.WriteString(fmt.Sprintf("Наименование: %s | Количество: %d\n", good.Name, good.Amount))
		sb.WriteString("в случае выбора \"Назад\" произойдет возвращение к предыдущему шагу")

		msg.Text = sb.String()
		msg.ReplyMarkup = BackAndOkKeyboard

		return msg
	default:
		UserStates.Delete(message.Chat.ID)
		UserGoods.Delete(message.Chat.ID)

		msg := tgbotapi.NewMessage(message.From.ID, "Возникла ошибка. Попробуйте снова")

		return msg
	}
}

func HandleDelete(message *tgbotapi.Message) tgbotapi.MessageConfig {
	msg := tgbotapi.NewMessage(message.Chat.ID, "")

	userState, _ := UserStates.Get(message.Chat.ID)
	phase := userState.Phase

	switch phase {
	case 0:
		id, parseErr := strconv.ParseInt(message.Text, 10, 64)
		if parseErr != nil {
			msg.Text = "Введенное значение не является целым числом. Пожалуйста, введите новое значение"
			msg.ReplyMarkup = BackKeyboard

			return msg
		}

		goodStr, dbErr := db.FindGoodStrInGoodsById(id)
		if dbErr != nil {
			msg.Text = "Возникла ошибка. Возможно, товара с данным номером не существует. Пожалуйста, попробуйте снова"
			msg.ReplyMarkup = BackKeyboard

			return msg
		}

		userState, _ := UserStates.Get(message.From.ID)
		userState.Phase = 1
		UserStates.Set(message.From.ID, userState)

		var sb strings.Builder
		sb.WriteString("Внимание, будет удален следующий товар:\n")
		sb.WriteString(goodStr)

		msg.Text = sb.String()
		msg.ReplyMarkup = BackAndOkKeyboard

		UserGoods.Set(message.From.ID, collections.Good{Id: id})

		return msg
	case 1:
		good, _ := UserGoods.Get(message.From.ID)
		id := good.Id

		goodStr, dbErr := db.FindGoodStrInGoodsById(id)
		if dbErr != nil {
			UserStates.Delete(message.From.ID)
			UserGoods.Delete(message.From.ID)

			msg.Text = "Возникла непредвиденная ошибка. Возвращение в главное меню"
			if IsAdmin(message.Chat.ID) {
				msg.ReplyMarkup = AdminStartKeyboard
			} else {
				msg.ReplyMarkup = CommonStartKeyboard
			}
			return msg
		}

		msg := tgbotapi.NewMessage(message.From.ID, "")

		var sb strings.Builder
		sb.WriteString("Выберите один из вариантов: \"Назад\" или \"Ок\"\n")
		sb.WriteString("В случае выбора \"Ок\" будет удален следующий товар:\n")
		sb.WriteString(goodStr)
		sb.WriteString("в случае выбора \"Назад\" произойдет возвращение к предыдущему шагу")

		msg.Text = sb.String()
		msg.ReplyMarkup = BackAndOkKeyboard

		return msg
	default:
		UserStates.Delete(message.Chat.ID)
		UserGoods.Delete(message.Chat.ID)

		msg := tgbotapi.NewMessage(message.From.ID, "Возникла ошибка. Попробуйте снова")

		return msg
	}
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

func HandleEdit(message *tgbotapi.Message) tgbotapi.MessageConfig {
	msg := tgbotapi.NewMessage(message.Chat.ID, "")

	userState, _ := UserStates.Get(message.Chat.ID)
	phase := userState.Phase

	switch phase {
	case 0:
		id, parseErr := strconv.ParseInt(message.Text, 10, 64)
		if parseErr != nil {
			msg.Text = "Введенное значение не является целым числом. Пожалуйста, введите новое значение"
			msg.ReplyMarkup = BackKeyboard

			return msg
		}

		if id < 0 {
			msg.Text = "Номер товара - это положительное число. Пожалуйста, введите новое значение"
			msg.ReplyMarkup = BackKeyboard

			return msg
		}

		good, dbErr := db.FindGoodInGoodsById(id)
		if dbErr != nil {
			msg.Text = "Возникла ошибка поиска товара. Возможно, товара не существует. Пожалуйста, введите новое значение"
			msg.ReplyMarkup = BackKeyboard

			return msg
		}

		userState.Phase = 1
		UserStates.Set(message.From.ID, userState)
		UserGoods.Set(message.From.ID, good)

		var sb strings.Builder
		sb.WriteString("Редактируется следующий товар:\n")
		sb.WriteString(fmt.Sprintf("Номер: %d | Наименование: %s | Количество: %d\n\n", good.Id, good.Name, good.Amount))
		sb.WriteString("Введите новое название товара. Если название менять не нужно, нажмите кнопку \"Далее\"\n")
		sb.WriteString("Для возвращения к предыдущему шагу, нажмите кнопку \"Назад\"\n")
		sb.WriteString("Для сохранения изменений, нажмите кнопку \"Ок\"\n")

		msg.Text = sb.String()
		msg.ReplyMarkup = BackAndNextAndOkKeyboard

		return msg
	case 1:
		good, _ := UserGoods.Get(message.From.ID)
		good.Name = message.Text
		UserGoods.Set(message.From.ID, good)

		userState.Phase = 2
		UserStates.Set(message.From.ID, userState)

		var sb strings.Builder
		sb.WriteString("Редактируется следующий товар:\n")
		sb.WriteString(fmt.Sprintf("Номер: %d | Наименование: %s | Количество: %d\n\n", good.Id, good.Name, good.Amount))
		sb.WriteString("Введите новое количество товара. Если количество менять не нужно, нажмите кнопку \"Далее\"\n")
		sb.WriteString("Для возвращения к предыдущему шагу, нажмите кнопку \"Назад\"\n")
		sb.WriteString("Для сохранения изменений, нажмите кнопку \"Ок\"\n")

		msg.Text = sb.String()
		msg.ReplyMarkup = BackAndNextAndOkKeyboard

		return msg
	case 2:
		amount, parseErr := strconv.ParseInt(message.Text, 10, 64)
		if parseErr != nil {
			msg.Text = "Введенное значение не является целым числом. Пожалуйста, введите новое значение или нажмите \"Ок\" для сохранения изменений"
			msg.ReplyMarkup = BackAndOkKeyboard

			return msg
		}

		if amount < 0 {
			msg.Text = "Количество товара не может быть отрицательным. Пожалуйста, введите новое значение или нажмите \"Ок\" для сохранения изменений"
			msg.ReplyMarkup = BackAndOkKeyboard

			return msg
		}

		good, _ := UserGoods.Get(message.From.ID)
		good.Amount = amount
		UserGoods.Set(message.From.ID, good)

		userState.Phase = 3
		UserStates.Set(message.From.ID, userState)

		var sb strings.Builder
		sb.WriteString("Редактируется следующий товар:\n")
		sb.WriteString(fmt.Sprintf("Номер: %d | Наименование: %s | Количество: %d\n\n", good.Id, good.Name, good.Amount))
		sb.WriteString("Для возвращения к предыдущему шагу, нажмите кнопку \"Назад\"\n")
		sb.WriteString("Для сохранения изменений, нажмите кнопку \"Ок\"\n")

		msg.Text = sb.String()
		msg.ReplyMarkup = BackAndOkKeyboard

		return msg
	default:
		UserStates.Delete(message.Chat.ID)
		UserGoods.Delete(message.Chat.ID)

		msg := tgbotapi.NewMessage(message.From.ID, "Возникла ошибка. Попробуйте снова")

		return msg
	}
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
		msg := tgbotapi.NewMessage(message.From.ID, "Неизвестная команда")

		if IsAdmin(message.Chat.ID) {
			msg.ReplyMarkup = AdminStartKeyboard
		} else {
			msg.ReplyMarkup = CommonStartKeyboard
		}

		if ok {
			UserStates.Delete(message.Chat.ID)
			UserGoods.Delete(message.Chat.ID)

			msg.Text = "Операция завершена досрочно"
			return msg
		}

		msg.Text = "Вы не находитесь в процессе выполнения операции"
		return msg
	case state == "adding":
		return HandleAdd(message)
	case state == "deleting":
		return HandleDelete(message)
	case state == "finding":
		return HandleFind(message)
	case state == "editing":
		return HandleEdit(message)
	default:
		msg := tgbotapi.NewMessage(message.From.ID, "Неизвестная команда")

		if IsAdmin(message.Chat.ID) {
			msg.ReplyMarkup = AdminStartKeyboard
		} else {
			msg.ReplyMarkup = CommonStartKeyboard
		}

		return msg
	}
}

func HandleBackAdd(cQuery *tgbotapi.CallbackQuery, userState collections.UserState) tgbotapi.MessageConfig {
	msg := tgbotapi.NewMessage(cQuery.From.ID, "")

	switch userState.Phase {
	case 0:
		UserStates.Delete(cQuery.From.ID)
		UserGoods.Delete(cQuery.From.ID)
		if IsAdmin(cQuery.From.ID) {
			msg.ReplyMarkup = AdminStartKeyboard
		} else {
			msg.ReplyMarkup = CommonStartKeyboard
		}
		msg.Text = "Операция добавления товара отменена"
		return msg
	case 1:
		userState.Phase = 0
		UserStates.Set(cQuery.From.ID, userState)
		UserGoods.Delete(cQuery.From.ID)

		msg.ReplyMarkup = BackKeyboard
		msg.Text = "Возвращение к шагу 1. Пожалуйста, введите название товара"
		return msg
	case 2:
		userState.Phase = 1
		UserStates.Set(cQuery.From.ID, userState)

		good, _ := UserGoods.Get(cQuery.From.ID)
		UserGoods.Set(cQuery.From.ID, collections.Good{Name: good.Name})

		msg.ReplyMarkup = BackKeyboard
		msg.Text = "Возвращение к шагу 2. Пожалуйста, введите количество товара"
		return msg
	default:
		UserStates.Delete(cQuery.From.ID)
		UserGoods.Delete(cQuery.From.ID)
		if IsAdmin(cQuery.From.ID) {
			msg.ReplyMarkup = AdminStartKeyboard
		} else {
			msg.ReplyMarkup = CommonStartKeyboard
		}
		msg.Text = "Неизвестная ошибка, возвращение к главному меню"
		return msg
	}
}

func HandleBackDelete(cQuery *tgbotapi.CallbackQuery, userState collections.UserState) tgbotapi.MessageConfig {
	msg := tgbotapi.NewMessage(cQuery.From.ID, "")

	switch userState.Phase {
	case 0:
		UserStates.Delete(cQuery.From.ID)
		UserGoods.Delete(cQuery.From.ID)
		if IsAdmin(cQuery.From.ID) {
			msg.ReplyMarkup = AdminStartKeyboard
		} else {
			msg.ReplyMarkup = CommonStartKeyboard
		}
		msg.Text = "Операция удаления товара отменена"
		return msg
	case 1:
		userState.Phase = 0
		UserStates.Set(cQuery.From.ID, userState)
		UserGoods.Delete(cQuery.From.ID)
		msg.ReplyMarkup = BackKeyboard
		msg.Text = "Возвращение к шагу 1. Пожалуйста, введите номер удаляемого товара"
		return msg
	default:
		UserStates.Delete(cQuery.From.ID)
		UserGoods.Delete(cQuery.From.ID)
		if IsAdmin(cQuery.From.ID) {
			msg.ReplyMarkup = AdminStartKeyboard
		} else {
			msg.ReplyMarkup = CommonStartKeyboard
		}
		msg.Text = "Неизвестная ошибка, возвращение к главному меню"
		return msg
	}
}

func HandleBackEdit(cQuery *tgbotapi.CallbackQuery, userState collections.UserState) tgbotapi.MessageConfig {
	msg := tgbotapi.NewMessage(cQuery.From.ID, "")

	switch userState.Phase {
	case 0:
		UserStates.Delete(cQuery.From.ID)
		UserGoods.Delete(cQuery.From.ID)
		if IsAdmin(cQuery.From.ID) {
			msg.ReplyMarkup = AdminStartKeyboard
		} else {
			msg.ReplyMarkup = CommonStartKeyboard
		}
		msg.Text = "Операция изменения товара отменена"
		return msg
	case 1:
		userState.Phase = 0
		UserStates.Set(cQuery.From.ID, userState)

		good, _ := UserGoods.Get(cQuery.From.ID)

		var sb strings.Builder
		sb.WriteString("Возвращение к выбору товара. Сейчас редактируется следующий товар:\n")
		sb.WriteString(fmt.Sprintf("Номер: %d | Наименование: %s | Количество: %d\n\n", good.Id, good.Name, good.Amount))
		sb.WriteString("Введите номер нового товара для редактирования. Если хотите продолжить редактирование текущего товара, то нажмите \"Далее\"\n")
		sb.WriteString("Для возвращения к предыдущему шагу, нажмите кнопку \"Назад\"\n")
		sb.WriteString("Для сохранения изменений, нажмите кнопку \"Ок\"\n")

		msg.Text = sb.String()
		msg.ReplyMarkup = BackAndNextAndOkKeyboard
		return msg
	case 2:
		userState.Phase = 1
		UserStates.Set(cQuery.From.ID, userState)

		good, _ := UserGoods.Get(cQuery.From.ID)

		var sb strings.Builder
		sb.WriteString("Возвращение к предыдущему шагу. Изменение наименования\n")
		sb.WriteString("Редактируется следующий товар:\n")
		sb.WriteString(fmt.Sprintf("Номер: %d | Наименование: %s | Количество: %d\n\n", good.Id, good.Name, good.Amount))
		sb.WriteString("Введите новое название товара. Если название менять не нужно, нажмите кнопку \"Далее\"\n")
		sb.WriteString("Для возвращения к предыдущему шагу, нажмите кнопку \"Назад\"\n")
		sb.WriteString("Для сохранения изменений, нажмите кнопку \"Ок\"\n")

		msg.Text = sb.String()
		msg.ReplyMarkup = BackAndNextAndOkKeyboard

		return msg
	case 3:
		userState.Phase = 2
		UserStates.Set(cQuery.From.ID, userState)

		good, _ := UserGoods.Get(cQuery.From.ID)

		var sb strings.Builder
		sb.WriteString("Возвращение к предыдущему шагу. Изменение количества товара\n")
		sb.WriteString("Редактируется следующий товар:\n")
		sb.WriteString(fmt.Sprintf("Номер: %d | Наименование: %s | Количество: %d\n\n", good.Id, good.Name, good.Amount))
		sb.WriteString("Введите новое количество товара. Если количество менять не нужно, нажмите кнопку \"Далее\"\n")
		sb.WriteString("Для возвращения к предыдущему шагу, нажмите кнопку \"Назад\"\n")
		sb.WriteString("Для сохранения изменений, нажмите кнопку \"Ок\"\n")

		msg.Text = sb.String()
		msg.ReplyMarkup = BackAndNextAndOkKeyboard

		return msg
	default:
		UserStates.Delete(cQuery.From.ID)
		UserGoods.Delete(cQuery.From.ID)
		if IsAdmin(cQuery.From.ID) {
			msg.ReplyMarkup = AdminStartKeyboard
		} else {
			msg.ReplyMarkup = CommonStartKeyboard
		}
		msg.Text = "Неизвестная ошибка, возвращение к главному меню"
		return msg
	}
}

func HandleBack(cQuery *tgbotapi.CallbackQuery) tgbotapi.MessageConfig {
	userState, _ := UserStates.Get(cQuery.Message.Chat.ID)
	state := userState.State

	switch state {
	case "adding":
		return HandleBackAdd(cQuery, userState)
	case "deleting":
		return HandleBackDelete(cQuery, userState)
	case "finding":
		msg := tgbotapi.NewMessage(cQuery.From.ID, "У операции поиска есть собственная кнопка отмены. Пожжалуйста, используйте ее")
		msg.ReplyMarkup = StopFindKeyboard

		return msg
	case "editing":
		return HandleBackEdit(cQuery, userState)
	default:
		msg := tgbotapi.NewMessage(cQuery.From.ID, "Вы не находитесь в процессе выполнения операции")
		if IsAdmin(cQuery.From.ID) {
			msg.ReplyMarkup = AdminStartKeyboard
		} else {
			msg.ReplyMarkup = CommonStartKeyboard
		}
		return msg
	}
}

func HandleOk(cQuery *tgbotapi.CallbackQuery) tgbotapi.MessageConfig {
	userState, _ := UserStates.Get(cQuery.Message.Chat.ID)
	state := userState.State
	phase := userState.Phase

	msg := tgbotapi.NewMessage(cQuery.From.ID, "")

	switch state {
	case "adding":
		if phase != 2 {
			msg.Text = "Вы не находитесь в финальной фазе добавления товара. Пожалуйста, не нажимайте кнопки просто так.\n"
			return msg
		}

		good, _ := UserGoods.Get(cQuery.From.ID)

		if err := db.InsertIntoGoods(good); err != nil {
			msg.Text = "Возникла ошибка. Пожалуйста, введите количество товара снова либо попробуйте начать заного"
			return msg
		}

		UserStates.Delete(cQuery.From.ID)
		UserGoods.Delete(cQuery.From.ID)

		msg.Text = "Товар успешно добавлен"
		if IsAdmin(cQuery.From.ID) {
			msg.ReplyMarkup = AdminStartKeyboard
		} else {
			msg.ReplyMarkup = CommonStartKeyboard
		}

		return msg
	case "deleting":
		if phase != 1 {
			msg.Text = "Вы не находитесь в финальной фазе удаления товара. Пожалуйста, не нажимайте кнопки просто так.\n"
			return msg
		}

		good, _ := UserGoods.Get(cQuery.From.ID)
		id := good.Id

		if dbErr := db.DeleteFromGoods(id); dbErr != nil {
			msg.Text = "Возникла ошибка. Возможно, товара не существует. Пожалуйста, введите новое значение"
			return msg
		}

		UserStates.Delete(cQuery.From.ID)
		UserGoods.Delete(cQuery.From.ID)

		msg.Text = "Товар успешно удалён"
		if IsAdmin(cQuery.From.ID) {
			msg.ReplyMarkup = AdminStartKeyboard
		} else {
			msg.ReplyMarkup = CommonStartKeyboard
		}

		return msg
	case "finding":
		msg.Text = "Пожалуйста, используйте кнопки по назначению. Если вы хотите прекратить поиск товаров, нажмите кнопку снизу"
		msg.ReplyMarkup = StopFindKeyboard

		return msg
	case "editing":
		good, _ := UserGoods.Get(cQuery.From.ID)

		if err := db.EditInGoods(good); err != nil {
			msg.Text = "Возникла ошибка. Возможно, редактируемого товара не существует. Пожалуйста, введите новое значение"
			return msg
		}

		UserStates.Delete(cQuery.From.ID)
		UserGoods.Delete(cQuery.From.ID)

		msg.Text = "Товар успешно изменён"
		if IsAdmin(cQuery.From.ID) {
			msg.ReplyMarkup = AdminStartKeyboard
		} else {
			msg.ReplyMarkup = CommonStartKeyboard
		}

		return msg
	default:
		msg.Text = "Вы не находитесь в процессе выполнения операции"
		if IsAdmin(cQuery.From.ID) {
			msg.ReplyMarkup = AdminStartKeyboard
		} else {
			msg.ReplyMarkup = CommonStartKeyboard
		}
		return msg
	}
}

func HandleNextEdit(cQuery *tgbotapi.CallbackQuery) tgbotapi.MessageConfig {
	userState, _ := UserStates.Get(cQuery.Message.Chat.ID)
	phase := userState.Phase

	msg := tgbotapi.NewMessage(cQuery.From.ID, "")

	switch phase {
	case 0:
		good, ok := UserGoods.Get(cQuery.From.ID)
		if !ok {
			msg.Text = "Сначала выберите товар для редактирования. Для этого введите порядковый номер интересующего товара"
			msg.ReplyMarkup = BackKeyboard
			return msg
		}

		userState.Phase = 1
		UserStates.Set(cQuery.From.ID, userState)

		var sb strings.Builder
		sb.WriteString("Переходим к следующему шагу. Изменяем наименование товара\n")
		sb.WriteString("Редактируется следующий товар:\n")
		sb.WriteString(fmt.Sprintf("Номер: %d | Наименование: %s | Количество: %d\n\n", good.Id, good.Name, good.Amount))
		sb.WriteString("Введите новое название товара. Если название менять не нужно, нажмите кнопку \"Далее\"\n")
		sb.WriteString("Для возвращения к предыдущему шагу, нажмите кнопку \"Назад\"\n")
		sb.WriteString("Для сохранения изменений, нажмите кнопку \"Ок\"\n")

		msg.Text = sb.String()
		msg.ReplyMarkup = BackAndNextAndOkKeyboard

		return msg
	case 1:
		userState.Phase = 2
		UserStates.Set(cQuery.From.ID, userState)

		good, _ := UserGoods.Get(cQuery.From.ID)

		var sb strings.Builder
		sb.WriteString("Переходим к следующему шагу. Изменяем количество товара\n")
		sb.WriteString("Редактируется следующий товар:\n")
		sb.WriteString(fmt.Sprintf("Номер: %d | Наименование: %s | Количество: %d\n\n", good.Id, good.Name, good.Amount))
		sb.WriteString("Введите новое количество товара. Если количество менять не нужно, нажмите кнопку \"Далее\"\n")
		sb.WriteString("Для возвращения к предыдущему шагу, нажмите кнопку \"Назад\"\n")
		sb.WriteString("Для сохранения изменений, нажмите кнопку \"Ок\"\n")

		msg.Text = sb.String()
		msg.ReplyMarkup = BackAndNextAndOkKeyboard

		return msg
	case 2:
		userState.Phase = 3
		UserStates.Set(cQuery.From.ID, userState)

		good, _ := UserGoods.Get(cQuery.From.ID)

		var sb strings.Builder
		sb.WriteString("Редактируется следующий товар:\n")
		sb.WriteString(fmt.Sprintf("Номер: %d | Наименование: %s | Количество: %d\n\n", good.Id, good.Name, good.Amount))
		sb.WriteString("Для возвращения к предыдущему шагу, нажмите кнопку \"Назад\"\n")
		sb.WriteString("Для сохранения изменений, нажмите кнопку \"Ок\"\n")

		msg.Text = sb.String()
		msg.ReplyMarkup = BackAndOkKeyboard

		return msg
	case 3:
		good, _ := UserGoods.Get(cQuery.From.ID)

		var sb strings.Builder
		sb.WriteString("Пожалуйста, не нажимайте кнопки просто так\n")
		sb.WriteString("Редактируется следующий товар:\n")
		sb.WriteString(fmt.Sprintf("Номер: %d | Наименование: %s | Количество: %d\n\n", good.Id, good.Name, good.Amount))
		sb.WriteString("Для возвращения к предыдущему шагу, нажмите кнопку \"Назад\"\n")
		sb.WriteString("Для сохранения изменений, нажмите кнопку \"Ок\"\n")

		msg.Text = sb.String()
		msg.ReplyMarkup = BackAndOkKeyboard

		return msg
	default:
		UserStates.Delete(cQuery.From.ID)
		UserGoods.Delete(cQuery.From.ID)
		if IsAdmin(cQuery.From.ID) {
			msg.ReplyMarkup = AdminStartKeyboard
		} else {
			msg.ReplyMarkup = CommonStartKeyboard
		}
		msg.Text = "Неизвестная ошибка, возвращение к главному меню"
		return msg
	}
}

func HandleNext(cQuery *tgbotapi.CallbackQuery) tgbotapi.MessageConfig {
	userState, _ := UserStates.Get(cQuery.Message.Chat.ID)
	state := userState.State

	switch state {
	case "editing":
		return HandleNextEdit(cQuery)
	default:
		msg := tgbotapi.NewMessage(cQuery.From.ID, "")
		msg.Text = "Вы не находитесь в процессе выполнения операции"
		if IsAdmin(cQuery.From.ID) {
			msg.ReplyMarkup = AdminStartKeyboard
		} else {
			msg.ReplyMarkup = CommonStartKeyboard
		}
		return msg
	}
}

func HandleQuery(bot *tgbotapi.BotAPI, cQuery *tgbotapi.CallbackQuery) tgbotapi.MessageConfig {
	msg := tgbotapi.NewMessage(cQuery.From.ID, "")
	switch cQuery.Data {
	case "add":
		if !IsAdmin(cQuery.Message.Chat.ID) {
			msg.ReplyMarkup = CommonStartKeyboard
			msg.Text = "Нет прав"
			return msg
		}

		UserGoods.Delete(cQuery.Message.Chat.ID)
		UserStates.Set(cQuery.Message.Chat.ID, collections.UserState{State: "adding", Phase: 0})
		_, err := bot.Request(tgbotapi.NewCallback(cQuery.ID, ""))
		if err != nil {
			log.Println("callback error:", err)
		}

		msg.ReplyMarkup = BackKeyboard
		msg.Text = "Добавляем товар. Шаг 1:\nВведите название товара"
		return msg
	case "delete":
		if !IsAdmin(cQuery.Message.Chat.ID) {
			msg.ReplyMarkup = CommonStartKeyboard
			msg.Text = "Нет прав"
			return msg
		}

		UserGoods.Delete(cQuery.Message.Chat.ID)
		UserStates.Set(cQuery.Message.Chat.ID, collections.UserState{State: "deleting", Phase: 0})
		_, err := bot.Request(tgbotapi.NewCallback(cQuery.ID, ""))
		if err != nil {
			log.Println("callback error:", err)
		}

		msg.Text = "Удаляем товары. Введите порядковый номер товара"
		msg.ReplyMarkup = BackKeyboard
		return msg
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
	case "edit":
		_, err := bot.Request(tgbotapi.NewCallback(cQuery.ID, ""))
		if err != nil {
			log.Println("callback error:", err)
		}

		UserGoods.Delete(cQuery.Message.Chat.ID)
		UserStates.Set(cQuery.Message.Chat.ID, collections.UserState{State: "editing", Phase: 0})

		msg := tgbotapi.NewMessage(cQuery.Message.Chat.ID, "Редактируем товар. Введите номер товара")
		msg.ReplyMarkup = BackKeyboard
		return msg
	case "back":
		_, err := bot.Request(tgbotapi.NewCallback(cQuery.ID, ""))
		if err != nil {
			log.Println("callback error:", err)
		}

		return HandleBack(cQuery)
	case "stopFind":
		_, err := bot.Request(tgbotapi.NewCallback(cQuery.ID, ""))
		if err != nil {
			log.Println("callback error:", err)
		}

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
		msg.Text = "Вы не находитесь в процессе выполнения операции поиска"
		return msg
	case "ok":
		_, err := bot.Request(tgbotapi.NewCallback(cQuery.ID, ""))
		if err != nil {
			log.Println("callback error:", err)
		}

		return HandleOk(cQuery)
	case "next":
		_, err := bot.Request(tgbotapi.NewCallback(cQuery.ID, ""))
		if err != nil {
			log.Println("callback error:", err)
		}

		return HandleNext(cQuery)

	default:
		UserStates.Delete(cQuery.Message.Chat.ID)
		_, err := bot.Request(tgbotapi.NewCallback(cQuery.ID, ""))
		if err != nil {
			log.Println("callback error:", err)
		}

		msg.Text = "Неизвестная команда"
		return msg
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
