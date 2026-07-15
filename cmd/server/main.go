package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"codeeval/internal/db"
	"codeeval/internal/handlers"
	"codeeval/internal/piston"

	"github.com/labstack/echo/v4"
)

func main() {
	ctx := context.Background()

	dsn := getenv("DATABASE_URL", "postgres://codeeval:codeeval@localhost:5432/codeeval?sslmode=disable")
	pistonURL := getenv("PISTON_URL", "http://localhost:2000/api/v2")
	pistonKey := os.Getenv("PISTON_API_KEY") // empty is fine for self-hosted
	addr := getenv("ADDR", ":8080")

	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		log.Fatalf("db connection failed: %v", err)
	}
	defer pool.Close()
	store := db.NewStore(pool)

	p0 := piston.NewClient(pistonURL, pistonKey)

	problemH := &handlers.ProblemHandlers{Store: store}
	submissionH := &handlers.SubmissionHandlers{Store: store, Piston: p0}
	authH := &handlers.AuthHandlers{Store: store}

	e := echo.New()
	e.HideBanner = true

	e.Static("/static", "static")

	e.GET("/login", authH.LoginPage)
	e.POST("/login", authH.Login)
	e.GET("/logout", authH.Logout)

	e.GET("/problems", problemH.List, handlers.RequireAuth)
	e.GET("/problems/:slug", problemH.Detail, handlers.RequireAuth)
	e.POST("/problems/:slug/submit", submissionH.Create, handlers.RequireAuth)

	e.GET("/", func(c echo.Context) error {
		return c.Redirect(http.StatusSeeOther, "/problems")
	})

	log.Printf("listening on %s", addr)
	e.Logger.Fatal(e.Start(addr))
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
