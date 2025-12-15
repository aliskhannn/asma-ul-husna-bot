package telegram

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
)

const namesPerPage = 5

func (h *Handler) handleAllCommand(ctx context.Context, chatID int64) {
	msg := newHTMLMessage(chatID, "")

	names := h.getAllNames(ctx)
	if names == nil {
		msg.Text = msgNameUnavailable
		h.send(msg)
		return
	}

	page := 0
	text, totalPages := buildNamesPage(names, page)

	prevData := fmt.Sprintf("name:%d", page-1)
	nextData := fmt.Sprintf("name:%d", page+1)

	msg.Text = text
	kb := buildNameKeyboard(page, totalPages, prevData, nextData)
	if kb != nil {
		msg.ReplyMarkup = *kb
	}

	h.send(msg)
}

func (h *Handler) handleRangeCommand(ctx context.Context, chatID int64, argsStr string) {
	msg := newHTMLMessage(chatID, "")

	args := strings.Fields(argsStr)
	if len(args) != 2 {
		msg.Text = msgUseRange
		h.send(msg)
		return
	}

	from, errFrom := strconv.Atoi(args[0])
	to, errTo := strconv.Atoi(args[1])
	if errFrom != nil || errTo != nil || from < 1 || to > 99 || from > to {
		msg.Text = msgInvalidRange
		h.send(msg)
		return
	}

	names := h.getAllNames(ctx)
	if names == nil {
		msg.Text = msgNameUnavailable
		h.send(msg)
		return
	}

	pages := buildRangePages(names, from, to)
	if len(pages) == 0 {
		msg.Text = msgNameUnavailable
		h.send(msg)
		return
	}

	page := 0
	totalPages := len(pages)

	prevData := fmt.Sprintf("range:%d:%d:%d", page-1, from, to)
	nextData := fmt.Sprintf("range:%d:%d:%d", page+1, from, to)

	msg = newHTMLMessage(chatID, pages[page])
	kb := buildNameKeyboard(page, totalPages, prevData, nextData)
	if kb != nil {
		msg.ReplyMarkup = *kb
	}

	h.send(msg)
}

func buildNameKeyboard(page, totalPages int, prevData, nextData string) *tgbotapi.InlineKeyboardMarkup {
	if totalPages <= 1 {
		return nil
	}

	var row []tgbotapi.InlineKeyboardButton

	if page > 0 {
		row = append(row, tgbotapi.NewInlineKeyboardButtonData("◀️️ Назад", prevData))
	}
	if page < totalPages-1 {
		row = append(row, tgbotapi.NewInlineKeyboardButtonData("Вперёд ▶️", nextData))
	}

	kb := tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{row},
	}
	return &kb
}

func buildNamesPage(names []entities.Name, page int) (text string, totalPages int) {
	totalPages = (len(names) + namesPerPage - 1) / namesPerPage
	if totalPages == 0 {
		return "", 0
	}

	pageNames := paginateNames(names, page, namesPerPage)

	var b strings.Builder
	for i, name := range pageNames {
		if i > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(processName(name))
	}

	return b.String(), totalPages
}

func buildRangePages(names []entities.Name, from, to int) (pages []string) {
	if from < 1 {
		from = 1
	}
	if to > len(names) {
		to = len(names)
	}
	if from > to {
		return nil
	}

	fromIdx := from - 1
	toIdx := to

	for start := fromIdx; start < toIdx; start += namesPerPage {
		end := start + namesPerPage
		if end > toIdx {
			end = toIdx
		}

		chunk := names[start:end]

		var b strings.Builder
		for i, name := range chunk {
			if i > 0 {
				b.WriteString("\n\n")
			}
			b.WriteString(processName(name))
		}

		pages = append(pages, b.String())
	}

	return pages
}

func paginateNames(names []entities.Name, page, namesPerPage int) []entities.Name {
	start := page * namesPerPage
	end := start + namesPerPage

	if start >= len(names) {
		return nil
	}
	if end > len(names) {
		end = len(names)
	}

	return names[start:end]
}

func (h *Handler) getAllNames(ctx context.Context) []entities.Name {
	names, err := h.nameService.GetAllNames(ctx)
	if err != nil {
		log.Printf("failed to get all names: %v", err)
		return nil
	}
	if len(names) == 0 {
		log.Println("no names found")
		return nil
	}

	return names
}
