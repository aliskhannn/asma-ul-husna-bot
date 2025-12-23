package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aliskhannn/asma-ul-husna-bot/internal/domain/entities"
)

var (
	ErrSessionNotFound  = errors.New("quiz session not found")
	ErrOptimisticLock   = errors.New("quiz session was modified by another process")
	ErrSessionNotActive = errors.New("quiz session is not active")
)

// QuizRepository provides access to quiz session and answer data in the database.
type QuizRepository struct {
	db *pgxpool.Pool
}

// NewQuizRepository creates a new QuizRepository with the provided database pool.
func NewQuizRepository(db *pgxpool.Pool) *QuizRepository {
	return &QuizRepository{db: db}
}

// CreateWithTx creates a new quiz session within a transaction.
func (r *QuizRepository) CreateWithTx(ctx context.Context, tx pgx.Tx, session *entities.QuizSession) (int64, error) {
	query := `
		INSERT INTO quiz_sessions (
			user_id, current_question_num, total_questions, 
			quiz_mode, session_status, started_at, version
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	var id int64
	err := tx.QueryRow(
		ctx,
		query,
		session.UserID,
		session.CurrentQuestionNum,
		session.TotalQuestions,
		session.QuizMode,
		session.SessionStatus,
		session.StartedAt,
		session.Version,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("create quiz session: %w", err)
	}

	return id, nil
}

// CreateQuestionWithTx creates a quiz question within a transaction.
func (r *QuizRepository) CreateQuestionWithTx(ctx context.Context, tx pgx.Tx, session *entities.QuizQuestion) (int64, error) {
	query := `
		INSERT INTO quiz_questions (
		    session_id, question_order, name_number, 
		    question_type, correct_answer, options, correct_index
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	var id int64
	err := tx.QueryRow(
		ctx,
		query,
		session.SessionID,
		session.QuestionOrder,
		session.NameNumber,
		session.QuestionType,
		session.CorrectAnswer,
		session.Options,
		session.CorrectIndex,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("create quiz question: %w", err)
	}

	return id, nil
}

// GetSessionForUpdateWithTx retrieves a session with row-level lock for update
func (r *QuizRepository) GetSessionForUpdateWithTx(ctx context.Context, tx pgx.Tx, sessionID, userID int64) (*entities.QuizSession, error) {
	query := `
		SELECT id, user_id, current_question_num, correct_answers, total_questions,
		       quiz_mode, session_status, started_at, completed_at, version
		FROM quiz_sessions
		WHERE id = $1 AND user_id = $2
		FOR UPDATE
	`

	var session entities.QuizSession
	err := tx.QueryRow(ctx, query, sessionID, userID).Scan(
		&session.ID,
		&session.UserID,
		&session.CurrentQuestionNum,
		&session.CorrectAnswers,
		&session.TotalQuestions,
		&session.QuizMode,
		&session.SessionStatus,
		&session.StartedAt,
		&session.CompletedAt,
		&session.Version,
	)

	if err != nil {
		return nil, fmt.Errorf("get session for update: %w", err)
	}

	if !session.IsActive() {
		return nil, ErrSessionNotActive
	}

	return &session, nil
}

// GetActiveSessionByUserID retrieves the active session for a user.
func (r *QuizRepository) GetActiveSessionByUserID(ctx context.Context, userID int64) (*entities.QuizSession, error) {
	query := `
		SELECT id, user_id, current_question_num, correct_answers, total_questions,
		       quiz_mode, session_status, started_at, completed_at, version
		FROM quiz_sessions
		WHERE user_id = $1 AND session_status = 'active'
		ORDER BY started_at DESC
		LIMIT 1
	`

	var session entities.QuizSession
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&session.ID,
		&session.UserID,
		&session.CurrentQuestionNum,
		&session.CorrectAnswers,
		&session.TotalQuestions,
		&session.QuizMode,
		&session.SessionStatus,
		&session.StartedAt,
		&session.CompletedAt,
		&session.Version,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("get quiz session: %w", err)
	}

	return &session, nil
}

// GetQuestionByOrder retrieves a question by its order in the session.
func (r *QuizRepository) GetQuestionByOrder(ctx context.Context, sessionID int64, order int) (*entities.QuizQuestion, error) {
	query := `
		SELECT id, session_id, question_order, name_number, question_type, 
		       correct_answer, options, correct_index, created_at
		FROM quiz_questions
		WHERE session_id = $1 AND question_order = $2
	`

	var q entities.QuizQuestion
	err := r.db.QueryRow(ctx, query, sessionID, order).Scan(
		&q.ID,
		&q.SessionID,
		&q.QuestionOrder,
		&q.NameNumber,
		&q.QuestionType,
		&q.CorrectAnswer,
		&q.Options,
		&q.CorrectIndex,
		&q.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("question not found")
		}
		return nil, fmt.Errorf("get question by order: %w", err)
	}

	return &q, nil
}

// SaveAnswerWithTx saves a quiz answer within a transaction.
func (r *QuizRepository) SaveAnswerWithTx(ctx context.Context, tx pgx.Tx, answer *entities.QuizAnswer) error {
	query := `
		INSERT INTO quiz_answers (user_id, session_id, question_id, name_number, user_answer, correct_answer, question_type, is_correct, answered_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := tx.Exec(
		ctx,
		query,
		answer.UserID,
		answer.SessionID,
		answer.QuestionID,
		answer.NameNumber,
		answer.UserAnswer,
		answer.CorrectAnswer,
		answer.QuestionType,
		answer.IsCorrect,
		answer.AnsweredAt,
	)

	if err != nil {
		return fmt.Errorf("save answer: %w", err)
	}

	return nil
}

// UpdateSessionWithTx updates a quiz session using optimistic locking.
func (r *QuizRepository) UpdateSessionWithTx(ctx context.Context, tx pgx.Tx, session *entities.QuizSession) error {
	query := `
		UPDATE quiz_sessions
		SET current_question_num = $1,
		    correct_answers = $2,
		    session_status = $3,
		    completed_at = $4,
		    version = version + 1
		WHERE id = $5 AND version = $6
	`

	result, err := tx.Exec(
		ctx,
		query,
		session.CurrentQuestionNum,
		session.CorrectAnswers,
		session.SessionStatus,
		session.CompletedAt,
		session.ID,
		session.Version,
	)

	if err != nil {
		return fmt.Errorf("update session: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrOptimisticLock
	}

	// Increment version locally
	session.Version++

	return nil
}

// AbandonOldSessions marks old active sessions as abandoned.
func (r *QuizRepository) AbandonOldSessions(ctx context.Context, userID int64) error {
	query := `
		UPDATE quiz_sessions
		SET session_status = 'abandoned'
		WHERE user_id = $1 AND session_status = 'active'
	`

	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("abandon old sessions: %w", err)
	}

	return nil
}
