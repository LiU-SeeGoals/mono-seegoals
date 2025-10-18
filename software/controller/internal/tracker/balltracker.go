// Description: BallTracker is a struct that contains a KalmanFilter and is used to track the ball in the video feed.
//
package tracker

import "gonum.org/v1/gonum/mat"

type BallTracker struct {
	KalmanFilter
}

func NewBallTracker() *BallTracker {
	X := mat.NewDense(4, 1, []float64{0, 0, 0, 0})

	F := mat.NewDense(4, 4, []float64{
		1, 0, 1, 0,
		0, 1, 0, 1,
		0, 0, 1, 0,
		0, 0, 0, 1})

	H := mat.NewDense(2, 4, []float64{
		1, 0, 0, 0,
		0, 1, 0, 0})

	Q := mat.NewDense(4, 4, []float64{
		0.1, 0, 0, 0,
		0, 0.1, 0, 0,
		0, 0, 0.1, 0,
		0, 0, 0, 0.1})

	R := mat.NewDense(2, 2, []float64{
		0.1, 0,
		0, 0.1})

	P := mat.NewDense(4, 4, []float64{
		0.1, 0, 0, 0,
		0, 0.1, 0, 0,
		0, 0, 0.1, 0,
		0, 0, 0, 0.1})

	dt := 0.1
	return &BallTracker{
		KalmanFilter: KalmanFilter{
			X:  X,
			F:  F,
			H:  H,
			Q:  Q,
			R:  R,
			P:  P,
			dt: dt,
		},
	}
}


