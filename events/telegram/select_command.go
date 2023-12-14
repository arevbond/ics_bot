package telegram

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"tg_ics_useful_bot/clients/jokesrv"
	"tg_ics_useful_bot/clients/telegram"
	"tg_ics_useful_bot/clients/xkcd"
	"tg_ics_useful_bot/lib/e"
	"tg_ics_useful_bot/lib/schedule"
	"tg_ics_useful_bot/lib/utils"
	"tg_ics_useful_bot/storage"
)

// selectCommand select one of available commands.
func (p *Processor) selectCommand(cmd string, chat *telegram.Chat, user *telegram.User, userStats *storage.DBUserStat,
	messageID int) (string, method, string, int, error) {
	var message string
	var mthd method
	var parseMode string
	var replyMessageId int
	var err error

	userWithChat := UserWithChat{chat.ID, user.ID}

	if _, ok := stateHomework[userWithChat]; ok {
		message = p.addHomeworkCmd(cmd, userWithChat)
		mthd = sendMessageMethod
		replyMessageId = messageID
	}

	switch {
	case isCommand(cmd, AllCmd):
		message = p.allUsernamesCmd(chat.ID)
		mthd = sendMessageMethod

	case isCommand(cmd, GayTopCmd):
		message, err = p.topGaysCmd(chat.ID)
		if err != nil {
			return "", UnsupportedMethod, parseMode, replyMessageId, e.Wrap("can't do GayTopCmd: ", err)
		}
		mthd = sendMessageMethod

	case isCommand(cmd, GayStartCmd):
		message, err = p.gameGayCmd(chat.ID)
		if err != nil {
			return "", UnsupportedMethod, parseMode, replyMessageId, e.Wrap("can't get message from gameGayCmd: ", err)
		}
		mthd = sendMessageMethod

	case isCommand(cmd, DickTopCmd):
		message, err = p.topDicksCmd(chat.ID)
		if err != nil {
			return "", UnsupportedMethod, parseMode, replyMessageId, e.Wrap(fmt.Sprintf("can't get top dics from chat %d: ", chat.ID), err)
		}
		mthd = sendMessageMethod

	case isCommand(cmd, DicStartCmd):
		err = p.tg.DeleteMessage(chat.ID, messageID)
		if err != nil {
			return "", UnsupportedMethod, parseMode, replyMessageId, e.Wrap(fmt.Sprintf("can't delete message: user #%d, chat id #%d", user.ID, chat.ID), err)
		}
		message, err = p.gameDickCmd(chat, user, userStats)
		if err != nil {
			return "", UnsupportedMethod, parseMode, replyMessageId, e.Wrap("can't get message from gameDickCmd: ", err)
		}
		mthd = sendMessageMethod

	// DUEL
	case isCommand(strings.Split(cmd, " ")[0], DickDuelCmd) || isCommand(cmd, DickDuelCmd):
		err = p.tg.DeleteMessage(chat.ID, messageID)
		if err != nil {
			return "", UnsupportedMethod, parseMode, replyMessageId, e.Wrap("can't delete message: ", err)
		}

		message, err = p.gameDuelCmd(chat, user, user.Username)
		if err != nil {
			return "", UnsupportedMethod, parseMode, replyMessageId, e.Wrap("can't do gameDuelCmd: ", err)
		}
		if utils.StringContains("@", cmd) {
			textSplited := strings.Split(cmd, "@")
			target := textSplited[len(textSplited)-1]
			log.Printf("[INFO] @%s вызывает на дуель @%s", user.Username, target)
			message, err = p.gameDuelCmd(chat, user, target)
			if err != nil {
				return "", UnsupportedMethod, parseMode, replyMessageId, e.Wrap("can't do gameDuelCmd: ", err)
			}
		}
		mthd = sendMessageMethod

	case isCommand(cmd, GetHPCmd):
		err = p.tg.DeleteMessage(chat.ID, messageID)
		if err != nil {
			return "", UnsupportedMethod, parseMode, replyMessageId, e.Wrap("can't delete message: ", err)
		}

		message, err = p.getHpCmd(user, chat)
		if err != nil {
			return "", UnsupportedMethod, parseMode, replyMessageId, e.Wrap("can't get hp in 'selectCommand':", err)
		}
		mthd = sendMessageMethod
	// END DUEL	COMMANDS

	case isCommand(cmd, XkcdCmd):
		var comics xkcd.Comics
		comics, err = xkcd.RandomComics()
		if err != nil {
			return "", UnsupportedMethod, parseMode, replyMessageId, e.Wrap("can't get comics from xkcd: ", err)
		}
		message = comics.Img
		mthd = sendPhotoMethod

	case isCommand(cmd, AnecdotCmd):
		message, err = jokesrv.Anecdot()
		if err != nil {
			return "", UnsupportedMethod, parseMode, replyMessageId, e.Wrap("can't get anecdot: ", err)
		}
		mthd = sendMessageMethod
	case isCommand(cmd, FlipCmd):
		message = hinkOrRoomCmd()
		mthd = sendPhotoMethod
	case isCommand(cmd, ScheduleCmd):
		calendarID, err := p.storage.GetCalendarID(context.Background(), chat.ID)
		if err != nil || calendarID == "" {
			message = msgCalendarNotExists
			log.Print("can't get calendarID: ", err)
		} else {
			message, err = schedule.ScheduleCmd(calendarID)
			parseMode = "Markdown"
			if err != nil {
				log.Printf("[ERROR] can't send schedule: %v", err)
				message = fmt.Sprintf(msgErrorSendMessage, calendarID)
				parseMode = ""
			}
		}
		mthd = sendMessageMethod
	case isCommand(strings.Split(cmd, " ")[0], AddCalendarIDCmd):
		if !p.isChatAdmin(user, chat.ID) {
			return msgForbiddenCalendarUpdate, sendMessageMethod, parseMode, replyMessageId, nil
		}
		strs := strings.Split(cmd, " ")
		calendarID := ""
		for _, str := range strs {
			if len(str) > 0 {
				calendarID = str
			}
		}
		err = p.storage.AddCalendarID(context.Background(), chat.ID, calendarID)
		if err != nil {
			message = fmt.Sprintf(msgErrorUpdateCalendarID, calendarID)
			log.Printf("can't update calender_id: %v", err)
		} else {
			message = msgSuccessUpdateCalendarID
		}
		mthd = sendMessageMethod

	case isCommand(cmd, AddHomeworkCmd):
		message = p.addHomeworkCmd(cmd, userWithChat)
		mthd = sendMessageWithButtonsMethod
		replyMessageId = messageID
	case isCommand(cmd, GetHomeworkCmd) || isCommand(strings.Split(cmd, " ")[0], GetHomeworkCmd):
		message = p.getHomeworkdCmd(cmd, chat.ID)
		mthd = sendMessageMethod
	case isCommand(cmd, CancelHomeworkCmd):
		if _, ok := stateHomework[userWithChat]; ok {
			delete(stateHomework, userWithChat)
			message = msgHomeworkCanceled
			mthd = sendMessageMethod
			replyMessageId = messageID
		}
	case isCommand(strings.Split(cmd, " ")[0], DeleteHomeworkCmd):
		val := ""
		for _, str := range strings.Split(cmd, " ")[1:] {
			if str != "" {
				val = str
				break
			}
		}
		num, err := strconv.Atoi(val)
		message = p.DeleteHomework(num)
		if err != nil {
			message = fmt.Sprintf("%s - некоректное значение id", val)
		}
		mthd = sendMessageMethod

	case isCommand(cmd, HelpCmd):
		message = msgHelp
		mthd = sendMessageMethod
		parseMode = "Markdown"

	case isCommand(strings.Split(cmd, " ")[0], ChangeDickCmd):
		strs := strings.Split(cmd, " ")
		chatIDStr, userIDStr, valueStr := strs[1], strs[2], strs[3]
		err = p.changeDickByAdminCmd(chatIDStr, userIDStr, valueStr)
		if err != nil {
			log.Print(err)
			return message, mthd, parseMode, replyMessageId, err
		}
		message = msgSuccessAdminChangeDickSize
		mthd = sendMessageMethod
	case isCommand(strings.Split(cmd, " ")[0], SendMessageByAdminCmd):
		strs := strings.Split(cmd, " ")
		chatIDStr, message := strs[1], strings.Join(strs[2:], " ")
		chatID, err := strconv.Atoi(chatIDStr)
		if err != nil {
			log.Print(err)
		}
		err = p.tg.SendMessage(chatID, message, parseMode, replyMessageId)
		if err != nil {
			log.Print(err)
		}
		mthd = doNothingMethod

	case isCommand(cmd, GetChatIDCmd):
		message = strconv.Itoa(chat.ID)
		mthd = sendMessageMethod
		replyMessageId = messageID

	case isCommand(cmd, GetMyStatsCmd):
		replyMessageId = messageID
		message = fmt.Sprintf(msgUserStats, userStats.MessageCount, userStats.DickPlusCount,
			userStats.DickMinusCount, userStats.YesCount, userStats.NoCount, userStats.DuelsCount,
			userStats.DuelsWinCount, userStats.DuelsLoseCount, userStats.KillCount, userStats.DieCount)
		mthd = sendMessageMethod
	case isCommand(cmd, GetChatStatsCmd):
		userStats, err = p.chatStats(chat.ID)
		if err != nil {
			return "", UnsupportedMethod, parseMode, replyMessageId, e.Wrap("can't get chat stats: ", err)
		}
		message = fmt.Sprintf(msgUserStats, userStats.MessageCount, userStats.DickPlusCount,
			userStats.DickMinusCount, userStats.YesCount, userStats.NoCount, userStats.DuelsCount,
			userStats.DuelsWinCount, userStats.DuelsLoseCount, userStats.KillCount, userStats.DieCount)
		replyMessageId = messageID
		mthd = sendMessageMethod
	}
	return message, mthd, parseMode, replyMessageId, nil
}