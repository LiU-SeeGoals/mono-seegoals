package main

import (
	"image"
	"fmt"
	"sync"
	// "time"

	fyne "fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"
	fynecanvas "fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
)

type Visualiser interface {
	// A visualiser is a specific type of GUI app
	// e.g. Fayne
	CreatePlotWindow(p *plot.Plot) Figure

	// Start the GUI application, generally is blocking and
	// needs to run on main thread
	Run()
}

type Figure interface{
	// A figure belongs to a specific visualiser
	// e.g. A FayneFigure is used to update plots in Fayne windows

	UpdatePlotWindow(p *plot.Plot)
}

var (
	Vis Visualiser
	VisOnce sync.Once
)

// Visualiser Singleton Structure
type FayneVisualiser struct {
	app        fyne.App
	mainWindow fyne.Window
}

type FayneFigure struct {
	imageCanvas *fynecanvas.Image
}

/*
	Creates singleton visualiser that needs to run from main thread

	Return another Visualiser to change the backend GUI
*/
func GetVisualiser() Visualiser {
	VisOnce.Do(func() {
		visApp := fyneapp.New()
		FayneVis := &FayneVisualiser{app: visApp}
		Vis = FayneVis

		// Create a persistent invisible window
		// One window needs to be open or main thread will die
		FayneVis.mainWindow = visApp.NewWindow("Persistent Window")
		FayneVis.mainWindow.Resize(fyne.NewSize(0, 0))
		FayneVis.mainWindow.SetFixedSize(true)
		FayneVis.mainWindow.SetCloseIntercept(func() {
			FayneVis.mainWindow.Hide()
		})
		FayneVis.mainWindow.Show() // Keep this window always open

		hide := func(){
			// Hack to hide the main window on startup to keep app running....
			// time.Sleep(100 * time.Millisecond)
			FayneVis.mainWindow.Hide()
		}
		visApp.Lifecycle().SetOnStarted(hide)
		visApp.Lifecycle().SetOnEnteredForeground(hide)
		visApp.Lifecycle().SetOnExitedForeground(hide)
	})
	return Vis
}

func (vis *FayneVisualiser) Run() {
	vis.app.Run()
}

func (fig *FayneFigure) UpdatePlotWindow(p *plot.Plot) {
	img := getPlotImg(p)

	fig.imageCanvas.Image = img
	fynecanvas.Refresh(fig.imageCanvas)

}

// CreatePlotWindow creates a new plotting window
func (vis *FayneVisualiser) CreatePlotWindow(p *plot.Plot) Figure {
	w := vis.app.NewWindow(p.Title.Text)

	imageCanvas := fynecanvas.NewImageFromImage(image.NewAlpha(image.Rect(0, 0, 600, 400)))
	imageCanvas.FillMode = fynecanvas.ImageFillOriginal

	content := container.NewCenter(imageCanvas)
	w.SetContent(content)
	w.Resize(fyne.NewSize(700, 500))
	w.Show()

	// Get image of current plot
	img := getPlotImg(p)

	imageCanvas.Image = img
	fynecanvas.Refresh(imageCanvas)
	return &FayneFigure{imageCanvas}
}

func getPlotImg(p *plot.Plot) image.Image{

	imgCanvas := vgimg.New(6*vg.Inch, 4*vg.Inch)
	p.Draw(draw.New(imgCanvas))
	return imgCanvas.Image()
}

func scatterPlt(points plotter.XYs) *plot.Plot {
	p := plot.New()
	p.Title.Text = "Real-Time Plot"
	p.X.Label.Text = "X-axis"
	p.Y.Label.Text = "Y-axis"

	scatter, err := plotter.NewScatter(points)
	if err != nil {
		panic(err)
	}
	p.Add(scatter)

	return p
}
func linePlt(points plotter.XYs) *plot.Plot {
	p := plot.New()
	p.Title.Text = "Real-Time Plot"
	p.X.Label.Text = "X-axis"
	p.Y.Label.Text = "Y-axis"

	line, err := plotter.NewLine(points)
	if err != nil {
		panic(err)
	}
	p.Add(line)

	return p
}

func main() {
	// Initialize the global Visualiser
	vis := GetVisualiser()

	// Start plot creation in separate goroutines
	go func() {
		points1 := plotter.XYs{
			{X: 0, Y: 0},
			{X: 1, Y: 1},
			{X: 2, Y: 4},
			{X: 3, Y: 9},
		}
		p := scatterPlt(points1)
		vis.CreatePlotWindow(p)
	}()

	go func() {
		points2 := plotter.XYs{
			{X: 0, Y: 0},
			{X: 1, Y: 1},
			{X: 2, Y: 4},
		}
		p := linePlt(points2)
		fig := vis.CreatePlotWindow(p)
		for{
			val := points2[len(points2) - 1]
			newx := val.X + 1
			newy := newx * newx
			fmt.Println(val)

			points2 = append(points2, plotter.XY{X: newx, Y: newy})

			p := scatterPlt(points2)
			fig.UpdatePlotWindow(p)
		}
	}()

	// Run the application main event loop
	vis.Run()
}
