// Package piston wraps the Piston REST API (https://github.com/engineer-man/piston)
// for sandboxed code execution. Works against a self-hosted Piston instance
// (the default; see docker-compose.yml) or any other deployment that speaks
// the same v2 API - just swap BaseURL.
package piston

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// LanguageRuntimes maps our app-level language keys to Piston's
// {language, version} runtime selector. "*" picks the latest installed
// version for that language. Extend as needed; keep in sync with the
// language <select> in templates/problem_detail.html and Monaco's
// monacoLangMap.
var LanguageRuntimes = map[string]struct{ Language, Version string }{
	"python3":    {"python", "*"},
	"go":         {"go", "*"},
	"java":       {"java", "*"},
	"cpp":        {"c++", "*"},
	"c":          {"c", "*"},
	"javascript": {"javascript", "*"},
}

type Client struct {
	BaseURL    string
	APIKey     string // optional; sent as "Authorization: Bearer <key>" if set
	HTTPClient *http.Client
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL:    baseURL,
		APIKey:     apiKey,
		HTTPClient: &http.Client{Timeout: 20 * time.Second},
	}
}

type file struct {
	Content string `json:"content"`
}

type executeRequest struct {
	Language       string `json:"language"`
	Version        string `json:"version"`
	Files          []file `json:"files"`
	Stdin          string `json:"stdin"`
	RunTimeout     int    `json:"run_timeout,omitempty"`
	RunMemoryLimit int    `json:"run_memory_limit,omitempty"`
}

// Stage mirrors one execution stage (compile or run) in Piston's response.
type Stage struct {
	Stdout string  `json:"stdout"`
	Stderr string  `json:"stderr"`
	Output string  `json:"output"`
	Code   *int    `json:"code"`
	Signal *string `json:"signal"`
}

// Result mirrors the fields we care about from Piston's /execute response.
type Result struct {
	Language string `json:"language"`
	Version  string `json:"version"`
	Compile  *Stage `json:"compile"` // present only for compiled languages
	Run      Stage  `json:"run"`
}

// maxRunTimeoutMS is Piston's own configured ceiling for run_timeout - it
// rejects the request outright (not just clamps it) if exceeded, so a
// problem's time_limit_ms must never be passed through uncapped.
const maxRunTimeoutMS = 3000

// RunOne submits one (source, stdin) pair and blocks until Piston finishes
// it - Piston's /execute endpoint is synchronous by design, no polling
// needed.
func (c *Client) RunOne(ctx context.Context, sourceCode, language, stdin string, timeLimitMS, memoryLimitKB int) (*Result, error) {
	runtime, ok := LanguageRuntimes[language]
	if !ok {
		return nil, fmt.Errorf("unsupported language: %s", language)
	}

	if timeLimitMS > maxRunTimeoutMS {
		timeLimitMS = maxRunTimeoutMS
	}

	reqBody := executeRequest{
		Language:       runtime.Language,
		Version:        runtime.Version,
		Files:          []file{{Content: sourceCode}},
		Stdin:          stdin,
		RunTimeout:     timeLimitMS,
		RunMemoryLimit: memoryLimitKB * 1024, // Piston limits are in bytes; ours are in KB
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/execute", c.BaseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("piston request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("piston returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result Result
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("decoding piston response: %w", err)
	}

	return &result, nil
}

// Status synthesizes a human-readable execution status and pass/fail
// eligibility from a Piston result, since Piston (unlike Judge0) has no
// single status field - just exit codes and signals per stage.
func Status(res *Result) (ok bool, status string) {
	if res.Compile != nil && res.Compile.Code != nil && *res.Compile.Code != 0 {
		return false, "Compile Error"
	}
	if res.Run.Signal != nil {
		return false, fmt.Sprintf("Killed (signal %s)", *res.Run.Signal)
	}
	if res.Run.Code != nil && *res.Run.Code != 0 {
		return false, fmt.Sprintf("Runtime Error (exit %d)", *res.Run.Code)
	}
	return true, "OK"
}
