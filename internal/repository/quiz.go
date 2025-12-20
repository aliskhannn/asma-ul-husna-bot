package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
)

type QuizRepository struct {
	db *pgxpool.Pool
}

func NewQuizRepository(db *pgxpool.Pool) *QuizRepository {
	return &QuizRepository{db: db}
}

func (r *QuizRepository) Create(ctx context.Context, s *entities.QuizSession) (int64, error) {
	query := `
        INSERT INTO quiz_sessions (
            user_id,
            current_question_num,
            correct_answers,
            total_questions,
            quiz_mode,
            session_status,
            started_at,
            completed_at
        )
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        RETURNING id
    `

	var id int64
	err := r.db.QueryRow(ctx, query,
		s.UserID,
		s.CurrentQuestionNum,
		s.CorrectAnswers,
		s.TotalQuestions,
		s.QuizMode,
		s.SessionStatus,
		s.StartedAt,
		s.CompletedAt,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("create quiz session: %w", err)
	}

	return id, nil
}

func (r *QuizRepository) GetByID(ctx context.Context, id int64) (*entities.QuizSession, error) {
	query := `
        SELECT id, user_id, current_question_num, correct_answers,
               total_questions, quiz_mode, session_status,
               started_at, completed_at
        FROM quiz_sessions
        WHERE id = $1
    `

	var s entities.QuizSession
	err := r.db.QueryRow(ctx, query, id).Scan(
		&s.ID,
		&s.UserID,
		&s.CurrentQuestionNum,
		&s.CorrectAnswers,
		&s.TotalQuestions,
		&s.QuizMode,
		&s.SessionStatus,
		&s.StartedAt,
		&s.CompletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get quiz session by id: %w", err)
	}

	return &s, nil
}

func (r *QuizRepository) Update(ctx context.Context, s *entities.QuizSession) error {
	query := `
        UPDATE quiz_sessions
        SET current_question_num = $2,
            correct_answers      = $3,
            session_status       = $4,
            completed_at         = $5
        WHERE id = $1
    `

	_, err := r.db.Exec(ctx, query,
		s.ID,
		s.CurrentQuestionNum,
		s.CorrectAnswers,
		s.SessionStatus,
		s.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("update quiz session: %w", err)
	}

	return nil
}

func (r *QuizRepository) SaveAnswer(ctx context.Context, a *entities.QuizAnswer) error {
	query := `
        INSERT INTO quiz_answers (
            user_id,
            session_id,
            name_number,
            user_answer,
            correct_answer,
            question_type,
            is_correct,
            answered_at
        )
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        RETURNING id
    `

	var id int64
	err := r.db.QueryRow(ctx, query,
		a.UserID,
		a.SessionID,
		a.NameNumber,
		a.UserAnswer,
		a.CorrectAnswer,
		a.QuestionType,
		a.IsCorrect,
		time.Now(),
	).Scan(&id)
	if err != nil {
		return fmt.Errorf("save quiz answer: %w", err)
	}

	a.ID = id
	return nil
}
