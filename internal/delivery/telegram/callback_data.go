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
)

// Settings sub-actions.
const (
	settingsMenu                  = "menu"
	settingsNamesPerDay           = "names_per_day"
	settingsQuizLength            = "quiz_length"
	settingsQuizMode              = "quiz_mode"
	settingsToggleTransliteration = "toggle_transliteration"
	settingsToggleAudio           = "toggle_audio"
)

// Quiz sub-actions.
const (
	quizStart  = "start"
	quizAnswer = "answer"
)

// callbackData represents structured callback data.
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
