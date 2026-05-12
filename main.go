package main

import (
	"log"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/compress"
	"github.com/gofiber/fiber/v3/middleware/static"
)

func main() {
	app := fiber.New()
	app.Use(compress.New())

	// Serve static files from the "./public" directory
	app.Get("/*", static.New("./public"))
	// => http://localhost:3000/js/script.js
	// => http://localhost:3000/css/style.css

	app.Get("/prefix*", static.New("./public"))
	// => http://localhost:3000/prefix/js/script.js
	// => http://localhost:3000/prefix/css/style.css

	// Serve a single file for any unmatched routes
	app.Get("*", static.New("./public/index.html"))
	// => http://localhost:3000/any/path/shows/index.html

	log.Fatal(app.Listen(":3000"))
}
