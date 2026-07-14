package models

import "time"

type User struct {
	ID           string
	Email        string
	Name         string
	PasswordHash string
	CreatedAt    time.Time
}

type Problem struct {
	ID                string
	Slug              string
	Title             string
	DescriptionMD     string
	Difficulty        string
	TimeLimitMS       int
	MemoryLimitKB     int
	CreatedBy         string
	CreatedAt         time.Time
	FunctionSignature []byte // raw JSONB; see internal/harness.ParseSignature
}

type TestCase struct {
	ID                 string
	ProblemID          string
	ArgsJSON           []byte // raw JSONB array of call arguments, in order
	ExpectedReturnJSON []byte // raw JSONB expected return value
	IsSample           bool
	Weight             int
	Ordinal            int
}

type Submission struct {
	ID          string
	UserID      string
	ProblemID   string
	Language    string
	SourceCode  string
	Status      string
	Score       int
	MaxScore    int
	SubmittedAt time.Time
	CompletedAt *time.Time
}

type SubmissionResult struct {
	ID           string
	SubmissionID string
	TestCaseID   string
	Passed       bool
	Stdout       string
	Stderr       string
	RuntimeMS    int
	MemoryKB     int
	ExecStatus   string
}

// SubmissionView bundles a submission with its results for template rendering.
type SubmissionView struct {
	Submission Submission
	Results    []SubmissionResult
}
