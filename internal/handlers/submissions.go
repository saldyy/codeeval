package handlers

import (
	"encoding/json"
	"html/template"
	"net/http"
	"reflect"
	"strings"

	"codeeval/internal/db"
	"codeeval/internal/harness"
	"codeeval/internal/models"
	"codeeval/internal/piston"

	"github.com/labstack/echo/v4"
)

type SubmissionHandlers struct {
	Store     *db.Store
	Piston    *piston.Client
	Templates *template.Template
}

// Create handles an HTMX form POST from the problem page, runs the code
// against all test cases synchronously (fine at small team scale), and
// returns an HTML partial that HTMX swaps into the results panel.
func (h *SubmissionHandlers) Create(c echo.Context) error {
	ctx := c.Request().Context()
	slug := c.Param("slug")
	userID := UserIDFromContext(c)

	language := c.FormValue("language")
	sourceCode := c.FormValue("source_code")

	problem, err := h.Store.GetProblemBySlug(ctx, slug)
	if err != nil || problem == nil {
		return echo.NewHTTPError(http.StatusNotFound, "problem not found")
	}

	sig, err := harness.ParseSignature(problem.FunctionSignature)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "invalid function signature")
	}
	wrapped := harness.Wrap(sig, language, sourceCode)

	testCases, err := h.Store.ListAllTestCases(ctx, problem.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to load test cases")
	}

	maxScore := 0
	for _, tc := range testCases {
		maxScore += tc.Weight
	}

	submissionID, err := h.Store.CreateSubmission(ctx, models.Submission{
		UserID:     userID,
		ProblemID:  problem.ID,
		Language:   language,
		SourceCode: sourceCode,
		MaxScore:   maxScore,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create submission")
	}

	score := 0
	var results []models.SubmissionResult
	overallStatus := "passed"

	for _, tc := range testCases {
		res, err := h.Piston.RunOne(ctx, wrapped, language, string(tc.ArgsJSON), problem.TimeLimitMS, problem.MemoryLimitKB)
		if err != nil {
			overallStatus = "error"
			results = append(results, models.SubmissionResult{
				SubmissionID: submissionID,
				TestCaseID:   tc.ID,
				Passed:       false,
				Stderr:       err.Error(),
				ExecStatus:   "client_error",
			})
			continue
		}

		ok, execStatus := piston.Status(res)
		stdout := strings.TrimRight(res.Run.Stdout, "\n")
		stderr := res.Run.Stderr

		passed := false
		if ok {
			var actual, expected any
			_ = json.Unmarshal([]byte(strings.TrimSpace(stdout)), &actual)
			_ = json.Unmarshal(tc.ExpectedReturnJSON, &expected)
			passed = reflect.DeepEqual(actual, expected)
		}
		if passed {
			score += tc.Weight
		} else if overallStatus == "passed" {
			overallStatus = "failed"
		}

		sr := models.SubmissionResult{
			SubmissionID: submissionID,
			TestCaseID:   tc.ID,
			Passed:       passed,
			Stdout:       stdout,
			Stderr:       stderr,
			ExecStatus:   execStatus,
		}
		if err := h.Store.InsertSubmissionResult(ctx, sr); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to save result")
		}
		results = append(results, sr)
	}

	if err := h.Store.UpdateSubmissionResult(ctx, submissionID, overallStatus, score); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update submission")
	}

	return c.Render(http.StatusOK, "partials/submission_result.html", map[string]any{
		"Status":    overallStatus,
		"Score":     score,
		"MaxScore":  maxScore,
		"Results":   results,
		"TestCases": testCases,
	})
}
