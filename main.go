package main

import (
	"database/sql"
	"embed"
	"io/fs"
	"log"
	"net/http"
	"time"
	"os"
	"runtime"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	_ "github.com/mattn/go-sqlite3"
	"github.com/gofiber/fiber/v2/middleware/cache"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/goccy/go-json"
)

//go:embed all:svelte/build
var embedDirStatic embed.FS

type Tes struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	config := fiber.Config{
		CaseSensitive:            true,
		StrictRouting:            true,
		DisableHeaderNormalizing: true,
		ServerHeader:             "go",
		JSONEncoder:              json.Marshal,
		JSONDecoder:              json.Unmarshal,
	}

	for i := range os.Args[1:] {
		if os.Args[1:][i] == "-prefork" {
			config.Prefork = true
		}
	}

	app := fiber.New(config)

	s, err := fs.Sub(embedDirStatic, "svelte/build")
	if err != nil {
		panic(err)
	}

	staticServer := filesystem.New(filesystem.Config{
		Root: http.FS(s),
	})

	db, err := sql.Open("sqlite3", "./fuji.db")
	if err != nil {
		log.Fatal(err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	defer db.Close()

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS tes (id INTEGER NOT NULL PRIMARY KEY, name TEXT)`)
	if err != nil {
		log.Fatal(err)
	}

	stmt, err := db.Prepare("INSERT INTO tes(name) VALUES(?)")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	_, err = stmt.Exec("fuji")
	if err != nil {
		log.Fatal(err)
	}

	app.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed,
	}))
	app.Use(cache.New(cache.Config{
		Expiration: 30 * time.Minute,
		CacheControl: true,
	}))

	app.Get("/", staticServer)
	app.Get("/_app/*", staticServer)
	app.Get("/service-worker.js", staticServer)
	app.Get("/favicon.png", staticServer)
	app.Get("/tes", func(c *fiber.Ctx) error {
		return handleTes(c, db)
	})
	app.Get("/*", func(c *fiber.Ctx) error {
		c.Path("/")
		return staticServer(c)
	})

	app.Listen(":8888")
}

func handleTes(c *fiber.Ctx, db *sql.DB) error {
	rows, err := db.Query("SELECT id, name FROM tes")
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString("Error querying data")
	}
	defer rows.Close()

	var results []Tes
	for rows.Next() {
		var tes Tes
		if err := rows.Scan(&tes.Id, &tes.Name); err != nil {
			return c.Status(http.StatusInternalServerError).SendString("Error scanning row")
		}
		results = append(results, tes)
	}

	return c.JSON(&results)
}