package handlers

import (
	"html/template"
	"net/http"

	"codeeval/internal/db"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

// Minimal cookie-based auth suitable for a small trusted internal team.
// For a bigger org, swap this for real sessions (Redis-backed) or OIDC/SSO.

type AuthHandlers struct {
	Store     *db.Store
	Templates *template.Template
}

const userIDKey = "userID"

func UserIDFromContext(c echo.Context) string {
	id, _ := c.Get(userIDKey).(string)
	return id
}

func (h *AuthHandlers) LoginPage(c echo.Context) error {
	return c.Render(http.StatusOK, "login.html", nil)
}

func (h *AuthHandlers) Login(c echo.Context) error {
	email := c.FormValue("email")
	password := c.FormValue("password")

	var userID, hash string
	err := h.Store.Pool.QueryRow(c.Request().Context(),
		"SELECT id, password_hash FROM users WHERE email = $1", email).Scan(&userID, &hash)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid credentials")
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid credentials")
	}

	c.SetCookie(&http.Cookie{
		Name:     "user_id",
		Value:    userID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		// Secure: true, // enable once served over TLS
	})
	return c.Redirect(http.StatusSeeOther, "/problems")
}

func (h *AuthHandlers) Logout(c echo.Context) error {
	c.SetCookie(&http.Cookie{Name: "user_id", Value: "", Path: "/", MaxAge: -1})
	return c.Redirect(http.StatusSeeOther, "/login")
}

// RequireAuth is middleware that redirects unauthenticated users to /login
// and injects the user ID into the request context otherwise.
func RequireAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		cookie, err := c.Cookie("user_id")
		if err != nil || cookie.Value == "" {
			return c.Redirect(http.StatusSeeOther, "/login")
		}
		c.Set(userIDKey, cookie.Value)
		return next(c)
	}
}
