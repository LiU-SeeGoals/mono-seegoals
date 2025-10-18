package main

import (
    "math/rand"
    "os"

    "gonum.org/v1/plot"
    "gonum.org/v1/plot/plotter"
    "gonum.org/v1/plot/vg"
    "gonum.org/v1/plot/vg/draw"
    "gonum.org/v1/plot/vg/vgimg"
)

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
func subplots(){
	points2 := plotter.XYs{
		{X: 0, Y: 0},
		{X: 1, Y: 1},
		{X: 2, Y: 4},
	}

    const rows, cols = 2, 1
    plots := make([][]*plot.Plot, rows)
    for j := range rows {
        plots[j] = make([]*plot.Plot, cols)
        for i := range cols {

			p := linePlt(points2)

            // make sure the horizontal scales match
            p.X.Min = 0
            p.X.Max = 5

            plots[j][i] = p
        }
    }

    img := vgimg.New(vg.Points(150), vg.Points(175))
    dc := draw.New(img)

    t := draw.Tiles{
        Rows: rows,
        Cols: cols,
    }

    canvases := plot.Align(plots, t, dc)
    for j := range rows {
        for i := range cols {
            if plots[j][i] != nil {
                plots[j][i].Draw(canvases[j][i])
            }
        }
    }
}

func main() {
    rand.Seed(int64(0))


    w, err := os.Create("aligned.png")
    if err != nil {
        panic(err)
    }

    png := vgimg.PngCanvas{Canvas: img}
    if _, err := png.WriteTo(w); err != nil {
        panic(err)
    }
}
