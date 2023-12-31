package telegram

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"tg_ics_useful_bot/clients/telegram"
	"tg_ics_useful_bot/lib/e"
	"tg_ics_useful_bot/storage"
	"time"
)

// dickTopExec предоставляет метод Exec для выполнения /top_dick.
type dickTopExec string

// Exec: /top_dick - пишет топ всех пенисов в чат.
func (a dickTopExec) Exec(p *Processor, inMessage string, user *telegram.User, chat *telegram.Chat,
	userStats *storage.DBUserStat, messageID int) (*Response, error) {
	message, err := p.topDicksCmd(chat.ID)
	if err != nil {
		return nil, e.Wrap(fmt.Sprintf("can't get top dics from chat %d: ", chat.ID), err)
	}
	mthd := sendMessageMethod
	return &Response{message: message, method: mthd, replyMessageId: -1, parseMode: telegram.Markdown}, nil
}

// dickStartExec предоставляет метод Exec для выполнения /dick.
type dickStartExec string

// Exec: /dick - игра в пенис.
func (a dickStartExec) Exec(p *Processor, inMessage string, user *telegram.User, chat *telegram.Chat,
	userStats *storage.DBUserStat, messageID int) (*Response, error) {
	message, err := p.gameDickCmd(chat, user, userStats)
	if err != nil {
		return nil, e.Wrap("can't get message from gameDickCmd: ", err)
	}
	mthd := sendMessageMethod
	return &Response{message: message, method: mthd, replyMessageId: -1}, nil
}

// topDicksCmd возвращает string сообщение со списком всех dick > 0 в чате.
func (p *Processor) topDicksCmd(chatID int) (msg string, err error) {
	users, err := p.storage.UsersByChat(context.Background(), chatID)
	if err != nil {
		return "", e.Wrap("[ERROR] can't get users: ", err)
	}

	result := ""
	for i, u := range users {
		if u.DickSize > 0 {
			if i == 0 {
				result += fmt.Sprintf("👑 *%s* — _%d см_\n", u.FirstName+" "+u.LastName, u.DickSize)
			} else {
				result += fmt.Sprintf("%d. %s — %d см\n", i+1, u.FirstName+" "+u.LastName, u.DickSize)
			}
		}
	}
	return result, nil
}

// gameDickCmd это функция изменяющая размер пениса на случайное число и время изменения пениса.
// /dick - command
// Возвращает сообщение, отправляемое в чат.
func (p *Processor) gameDickCmd(chat *telegram.Chat, user *telegram.User, userStats *storage.DBUserStat) (msg string, err error) {
	defer func() { err = e.WrapIfErr("error in gameDickCmd: ", err) }()

	dbUser, err := p.storage.GetUser(context.Background(), user.ID, chat.ID)
	if err != nil {
		return "", err
	}

	canChange, err := p.canChangeDickSize(dbUser)
	if err != nil {
		return "", err
	}

	if canChange {
		oldDickSize := dbUser.DickSize
		err = p.updateRandomDickAndChangeTime(dbUser, userStats)
		if err != nil {
			return "", err
		}
		if oldDickSize == 0 {
			return fmt.Sprintf(msgCreateUser, dbUser.Username) + fmt.Sprintf(msgDickSize, dbUser.DickSize), nil
		}
		return fmt.Sprintf(msgChangeDickSize, dbUser.Username, oldDickSize, dbUser.DickSize), nil
	}
	return fmt.Sprintf(msgAlreadyPlays, dbUser.Username), nil
}

// updateRandomDickAndChangeTime изменяет значение пениса на слуайное число и время его изменения в базе данных.
func (p *Processor) updateRandomDickAndChangeTime(user *storage.DBUser, userStats *storage.DBUserStat) error {
	var value int
	for {
		value = RandomValue()
		if user.DickSize+value > 0 {
			break
		}
	}

	if IsJackpot() {
		value = 100
	}

	if value > 0 {
		userStats.DickPlusCount++
		err := p.storage.UpdateUserStats(context.Background(), userStats)
		if err != nil {
			log.Print(err)
		}
	} else {
		userStats.DickMinusCount++
		err := p.storage.UpdateUserStats(context.Background(), userStats)
		if err != nil {
			log.Print(err)
		}
	}

	user.DickSize += value
	user.ChangeDickAt = time.Now()
	err := p.storage.UpdateUser(context.Background(), user)
	if err != nil {
		return e.Wrap(fmt.Sprintf("chat id %d, user %s can't change dick size or change dick at: ", user.ChatID, user.Username), err)
	}
	return nil
}

// canChangeDickSize - может ли пользователь изменить пенис сегодня. (остались ли у него попытки)
// Обновляет попытки каждый день до 0.
func (p *Processor) canChangeDickSize(user *storage.DBUser) (bool, error) {
	yearLastTry, monthLastTry, dayLastTry := user.ChangeDickAt.Date()
	year, month, today := time.Now().Date()
	if (month == monthLastTry && today > dayLastTry) || month > monthLastTry || year > yearLastTry {
		user.CurDickChangeCount = 0
		err := p.storage.UpdateUser(context.Background(), user)
		if err != nil {
			return false, e.Wrap("can't update user in 'canChangeDickSize'", err)
		}
	}
	if user.CurDickChangeCount+1 <= user.MaxDickChangeCount {
		user.CurDickChangeCount++
		err := p.storage.UpdateUser(context.Background(), user)
		if err != nil {
			return false, e.Wrap("can't update user in 'canChangeDickSize'", err)
		}
		return true, nil
	}

	return false, nil
}

// RandomValue возвращает случайное положительное или отрицательное число в конкретном диапозоне.
func RandomValue() int {
	sign := rand.Intn(10)
	value := rand.Intn(15)

	if value == 0 {
		value++
	}

	if sign > 1 {
		return value
	}
	return -1 * value
}

// IsJackpot показывает выиграл ли пользователь джекпот.
func IsJackpot() bool {
	if value := rand.Intn(100); value == 77 {
		return true
	}
	return false
}
