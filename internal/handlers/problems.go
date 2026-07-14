package handlers

import (
	"encoding/json"
	"html/template"
	"net/http"

	"codeeval/internal/db"
	"codeeval/internal/harness"
	"codeeval/internal/piston"

	"github.com/labstack/echo/v4"
)

type ProblemHandlers struct {
	Store     *db.Store
	Templates *template.Template
}

func (h *ProblemHandlers) List(c echo.Context) error {
	problems, err := h.Store.ListProblems(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to load problems")
	}
	return c.Render(http.StatusOK, "problems_list.html", map[string]any{
		"Problems": problems,
	})
}

func (h *ProblemHandlers) Detail(c echo.Context) error {
	slug := c.Param("slug")

	problem, err := h.Store.GetProblemBySlug(c.Request().Context(), slug)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to load problem")
	}
	if problem == nil {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	samples, err := h.Store.ListSampleTestCases(c.Request().Context(), problem.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to load samples")
	}

	sig, err := harness.ParseSignature(problem.FunctionSignature)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "invalid function signature")
	}
	stubs := map[string]string{}
	for lang := range piston.LanguageRuntimes {
		stubs[lang] = harness.Stub(sig, lang)
	}
	stubsJSON, err := json.Marshal(stubs)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to build editor stubs")
	}

	return c.Render(http.StatusOK, "problem_detail.html", map[string]any{
		"Problem":   problem,
		"Samples":   samples,
		"StubsJSON": template.JS(stubsJSON),
	})
}
