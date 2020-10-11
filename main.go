package main

import (
	"flag"
	"go2dgraphics/internal/game"
	_ "image/png"
	"log"
	"runtime"
)

func init() {
	// GLFW event handling must run on the main OS thread
	runtime.LockOSThread()
}

func main() {
	windowConfig := &game.WindowConfig{}

	flag.StringVar(&windowConfig.Title, "window-title", "OpenGL Go Sprite Example", "Game window width")
	flag.IntVar(&windowConfig.Width, "window-width", 1440, "Game window width")
	flag.IntVar(&windowConfig.Height, "window-height", 810, "Game window height")

	gameWindow, err := game.NewWindow(windowConfig)
	if err != nil {
		log.Fatalln(err.Error())
	}

	gameWindow.OpenAndWait()
}
