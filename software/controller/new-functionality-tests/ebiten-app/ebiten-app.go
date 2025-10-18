package main

import (
	// "bytes"
	"image"
	imagedraw "image/draw"

	// "image/png"
	"log"
	"math"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	vdraw "gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
)

const (
	screenWidth  = 800
	screenHeight = 600
)

type RealTimePlotter struct {
	sync.Mutex
	buffers [2]*ebiten.Image
	active  int
	t       float64
}

func NewRealTimePlotter() *RealTimePlotter {
	rtp := &RealTimePlotter{}
	rtp.buffers[0] = ebiten.NewImage(screenWidth, screenHeight)
	rtp.buffers[1] = ebiten.NewImage(screenWidth, screenHeight)
	go rtp.updateLoop()
	return rtp
}

func (r *RealTimePlotter) updateLoop() {
	for {
		r.generatePlot()
		time.Sleep(100 * time.Millisecond)
	}
}

func (r *RealTimePlotter) generatePlot() {
	r.Lock()
	defer r.Unlock()

	p := plot.New()
	p.Title.Text = "Real-Time Sine Wave"
	p.X.Label.Text = "Time"
	p.Y.Label.Text = "Amplitude"

	n := 100
	pts := make(plotter.XYs, n)
	for i := 0; i < n; i++ {
		x := r.t + float64(i)*0.1
		pts[i].X = x
		pts[i].Y = math.Sin(x)
	}
	r.t += 0.1

	line, err := plotter.NewLine(pts)
	if err != nil {
		log.Println(err)
		return
	}
	p.Add(line)

	// Match Ebiten's dimensions exactly
	imgCanvas := vgimg.NewWith(
		vgimg.UseWH(vg.Length(screenWidth), vg.Length(screenHeight)),
		vgimg.UseDPI(96),
	)

	p.Draw(vdraw.New(imgCanvas))

	// Directly convert canvas image to RGBA
	img := imgCanvas.Image()

	rgbaImg := image.NewRGBA(img.Bounds())
	imagedraw.Draw(rgbaImg, img.Bounds(), img, image.Point{}, imagedraw.Src)

	// Update buffer with correctly sized RGBA image
	inactiveIdx := (r.active + 1) % 2
	r.buffers[inactiveIdx].WritePixels(rgbaImg.Pix)
	r.active = inactiveIdx
}

type Game struct {
	plotter *RealTimePlotter
}

func (g *Game) Update() error {
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.plotter.Lock()
	buffer := g.plotter.buffers[g.plotter.active]
	g.plotter.Unlock()

	screen.DrawImage(buffer, nil)
	ebitenutil.DebugPrint(screen, "Real-Time Double Buffered Plot (Fixed)")
}

func (g *Game) Layout(_, _ int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	plotter := NewRealTimePlotter()
	game := &Game{plotter: plotter}

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Real-Time Plot Renderer")
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
