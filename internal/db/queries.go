package db

import (
	"context"

	"codeeval/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	Pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{Pool: pool}
}

func (s *Store) ListProblems(ctx context.Context) ([]models.Problem, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, slug, title, description_md, difficulty, time_limit_ms, memory_limit_kb, created_at, function_signature
		FROM problems ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Problem
	for rows.Next() {
		var p models.Problem
		if err := rows.Scan(&p.ID, &p.Slug, &p.Title, &p.DescriptionMD, &p.Difficulty, &p.TimeLimitMS, &p.MemoryLimitKB, &p.CreatedAt, &p.FunctionSignature); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) GetProblemBySlug(ctx context.Context, slug string) (*models.Problem, error) {
	var p models.Problem
	err := s.Pool.QueryRow(ctx, `
		SELECT id, slug, title, description_md, difficulty, time_limit_ms, memory_limit_kb, created_at, function_signature
		FROM problems WHERE slug = $1`, slug).
		Scan(&p.ID, &p.Slug, &p.Title, &p.DescriptionMD, &p.Difficulty, &p.TimeLimitMS, &p.MemoryLimitKB, &p.CreatedAt, &p.FunctionSignature)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Store) ListSampleTestCases(ctx context.Context, problemID string) ([]models.TestCase, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, problem_id, args_json, expected_return_json, is_sample, weight, ordinal
		FROM test_cases WHERE problem_id = $1 AND is_sample = true ORDER BY ordinal`, problemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTestCases(rows)
}

func (s *Store) ListAllTestCases(ctx context.Context, problemID string) ([]models.TestCase, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, problem_id, args_json, expected_return_json, is_sample, weight, ordinal
		FROM test_cases WHERE problem_id = $1 ORDER BY ordinal`, problemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTestCases(rows)
}

func scanTestCases(rows pgx.Rows) ([]models.TestCase, error) {
	var out []models.TestCase
	for rows.Next() {
		var t models.TestCase
		if err := rows.Scan(&t.ID, &t.ProblemID, &t.ArgsJSON, &t.ExpectedReturnJSON, &t.IsSample, &t.Weight, &t.Ordinal); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) CreateSubmission(ctx context.Context, sub models.Submission) (string, error) {
	var id string
	err := s.Pool.QueryRow(ctx, `
		INSERT INTO submissions (user_id, problem_id, language, source_code, status, max_score)
		VALUES ($1, $2, $3, $4, 'pending', $5)
		RETURNING id`,
		sub.UserID, sub.ProblemID, sub.Language, sub.SourceCode, sub.MaxScore).Scan(&id)
	return id, err
}

func (s *Store) UpdateSubmissionResult(ctx context.Context, submissionID, status string, score int) error {
	_, err := s.Pool.Exec(ctx, `
		UPDATE submissions SET status = $1, score = $2, completed_at = now() WHERE id = $3`,
		status, score, submissionID)
	return err
}

func (s *Store) InsertSubmissionResult(ctx context.Context, r models.SubmissionResult) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO submission_results (submission_id, test_case_id, passed, stdout, stderr, runtime_ms, memory_kb, exec_status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		r.SubmissionID, r.TestCaseID, r.Passed, r.Stdout, r.Stderr, r.RuntimeMS, r.MemoryKB, r.ExecStatus)
	return err
}

func (s *Store) GetSubmission(ctx context.Context, id string) (*models.Submission, error) {
	var sub models.Submission
	err := s.Pool.QueryRow(ctx, `
		SELECT id, user_id, problem_id, language, source_code, status, score, max_score, submitted_at, completed_at
		FROM submissions WHERE id = $1`, id).
		Scan(&sub.ID, &sub.UserID, &sub.ProblemID, &sub.Language, &sub.SourceCode, &sub.Status, &sub.Score, &sub.MaxScore, &sub.SubmittedAt, &sub.CompletedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &sub, err
}

func (s *Store) ListSubmissionResults(ctx context.Context, submissionID string) ([]models.SubmissionResult, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, submission_id, test_case_id, passed, stdout, stderr, runtime_ms, memory_kb, exec_status
		FROM submission_results WHERE submission_id = $1`, submissionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.SubmissionResult
	for rows.Next() {
		var r models.SubmissionResult
		if err := rows.Scan(&r.ID, &r.SubmissionID, &r.TestCaseID, &r.Passed, &r.Stdout, &r.Stderr, &r.RuntimeMS, &r.MemoryKB, &r.ExecStatus); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
