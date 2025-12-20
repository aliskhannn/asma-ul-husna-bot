package telegram

import (
	"strconv"
	"strings"
)

// Callback action constants.
const (
	actionName     = "name"
	actionRange    = "range"
	actionSettings = "settings"
	actionQuiz     = "quiz"
	actionProgress = "progress"
	actionReminder = "reminder"
)

// Settings sub-actions.
const (
	settingsMenu        = "menu"
	settingsNamesPerDay = "names_per_day"
	settingsQuizMode    = "quiz_mode"
	settingsReminders   = "reminders"
)

// Reminder sub-actions.
const (
	reminderToggle    = "toggle"
	reminderSetTime   = "set_time"
	reminderStartQuiz = "start_quiz"
	reminderSnooze    = "snooze"
	reminderDisable   = "disable"
)

// Quiz sub-actions.
const (
	quizStart = "start"
)

// callbackData represents structured callback data.
type callbackData struct {
	Action string
	Params []string
	Raw    string
}

// Encode creates callback string.
func (cd callbackData) encode() string {
	if len(cd.Params) == 0 {
		return cd.Action
	}
	return cd.Action + ":" + strings.Join(cd.Params, ":")
}

// decodeCallback parses callback data string.
func decodeCallback(data string) callbackData {
	parts := strings.Split(data, ":")
	if len(parts) == 0 {
		return callbackData{Raw: data}
	}

	return callbackData{
		Action: parts[0],
		Params: parts[1:],
		Raw:    data,
	}
}

// Callback builders.

func buildNameCallback(page int) string {
	return callbackData{
		Action: actionName,
		Params: []string{strconv.Itoa(page)},
	}.encode()
}

func buildRangeCallback(page, from, to int) string {
	return callbackData{
		Action: actionRange,
		Params: []string{
			strconv.Itoa(page),
			strconv.Itoa(from),
			strconv.Itoa(to),
		},
	}.encode()
}

func buildSettingsCallback(subAction string, value ...string) string {
	params := []string{subAction}
	params = append(params, value...)
	return callbackData{
		Action: actionSettings,
		Params: params,
	}.encode()
}

func buildQuizAnswerCallback(sessionID int64, questionNum, answerIndex int) string {
	return callbackData{
		Action: actionQuiz,
		Params: []string{
			strconv.FormatInt(sessionID, 10),
			strconv.Itoa(questionNum),
			strconv.Itoa(answerIndex),
		},
	}.encode()
}

func buildQuizStartCallback() string {
	return callbackData{
		Action: actionQuiz,
		Params: []string{quizStart},
	}.encode()
}

func buildProgressCallback() string {
	return actionProgress
}

func buildReminderToggleCallback() string {
	return buildSettingsCallback(settingsReminders, reminderToggle)
}

func buildReminderSetTimeCallback(hour string) string {
	return buildSettingsCallback(settingsReminders, reminderSetTime, hour)
}

func buildReminderStartQuizCallback() string {
	return callbackData{
		Action: actionReminder,
		Params: []string{reminderStartQuiz},
	}.encode()
}

func buildReminderSnoozeCallback() string {
	return callbackData{
		Action: actionReminder,
		Params: []string{reminderSnooze},
	}.encode()
}

func buildReminderDisableCallback() string {
	return callbackData{
		Action: actionReminder,
		Params: []string{reminderDisable},
	}.encode()
}
