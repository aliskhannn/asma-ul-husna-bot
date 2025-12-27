package service

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
	"github.com/aliskhannn/asma-ul-husna-bot/internal/infra/postgres/repository"
)

var ErrNoQuestionsAvailable = errors.New("no questions available for quiz")

// questionTypes contains possible types of quiz questions.
var questionTypes = []entities.QuestionType{
	entities.QuestionTypeTranslation,
	entities.QuestionTypeTranslation,
	entities.QuestionTypeMeaning,
	entities.QuestionTypeArabic,
}

// QuizService provides business logic for quiz generation and management.
type QuizService struct {
	tr               Transactor
	nameRepo         NameRepository
	progressRepo     ProgressRepository
	quizRepo         QuizRepository
	settingsRepo     SettingsRepository
	dailyNameRepo    DailyNameRepository
	questionSelector *QuestionSelector
	optionGenerator  *OptionGenerator
	answerValidator  *AnswerValidator
	logger           *zap.Logger
}

// NewQuizService creates a new QuizService with the provided repositories.
func NewQuizService(
	tr Transactor,
	nameRepo NameRepository,
	progressRepo ProgressRepository,
	quizRepo QuizRepository,
	settingsRepo SettingsRepository,
	dailyNameRepo DailyNameRepository,
	logger *zap.Logger,
) *QuizService {
	return &QuizService{
		tr:            tr,
		nameRepo:      nameRepo,
		quizRepo:      quizRepo,
		settingsRepo:  settingsRepo,
		dailyNameRepo: dailyNameRepo,

		questionSelector: NewQuestionSelector(progressRepo, settingsRepo, dailyNameRepo),
		answerValidator:  NewAnswerValidator(),
		logger:           logger,
	}
}

// AnswerResult contains the result of submitting an answer.
type AnswerResult struct {
	IsCorrect         bool
	CorrectAnswer     string
	NameNumber        int
	IsSessionComplete bool
	Score             int
	Total             int
	SessionID         int64
}

// StartQuizSession creates a new quiz session with questions.
func (s *QuizService) StartQuizSession(
	ctx context.Context, userID int64, totalQuestions int,
) (*entities.QuizSession, []entities.Name, error) {
	// Abandon any old active sessions
	if err := s.quizRepo.AbandonOldSessions(ctx, userID); err != nil {
		return nil, nil, fmt.Errorf("abandon old sessions: %w", err)
	}

	// Get user settings
	settings, err := s.settingsRepo.GetByUserID(ctx, userID)
	if err != nil {
		if !errors.Is(err, repository.ErrSettingsNotFound) {
			return nil, nil, fmt.Errorf("get settings: %w", err)
		}
		// Use defaults if settings not found
		settings = entities.NewUserSettings(userID)
	}

	// Select questions using smart algorithm
	nameNumbers, err := s.questionSelector.SelectQuestions(ctx, userID, totalQuestions, settings.QuizMode)
	if err != nil {
		return nil, nil, fmt.Errorf("select questions: %w", err)
	}

	if len(nameNumbers) == 0 {
		return nil, nil, ErrNoQuestionsAvailable
	}

	// Fetch name details
	names, err := s.nameRepo.GetByNumbers(nameNumbers)
	if err != nil {
		return nil, nil, fmt.Errorf("get names: %w", err)
	}

	if len(names) == 0 {
		return nil, nil, ErrNoQuestionsAvailable
	}

	// Get all names for option generation
	allNames, err := s.nameRepo.GetAll()
	if err != nil {
		return nil, nil, fmt.Errorf("get all names: %w", err)
	}

	// Initialize option generator
	optionGenerator := NewOptionGenerator(allNames)

	// Create session
	session := &entities.QuizSession{
		UserID:             userID,
		CurrentQuestionNum: 1,
		TotalQuestions:     len(names),
		QuizMode:           settings.QuizMode,
		SessionStatus:      "active",
		StartedAt:          time.Now(),
		Version:            0,
	}

	err = s.tr.WithinTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		quizRepoTx := repository.NewQuizRepository(tx)

		sessionID, err := quizRepoTx.Create(ctx, session)
		if err != nil {
			return fmt.Errorf("create session: %w", err)
		}
		session.ID = sessionID

		// Create questions
		for i, name := range names {
			questionType := s.randomQuestionType()

			// Generate 4 options including the correct answer
			options, correctIndex := optionGenerator.GenerateOptions(&name, questionType)

			correctAnswer := s.getCorrectAnswerByType(&name, questionType)

			question := &entities.QuizQuestion{
				SessionID:     sessionID,
				QuestionOrder: i + 1,
				NameNumber:    name.Number,
				QuestionType:  string(questionType),
				CorrectAnswer: correctAnswer,
				Options:       options,
				CorrectIndex:  correctIndex,
				CreatedAt:     time.Now(),
			}

			_, err := quizRepoTx.CreateQuestion(ctx, question)
			if err != nil {
				return fmt.Errorf("create question %d: %w", i+1, err)
			}
		}

		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return session, names, nil
}

// SubmitAnswer processes a user's answer with race condition protection.
func (s *QuizService) SubmitAnswer(
	ctx context.Context,
	sessionID int64,
	userID int64,
	selectedOption string, // The button callback data (e.g., "opt_1", "opt_2", etc.)
) (*AnswerResult, error) {
	// Parse selected index
	selectedIndex, err := strconv.Atoi(selectedOption)
	if err != nil {
		return nil, fmt.Errorf("invalid option index: %w", err)
	}

	var res *AnswerResult

	err = s.tr.WithinTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		quizRepoTx := repository.NewQuizRepository(tx)
		progressRepoTx := repository.NewProgressRepository(tx)

		// Get session with lock
		session, err := quizRepoTx.GetSessionForUpdate(ctx, sessionID, userID)
		if err != nil {
			return fmt.Errorf("get session: %w", err)
		}

		// Get current question
		currentQuestion, err := quizRepoTx.GetQuestionByOrder(ctx, session.ID, session.CurrentQuestionNum)
		if err != nil {
			return fmt.Errorf("get current question: %w", err)
		}

		// Validate answer by comparing indices
		isCorrect := selectedIndex == currentQuestion.CorrectIndex

		// Get actual answer text for logging
		userAnswerText := "invalid"
		if selectedIndex >= 0 && selectedIndex < len(currentQuestion.Options) {
			userAnswerText = currentQuestion.Options[selectedIndex]
		}

		// Save answer
		answer := &entities.QuizAnswer{
			UserID:        userID,
			SessionID:     sessionID,
			QuestionID:    currentQuestion.ID,
			NameNumber:    currentQuestion.NameNumber,
			UserAnswer:    userAnswerText,
			CorrectAnswer: currentQuestion.CorrectAnswer,
			QuestionType:  currentQuestion.QuestionType,
			IsCorrect:     isCorrect,
			AnsweredAt:    time.Now(),
		}

		if err := quizRepoTx.SaveAnswer(ctx, answer); err != nil {
			return fmt.Errorf("save answer: %w", err)
		}

		// Update progress (SRS)
		quality := entities.DetermineQuality(isCorrect, true)
		if err := s.updateProgressTx(ctx, progressRepoTx, userID, currentQuestion.NameNumber, quality); err != nil {
			return fmt.Errorf("update progress: %w", err)
		}

		// Update session
		if isCorrect {
			session.IncrementCorrectAnswers()
		}
		session.IncrementQuestion()

		// Check if session is complete
		if session.ShouldComplete() {
			session.MarkCompleted(time.Now())
		}

		// Update session with optimistic locking
		if err := quizRepoTx.UpdateSession(ctx, session); err != nil {
			if errors.Is(err, repository.ErrOptimisticLock) {
				return errors.New("answer already submitted, please wait")
			}
			return fmt.Errorf("update session: %w", err)
		}

		res = &AnswerResult{
			IsCorrect:         isCorrect,
			CorrectAnswer:     currentQuestion.CorrectAnswer,
			NameNumber:        currentQuestion.NameNumber,
			IsSessionComplete: session.IsCompleted(),
			Score:             session.CorrectAnswers,
			Total:             session.TotalQuestions,
			SessionID:         sessionID,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *QuizService) IsFirstQuiz(ctx context.Context, userID int64) (bool, error) {
	return s.quizRepo.IsFirstQuiz(ctx, userID)
}

// GetActiveSession retrieves the active quiz session for a user.
func (s *QuizService) GetActiveSession(ctx context.Context, userID int64) (*entities.QuizSession, error) {
	session, err := s.quizRepo.GetActiveSessionByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrSessionNotFound) {
			return nil, nil // No active session
		}
		return nil, fmt.Errorf("get active session: %w", err)
	}

	return session, nil
}

// GetCurrentQuestion retrieves the current question for an active session.
func (s *QuizService) GetCurrentQuestion(ctx context.Context, sessionID int64, questionNum int) (*entities.QuizQuestion, *entities.Name, error) {
	question, err := s.quizRepo.GetQuestionByOrder(ctx, sessionID, questionNum)
	if err != nil {
		return nil, nil, fmt.Errorf("get question: %w", err)
	}

	name, err := s.nameRepo.GetByNumber(question.NameNumber)
	if err != nil {
		return nil, nil, fmt.Errorf("get name: %w", err)
	}

	return question, name, nil
}

// randomQuestionType selects a random question type.
func (s *QuizService) randomQuestionType() entities.QuestionType {
	return questionTypes[rand.Intn(len(questionTypes))]
}

// getCorrectAnswerByType returns the correct answer based on question type.
func (s *QuizService) getCorrectAnswerByType(name *entities.Name, questionType entities.QuestionType) string {
	switch questionType {
	case entities.QuestionTypeTranslation:
		return name.ArabicName
	case entities.QuestionTypeTransliteration:
		return name.Translation
	case entities.QuestionTypeMeaning:
		return name.Transliteration
	case entities.QuestionTypeArabic:
		return name.Translation
	default:
		return name.Translation
	}
}

// validateAnswer checks if the selected option matches the correct answer.
func (s *QuizService) validateAnswer(selectedOption string, name *entities.Name, questionType string) bool {
	if s.answerValidator == nil {
		s.answerValidator = NewAnswerValidator()
	}

	var correctAnswer string

	switch questionType {
	case string(entities.QuestionTypeTranslation):
		correctAnswer = name.Translation
	case string(entities.QuestionTypeTransliteration):
		correctAnswer = name.Transliteration
	case string(entities.QuestionTypeMeaning):
		correctAnswer = name.Meaning
	default:
		return false
	}

	return s.answerValidator.Validate(selectedOption, correctAnswer)
}

// updateProgress updates user progress with SRS algorithm.
func (s *QuizService) updateProgressTx(
	ctx context.Context,
	progressRepo ProgressRepository,
	userID int64,
	nameNumber int,
	quality entities.AnswerQuality,
) error {
	// Get existing progress
	progress, err := progressRepo.Get(ctx, userID, nameNumber)
	if err != nil {
		if !errors.Is(err, repository.ErrProgressNotFound) {
			return err
		}
		// Create new progress
		progress = entities.NewUserProgress(userID, nameNumber)
	}

	// Update SRS
	now := time.Now()
	progress.UpdateSRS(quality, now)

	return progressRepo.Upsert(ctx, progress)
}
