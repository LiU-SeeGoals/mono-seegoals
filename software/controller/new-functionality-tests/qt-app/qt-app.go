
package main

import (
	"os"

	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/gui"
	"github.com/therecipe/qt/widgets"
)

func main() {
	app := widgets.NewQApplication(len(os.Args), os.Args)

	window := widgets.NewQMainWindow(nil, 0)
	window.SetWindowTitle("Football Field")
	window.SetFixedSize2(800, 500)

	fieldWidget := widgets.NewQWidget(nil, 0)
	fieldWidget.ConnectPaintEvent(func(event *gui.QPaintEvent) {
		drawField(fieldWidget)
	})

	window.SetCentralWidget(fieldWidget)
	window.Show()

	app.Exec()
}

// Corrected drawing function
func drawField(widget *widgets.QWidget) {
	painter := gui.NewQPainter2(widget)

	// Field background color (green)
	fieldRect := core.NewQRect(0, 0, 800, 500)
	painter.FillRect4(fieldRect, gui.NewQColor3(34, 139, 34, 255))

	// White Pen for lines
	pen := gui.NewQPen3(gui.NewQColor2(core.Qt__white))
	pen.SetWidth(2)
	painter.SetPen(pen)

	// Outer boundary
	painter.DrawRect3(fieldRect)

	// Midline
	painter.DrawLine3(400, 0, 400, 500)

	// Center circle
	centerCircleRect := core.NewQRectF4(340, 190, 120, 120)
	painter.DrawEllipse(centerCircleRect)

	// Left goal area
	painter.DrawRect3(core.NewQRect(0, 150, 100, 200))

	// Right goal area
	painter.DrawRect3(core.NewQRect(700, 150, 100, 200))

	// Left penalty area
	painter.DrawRect3(core.NewQRect(0, 100, 150, 300))

	// Right penalty area
	painter.DrawRect3(core.NewQRect(650, 100, 150, 300))

	// Left penalty mark
	painter.SetBrush(gui.NewQBrush2(core.Qt__white, core.Qt__SolidPattern))
	leftPenaltyMark := core.NewQRectF4(105, 245, 10, 10)
	painter.DrawEllipse(leftPenaltyMark)

	// Right penalty mark
	rightPenaltyMark := core.NewQRectF4(685, 245, 10, 10)
	painter.DrawEllipse(rightPenaltyMark)

	painter.End()
}

