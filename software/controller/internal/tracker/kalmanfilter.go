// Description: This file contains the implementation of the Kalman Filter.
// https://medium.com/team-rover/understanding-kalman-filter-and-its-equations-5fcc5d5fe61e
// https://pkg.go.dev/gonum.org/v1/gonum/mat
package tracker

import (
	"gonum.org/v1/gonum/mat"
	. "github.com/LiU-SeeGoals/controller/internal/logger"
)

type KalmanFilter struct {
	X  *mat.Dense // State vector
	F  *mat.Dense // State transition matrix
	H  *mat.Dense // Observation matrix
	Q  *mat.Dense // Process noise covariance
	R  *mat.Dense // Measurement noise covariance
	P  *mat.Dense // Estimate error covariance
	B  *mat.Dense // Control matrix
	dt float64    // Time step
}

func NewKalmanFilter(X, F, H, Q, R, P *mat.Dense, dt float64) *KalmanFilter {
	return &KalmanFilter{
		X:  X,
		F:  F,
		H:  H,
		Q:  Q,
		R:  R,
		P:  P,
		dt: dt,
	}

}

func (kf *KalmanFilter) Predict(controlMatrix *mat.Dense) {

	// Make a prediction of the current state based on th eprevious state
	// by multiplying the state transition matrix by the previous state
	// X = F * X
	kf.X.Mul(kf.X, kf.F)

	// If a control matrix is provided, update the state based on the control matrix
	// X = X + B * u
	if controlMatrix != nil {
		var temp mat.Dense
		temp.Mul(controlMatrix, kf.Q) // temp = B * u
		kf.X.Add(kf.X, &temp)         // X = X + temp
	}

	// Update the estimate error covariance
	// P = F * P * F^T+ Q
	var temp mat.Dense
	temp.Mul(kf.P, kf.F.T()) // temp = P * F^T
	kf.P.Mul(kf.F, &temp)    // P = F * temp
	kf.P.Add(kf.P, kf.Q)     // P = P + Q
}

func (kf *KalmanFilter) Update(Z *mat.Dense) {

	// ---Innovation or measurement residual---
	// Y = Z - H * X
	var temp mat.Dense
	temp.Mul(kf.H, kf.X) // temp = H * X
	var Y mat.Dense
	Y.Sub(Z, &temp) // Y = Z - temp

	// ---Kalman gain---
	// K = P * H^T * (H * P * H^T + R)^-1
	var K mat.Dense
	K.Mul(kf.P, kf.H.T()) // ek = P * H^T
	// S = H * P * H^T + R
	var S mat.Dense
	S.Mul(kf.H, kf.P)   // S = H * P
	S.Mul(&S, kf.H.T()) // S = S * H^T
	S.Add(&S, kf.R)     // S = S + R
	// Use QR decomposition
	var qr mat.QR
	qr.Factorize(&S)
	r, c := S.Dims()
	I := mat.NewDense(r, c, nil)
	for i := 0; i < r; i++ {
		I.Set(i, i, 1.0)
	}
	S_inv := mat.NewDense(3, 3, nil)
	err := qr.SolveTo(S_inv, false, I)
	if err != nil {
		Logger.Fatal("QR decomposition failed: ", err)
	}
	K.Mul(&K, S_inv) // K = K * S_inv

	// ---Update the state estimate---
	// X = X + K * Y
	var temp2 mat.Dense
	temp2.Mul(&K, &Y) // temp2 = K * Y
	kf.X.Add(kf.X, &temp2) // X = X + temp2

	// ---Update the estimate error covariance---
	// P = (I - K * H) * P
	var temp3 mat.Dense
	temp3.Mul(&K, kf.H) // temp3 = K * H
	r, c = temp3.Dims()
	I = mat.NewDense(r, c, nil)
	for i := 0; i < r; i++ {
		I.Set(i, i, 1.0)
	}
	I.Sub(I, &temp3) // I = I - temp3
	kf.P.Mul(I, kf.P) // P = I * P

}
