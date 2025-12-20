package service

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
	"github.com/aliskhannn/asma-ul-husna-bot/internal/repository"
)

type NameRepo interface {
	GetByNumber(_ context.Context, number int) (*entities.Name, error)
	GetAll(_ context.Context) ([]*entities.Name, error)
}

type ProgressRepo interface {
	GetByUserID(ctx context.Context, userID int64) ([]*entities.UserProgress, error)
	RecordReview(ctx context.Context, userID int64, nameNumber int, isCorrect bool, reviewedAt time.Time) error
	GetNamesToReview(ctx context.Context, userID int64, limit int) ([]int, error)
	GetNewNames(ctx context.Context, userID int64, limit int) ([]int, error)
	GetNamesDueForReview(ctx context.Context, userID int64, limit int) ([]int, error)
	Get(ctx context.Context, userID int64, nameNumber int) (*entities.UserProgress, error)
	Upsert(ctx context.Context, progress *entities.UserProgress) error
}

type QuizRepo interface {
	Create(ctx context.Context, s *entities.QuizSession) (int64, error)
	GetByID(ctx context.Context, id int64) (*entities.QuizSession, error)
	Update(ctx context.Context, s *entities.QuizSession) error
	SaveAnswer(ctx context.Context, a *entities.QuizAnswer) error
}

var questionTypes = []string{"translation", "transliteration", "meaning", "arabic"}

const (
	maxDueReviews = 20
	maxNewNames   = 5
	maxReviews    = 30
	maxNew        = 10
)

var ErrNoQuestionsAvailable = errors.New("no questions available")

type SettingsRepo interface {
	GetByUserID(ctx context.Context, userID int64) (*entities.UserSettings, error)
}

type QuizService struct {
	nameRepo     NameRepo
	progressRepo ProgressRepo
	quizRepo     QuizRepo
	settingsRepo SettingsRepo
}

func NewQuizService(
	nameRepo NameRepo,
	progressRepo ProgressRepo,
	quizRepo QuizRepo,
	settingsRepo SettingsRepo,
) *QuizService {
	return &QuizService{
		nameRepo:     nameRepo,
		progressRepo: progressRepo,
		quizRepo:     quizRepo,
		settingsRepo: settingsRepo,
	}
}

func (s *QuizService) GenerateQuiz(
	ctx context.Context, userID int64, mode string,
) (*entities.QuizSession, []entities.Question, error) {
	settings, err := s.settingsRepo.GetByUserID(ctx, userID)
	if err != nil && !errors.Is(err, repository.ErrSettingsNotFound) {
		return nil, nil, err
	}

	if settings == nil {
		settings = entities.NewUserSettings(userID)
	}

	var nameNumbers []int

	switch mode {
	case "daily", "mixed":
		// 1. First, we take repetitions (priority!).
		nameNumbers, err = s.getDailyQuizNames(ctx, userID)
		if err != nil {
			return nil, nil, err
		}

	case "review":
		// Only repetitions.
		nameNumbers, err = s.getReviewQuizNames(ctx, userID, settings)
		if err != nil {
			return nil, nil, err
		}

	case "new":
		// New only.
		nameNumbers, err = s.getNewQuizNames(ctx, userID)
		if err != nil {
			return nil, nil, err
		}

	default:
		return nil, nil, fmt.Errorf("unknown quiz mode: %s", mode)
	}

	if len(nameNumbers) == 0 {
		return nil, nil, ErrNoQuestionsAvailable
	}

	session := entities.NewQuizSession(userID, len(nameNumbers), mode)
	id, err := s.quizRepo.Create(ctx, session)
	if err != nil {
		return nil, nil, err
	}
	session.ID = id

	questions, err := s.generateQuestions(ctx, nameNumbers, settings)
	if err != nil {
		return nil, nil, err
	}
	if len(questions) == 0 {
		return nil, nil, ErrNoQuestionsAvailable
	}

	return session, questions, nil
}

func (s *QuizService) getDailyQuizNames(ctx context.Context, userID int64) ([]int, error) {
	dueNames, err := s.progressRepo.GetNamesDueForReview(ctx, userID, maxDueReviews)
	if err != nil {
		return nil, fmt.Errorf("get names due for review: %w", err)
	}

	newNamesCount := 0
	if len(dueNames) < 5 {
		newNamesCount = maxNewNames - len(dueNames)
		if newNamesCount > maxNewNames {
			newNamesCount = maxNewNames
		}
	}

	var nameNumbers []int
	nameNumbers = append(nameNumbers, dueNames...)

	if newNamesCount > 0 {
		newNames, err := s.progressRepo.GetNewNames(ctx, userID, newNamesCount)
		if err != nil {
			return nil, fmt.Errorf("get new names: %w", err)
		}
		nameNumbers = append(nameNumbers, newNames...)
	}

	return nameNumbers, nil
}

func (s *QuizService) getReviewQuizNames(ctx context.Context, userID int64, settings *entities.UserSettings) ([]int, error) {
	reviewLimit := settings.MaxReviewsPerDay
	if reviewLimit == 0 || reviewLimit > maxReviews {
		reviewLimit = maxReviews
	}

	dueNames, err := s.progressRepo.GetNamesDueForReview(ctx, userID, reviewLimit)
	if err != nil {
		return nil, fmt.Errorf("get due names: %w", err)
	}

	return dueNames, nil
}

func (s *QuizService) getNewQuizNames(ctx context.Context, userID int64) ([]int, error) {
	newNames, err := s.progressRepo.GetNewNames(ctx, userID, maxNew)
	if err != nil {
		return nil, fmt.Errorf("get new names: %w", err)
	}

	return newNames, nil
}

func (s *QuizService) GetSession(ctx context.Context, sessionID int64) (*entities.QuizSession, error) {
	return s.quizRepo.GetByID(ctx, sessionID)
}

func (s *QuizService) CheckAndSaveAnswer(
	ctx context.Context,
	userID int64,
	session *entities.QuizSession,
	q *entities.Question,
	selectedIndex int,
) (*entities.QuizAnswer, error) {
	if selectedIndex < 0 || selectedIndex >= len(q.Options) {
		return nil, fmt.Errorf("invalid selected index")
	}

	userAnswer := q.Options[selectedIndex]
	correctAnswer := q.CorrectAnswer

	qa := entities.NewQuizAnswer(userID, session.ID, q.NameNumber, q.Type)
	qa.CheckAnswer(userAnswer, correctAnswer)

	if err := s.quizRepo.SaveAnswer(ctx, qa); err != nil {
		return nil, err
	}

	reviewedAt := time.Now()

	if err := s.progressRepo.RecordReview(ctx, userID, q.NameNumber, qa.IsCorrect, reviewedAt); err != nil {
		return nil, err
	}

	if qa.IsCorrect {
		session.CorrectAnswers++
	}
	session.CurrentQuestionNum++
	if session.CurrentQuestionNum > session.TotalQuestions {
		session.SessionStatus = "completed"
		now := time.Now()
		session.CompletedAt = &now
	}

	if err := s.quizRepo.Update(ctx, session); err != nil {
		return nil, err
	}

	return qa, nil
}

func (s *QuizService) generateQuestions(
	ctx context.Context, nameNumbers []int, settings *entities.UserSettings,
) ([]entities.Question, error) {
	questions := make([]entities.Question, 0, len(nameNumbers))

	allNames, err := s.nameRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	for _, num := range nameNumbers {
		name, err := s.nameRepo.GetByNumber(ctx, num)
		if err != nil {
			return nil, err
		}

		qType := s.randomQuestionType(settings)

		var q entities.Question
		switch qType {
		case "translation":
			q = s.generateTranslationQuestion(name, allNames)
		case "transliteration":
			q = s.generateTransliterationQuestion(name, allNames)
		case "meaning":
			q = s.generateMeaningQuestion(name, allNames)
		case "arabic":
			q = s.generateArabicQuestion(name, allNames)
		default:
			continue
		}

		questions = append(questions, q)
	}

	rand.Shuffle(len(questions), func(i, j int) {
		questions[i], questions[j] = questions[j], questions[i]
	})

	return questions, nil
}

func (s *QuizService) randomQuestionType(_ *entities.UserSettings) string {
	return questionTypes[rand.Intn(len(questionTypes))]
}

func (s *QuizService) getRandomDistractors(
	all []*entities.Name,
	targetNumber int,
	count int,
) []*entities.Name {
	candidates := make([]*entities.Name, 0, len(all))
	for _, n := range all {
		if n.Number != targetNumber {
			candidates = append(candidates, n)
		}
	}

	if len(candidates) <= count {
		return candidates
	}

	rand.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})

	return candidates[:count]
}

func buildOptionsWithCorrect(correct string, distractors []string) ([]string, int) {
	options := make([]string, 0, 1+len(distractors))
	options = append(options, correct)
	options = append(options, distractors...)

	rand.Shuffle(len(options), func(i, j int) {
		options[i], options[j] = options[j], options[i]
	})

	correctIndex := 0
	for i, opt := range options {
		if opt == correct {
			correctIndex = i
			break
		}
	}

	return options, correctIndex
}

func (s *QuizService) generateTranslationQuestion(
	target *entities.Name,
	allNames []*entities.Name,
) entities.Question {
	distractorNames := s.getRandomDistractors(allNames, target.Number, 3)

	distractorOptions := make([]string, 0, len(distractorNames))
	for _, d := range distractorNames {
		distractorOptions = append(distractorOptions, d.ArabicName)
	}

	options, correctIndex := buildOptionsWithCorrect(target.ArabicName, distractorOptions)

	return entities.Question{
		NameNumber:    target.Number,
		Type:          "translation",
		Question:      fmt.Sprintf("Какое арабское имя означает *%s*?", target.Translation),
		Options:       options,
		CorrectIndex:  correctIndex,
		CorrectAnswer: target.ArabicName,
	}
}

func (s *QuizService) generateTransliterationQuestion(
	target *entities.Name,
	allNames []*entities.Name,
) entities.Question {
	distractorNames := s.getRandomDistractors(allNames, target.Number, 3)

	distractorOptions := make([]string, 0, len(distractorNames))
	for _, d := range distractorNames {
		distractorOptions = append(distractorOptions, d.Translation)
	}

	options, correctIndex := buildOptionsWithCorrect(target.Translation, distractorOptions)

	return entities.Question{
		NameNumber:    target.Number,
		Type:          "transliteration",
		Question:      fmt.Sprintf("Что означает имя *%s*?", target.Transliteration),
		Options:       options,
		CorrectIndex:  correctIndex,
		CorrectAnswer: target.Translation,
	}
}

func (s *QuizService) generateMeaningQuestion(
	target *entities.Name,
	allNames []*entities.Name,
) entities.Question {
	distractorNames := s.getRandomDistractors(allNames, target.Number, 3)

	distractorOptions := make([]string, 0, len(distractorNames))
	for _, d := range distractorNames {
		distractorOptions = append(distractorOptions, d.Transliteration)
	}

	options, correctIndex := buildOptionsWithCorrect(target.Transliteration, distractorOptions)

	text := target.Meaning
	if text == "" {
		text = target.Translation
	}

	return entities.Question{
		NameNumber:    target.Number,
		Type:          "meaning",
		Question:      fmt.Sprintf("Какое из имён соответствует значению: *%s*?", text),
		Options:       options,
		CorrectIndex:  correctIndex,
		CorrectAnswer: target.Transliteration,
	}
}

func (s *QuizService) generateArabicQuestion(
	target *entities.Name,
	allNames []*entities.Name,
) entities.Question {
	distractorNames := s.getRandomDistractors(allNames, target.Number, 3)

	distractorOptions := make([]string, 0, len(distractorNames))
	for _, d := range distractorNames {
		distractorOptions = append(distractorOptions, d.Translation)
	}

	options, correctIndex := buildOptionsWithCorrect(target.Translation, distractorOptions)

	return entities.Question{
		NameNumber:    target.Number,
		Type:          "arabic",
		Question:      fmt.Sprintf("Что означает арабское имя *%s*?", target.ArabicName),
		Options:       options,
		CorrectIndex:  correctIndex,
		CorrectAnswer: target.Translation,
	}
}
