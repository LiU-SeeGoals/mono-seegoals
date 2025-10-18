
# Visualiser

The visualiser is a local GUI that can run directly in go, currently (2025-04-06) used to create plots.
But the GUI can be used to visualise more thigns if desired in the future.

As of (2025-04-06) visualiser consists of an interface with functions used to plot. 
The interface exists in case fayne (current GUI) is bad for some reason, and hopefully it is easy to switch

``` go
func test() {
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
```
