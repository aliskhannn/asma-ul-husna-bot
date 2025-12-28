# Asma ul‑Husna Bot

A Telegram bot that helps you learn the 99 Beautiful Names of Allah (Asma ul‑Husna) using daily name cards, quizzes, progress tracking, and reminders.

## Features

- 📖 Name cards with translation, transliteration, and audio pronunciation
- 📅 **Daily plan** (`/today`) generated automatically from your “names per day” setting (includes due/review items when applicable)
- 🧠 Quizzes to reinforce learning and check retention
- 📊 Progress tracking and statistics (`/progress`)
- 🔔 Flexible reminders with interval + time window (`/settings`)
- ⚙️ Learning modes:
    - **Guided**: focus on today’s planned names; `/random` picks from today’s list
    - **Free**: explore without being limited by the daily plan; `/random` picks from all 99

## How it works

The bot keeps a daily learning plan based on your settings (especially “names per day”) and shows it in `/today`.[46]
Use quizzes to consolidate learning, and reminders to stay consistent.

Recommended loop:
- `/today → (optional: 🔊 Listen) → /quiz → /progress`

## Commands

### Learning
- `/today` — open today’s list (with pagination + audio button)
- `/quiz` — start a quiz for your current learning set (may resume an active session)
- `/random` — random name (Guided: from today; Free: from all 99)

### Browse
- `1-99` — open a specific name by number (send “10” to open name #10)
- `N M` — open a range by sending two numbers (example: `5 10`)
- `/all` — list all 99 names (paginated)

### Progress & settings
- `/progress` — show learning statistics
- `/settings` — names per day, learning mode, quiz mode, reminders
- `/help` — help and commands list
- `/reset` — reset progress and settings (with confirmation)

## Notes

- `/random`, `1-99`, and `N M` are primarily for exploration; learning behavior can depend on the current mode (Guided/Free).
- Reminders can be enabled/disabled and configured in `/settings` (interval and time window).

## License

This project is licensed under the MIT License. See `LICENSE` for details.