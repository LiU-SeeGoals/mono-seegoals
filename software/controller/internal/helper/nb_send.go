package helper

// NB_Send puts v on c without blocking. When c is full the oldest queued value
// is dropped to make room, so receivers always see the most recent values.
func NB_Send[M any](c chan *M, v *M) {
	for {
		select {
		case c <- v:
			return
		default:
		}
		select {
		case <-c:
		default:
		}
	}
}
