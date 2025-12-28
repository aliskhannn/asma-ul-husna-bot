package telegram

import (
	"strconv"
	"strings"
)

// Callback action constants.
const (
	actionName       = "name"
	actionRange      = "range"
	actionSettings   = "settings"
	actionQuiz       = "quiz"
	actionProgress   = "progress"
	actionReminder   = "reminder"
	actionOnboarding = "onboarding"
	actionToday      = "today"
	actionReset      = "reset"
)

// Settings sub-actions.
const (
	settingsMenu         = "menu"
	settingsLearningMode = "learning_mode"
	settingsNamesPerDay  = "names_per_day"
	settingsQuizMode     = "quiz_mode"
	settingsReminders    = "reminders"
)

// Reminder sub-actions.
const (
	reminderToggle    = "toggle"
	reminderStartQuiz = "start_quiz"
	reminderSnooze    = "snooze"
	reminderDisable   = "disable"
)

// Quiz sub-actions.
const (
	quizStart = "start"
)

// Onboarding sub-actions.
const (
	onboardingStep      = "step"
	onboardingNames     = "names"
	onboardingMode      = "mode"
	onboardingReminders = "reminders"
	onboardingCmd       = "cmd"
)

const (
	todayPage  = "page"
	todayAudio = "audio"
)

const (
	resetConfirm = "confirm"
	resetCancel  = "cancel"
)

// callbackData represents structured callback data.
type callbackData struct {
	Action string
	Params []string
	Raw    string
}

// encode creates callback string.
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

func buildTodayPageCallback(page int) string {
	return callbackData{
		Action: actionToday,
		Params: []string{todayPage, strconv.Itoa(page)},
	}.encode()
}

func buildTodayAudioCallback(nameNumber int) string {
	return callbackData{
		Action: actionToday,
		Params: []string{todayAudio, strconv.Itoa(nameNumber)},
	}.encode()
}

// buildNameCallback builds callback data for opening a "name" page.
func buildNameCallback(page int) string {
	return callbackData{
		Action: actionName,
		Params: []string{strconv.Itoa(page)},
	}.encode()
}

// buildRangeCallback builds callback data for opening a "range" page.
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

// buildSettingsCallback builds callback data for settings-related actions.
func buildSettingsCallback(subAction string, value ...string) string {
	params := []string{subAction}
	params = append(params, value...)
	return callbackData{
		Action: actionSettings,
		Params: params,
	}.encode()
}

// buildQuizAnswerCallback builds callback data for answering a quiz question.
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

// buildQuizStartCallback builds callback data for starting a quiz session.
func buildQuizStartCallback() string {
	return callbackData{
		Action: actionQuiz,
		Params: []string{quizStart},
	}.encode()
}

// buildProgressCallback builds callback data for opening the progress view.
func buildProgressCallback() string {
	return actionProgress
}

// buildReminderToggleCallback builds callback data for toggling reminders.
func buildReminderToggleCallback() string {
	return buildSettingsCallback(settingsReminders, reminderToggle)
}

// buildReminderStartQuizCallback builds callback data for starting a quiz from a reminder message.
func buildReminderStartQuizCallback() string {
	return callbackData{
		Action: actionReminder,
		Params: []string{reminderStartQuiz},
	}.encode()
}

// buildReminderSnoozeCallback builds callback data for snoozing reminders.
func buildReminderSnoozeCallback() string {
	return callbackData{
		Action: actionReminder,
		Params: []string{reminderSnooze},
	}.encode()
}

// buildReminderDisableCallback builds callback data for disabling reminders.
func buildReminderDisableCallback() string {
	return callbackData{
		Action: actionReminder,
		Params: []string{reminderDisable},
	}.encode()
}

func buildOnboardingStepCallback(step int) string {
	return callbackData{
		Action: actionOnboarding,
		Params: []string{onboardingStep, strconv.Itoa(step)},
	}.encode()
}

func buildOnboardingNamesPerDayCallback(n int) string {
	return callbackData{
		Action: actionOnboarding,
		Params: []string{onboardingNames, strconv.Itoa(n)},
	}.encode()
}

func buildOnboardingModeCallback(mode string) string {
	return callbackData{
		Action: actionOnboarding,
		Params: []string{onboardingMode, mode}, // guided/free
	}.encode()
}

func buildOnboardingRemindersCallback(choice string) string {
	return callbackData{
		Action: actionOnboarding,
		Params: []string{onboardingReminders, choice}, // yes/no
	}.encode()
}

func buildOnboardingCmdCallback(cmd string) string {
	return callbackData{
		Action: actionOnboarding,
		Params: []string{onboardingCmd, cmd}, // next/all
	}.encode()
}

func buildResetConfirmCallback() string {
	return callbackData{Action: actionReset, Params: []string{resetConfirm}}.encode()
}

func buildResetCancelCallback() string {
	return callbackData{Action: actionReset, Params: []string{resetCancel}}.encode()
}
