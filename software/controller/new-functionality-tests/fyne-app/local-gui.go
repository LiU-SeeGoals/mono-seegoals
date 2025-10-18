package main

import (
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/canvas"
	"image/color"
)

func main() {
	a := app.New()
	w := a.NewWindow("Football Field")

	// Create a green rectangle as the football field
	field := canvas.NewRectangle(color.RGBA{0, 128, 0, 255})
	field.Resize(fyne.NewSize(800, 400))
	field.Move(fyne.NewPos(50, 50))  // Important: Positioning the field on the canvas

	// Create some circles to represent players
	player1 := canvas.NewCircle(color.RGBA{255, 0, 0, 255}) // Red team
	player1.Resize(fyne.NewSize(20, 20))
	player1.Move(fyne.NewPos(200, 200))

	player2 := canvas.NewCircle(color.RGBA{0, 0, 255, 255}) // Blue team
	player2.Resize(fyne.NewSize(20, 20))
	player2.Move(fyne.NewPos(400, 200))

	// Add everything to a CanvasObject and render it
	content := container.NewWithoutLayout(field, player1, player2)
	w.SetContent(content)

	w.Resize(fyne.NewSize(900, 500))
	w.ShowAndRun()
}
