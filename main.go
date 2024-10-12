package main

import (
	"github.com/gofiber/fiber/v2"
	"embed"
	"io/fs"
	"net/http"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
)

//go:embed all:svelte/build
var embedDirStatic embed.FS

func main() {
	app := fiber.New()

	s, err := fs.Sub(embedDirStatic, "svelte/build")
    if err != nil {
        panic(err)
    }

	staticServer := filesystem.New(filesystem.Config{
		Root: http.FS(s),
	})

	app.Get("/", staticServer)
	app.Get("/_app/*", staticServer)
	app.Get("/service-worker.js", staticServer)
    app.Get("/favicon.png", staticServer)
	app.Get("/*", func(c *fiber.Ctx) error {
		c.Path("/")
		return staticServer(c)
	})

	app.Listen(":8000")
}