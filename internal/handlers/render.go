package handlers

import (
	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
)

// render writes a templ component directly to the response - templ
// doesn't go through Echo's Renderer interface (which expects a template
// name + map[string]any), it renders straight to an io.Writer.
func render(c echo.Context, status int, component templ.Component) error {
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTMLCharsetUTF8)
	c.Response().WriteHeader(status)
	return component.Render(c.Request().Context(), c.Response())
}
