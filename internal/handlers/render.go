package handlers

import (
	"html/template"
	"io"

	"github.com/labstack/echo/v4"
)

// Renderer adapts the app's shared html/template set to Echo's Renderer
// interface so handlers can call c.Render(status, name, data).
type Renderer struct {
	templates *template.Template
}

func NewRenderer(tmpl *template.Template) *Renderer {
	return &Renderer{templates: tmpl}
}

func (r *Renderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return r.templates.ExecuteTemplate(w, name, data)
}
