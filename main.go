package main

import (
	_ "embed"
	"log"

	"github.com/wailsapp/wails"
)

//go:embed frontend/build/main.js
var js string

func main() {
	log.Println("Starting API Watcher...")

	app := wails.CreateApp(&wails.AppConfig{
		Width:  1200,
		Height: 800,
		Title:  "API Watcher",
		JS:     js,
		Colour: "#131313",
	})

	// Bind the App struct to expose methods to JavaScript/React
	apiApp := NewApp()
	app.Bind(apiApp)

	log.Println("Running application...")
	app.Run()
}
