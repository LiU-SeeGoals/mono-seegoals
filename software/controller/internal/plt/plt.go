package plt

import (
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/plotter"
)

var p *plot.Plot

func Init() {
	p = plot.New();
	// p.X.Max = 4000
	// p.Y.Max = 2000
	// p.X.Min = -4000
	// p.Y.Min = -2000
}

func InitFieldPlot() {
	p = plot.New();
	p.X.Max = 4000
	p.Y.Max = 2000
	p.X.Min = -4000
	p.Y.Min = -2000
}

func Get() *plot.Plot {
	return p
}

func SaveFig(name string) {
	err := p.Save(8 * vg.Inch, 8 * vg.Inch, name)
	if err != nil{
		panic(err)
	}
}

func Scatter(points plotter.XYs) {

	scatter, err := plotter.NewScatter(points)
	if err != nil {
		panic(err)
	}
	p.Add(scatter)
}

func RayPlt(start plotter.XY, points plotter.XYs) *plot.Plot {
	p := plot.New()
	p.Title.Text = "Real-Time Plot"
	p.X.Label.Text = "X-axis"
	p.Y.Label.Text = "Y-axis"

	for i := range points {
		line, err := plotter.NewLine(plotter.XYs{start, points[i]})
		if err != nil {
			panic(err)
		}
		p.Add(line)
	}

	return p
}

func Line(start plotter.XY, end plotter.XY) {
	points := plotter.XYs{start, end}

	line, err := plotter.NewLine(points)
	if err != nil {
		panic(err)
	}
	p.Add(line)
}

func Plot(points plotter.XYs) {

	line, err := plotter.NewLine(points)
	if err != nil {
		panic(err)
	}
	p.Add(line)
}
