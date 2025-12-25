# Asma ul‑Husna Bot

A Telegram bot that helps you learn the 99 Beautiful Names of Allah (Asma ul‑Husna) through name cards, quizzes, progress tracking, and reminders.

## Features

- 📖 Name cards with translation, transliteration, and audio pronunciation
- 🎯 Quizzes to reinforce learning and check retention
- 📊 Progress tracking (learned / in progress) + streaks
- 🔔 Flexible reminders (time window + interval)
- ⚙️ Learning modes:
    - **Guided**: daily plan with a configurable “names per day” limit
    - **Free**: explore and learn without being blocked by the daily quota
- 🧠 Spaced repetition mechanics to improve long‑term memorization

## How to use

Recommended learning loop:
- `/next → /today → /quiz`

Where:
- `/next` shows the next name to learn and can introduce a new one
- `/today` lists today’s names (in Guided mode)
- `/quiz` helps you consolidate and move forward

## Commands

### Learning
- `/next` — show the next name / introduce a new one
- `/today` — list today’s names
- `/quiz` — start a quiz for the current learning set
- `/random` — random name (Guided: from today’s list, Free: from all 99)

### Browse
- `1-99` — open a specific name by number (browse mode)
- `/all` — list all 99 names
- `/range N M` — list names from N to M (e.g. `/range 1 10`)

### Progress & settings
- `/progress` — show learning statistics
- `/settings` — learning mode, quiz options, reminders, names per day
- `/help` — show help and commands list
- `/reset` — reset progress (if enabled)

## Notes

- `/random` and `1-99` are intended for exploration and may not affect progress.
- Some behavior depends on the current learning mode (Guided / Free).

## License

This project is licensed under the MIT License. See `LICENSE` for details.