package helper

// NB_KeepLatestChan creates a pair of channels, one for sending and one for receiving.
// The sender channel will not block, but will keep the latest value sent to it.
// The receiver channel will always have the latest value sent by the sender channel.
// The receiver channel will block until a value is sent by the sender channel.
// This is a custom implementation so you have to be careful with it.

func NB_KeepLatestChan[M any]() (chan M, chan M) {
	from := make(chan M, 1)
	to := make(chan M, 1)
	go pipe(from, to)
	return from, to
}

func pipe[M any](from <-chan M, to chan M) {
	// close the channel when the sender is done
	defer close(to)
	// keep reading from the channel until it is closed
	for v := range from {
		select {
		// try to send the value to the channel
		case to <- v:
			// easy case, just we just pass the value along
			continue
		default:
			// if we can't send the value to the because the channel is full
			select {
			// try to read from the channel to make room for the new value
			case <-to:
				to <- v
			// someone read from the channel before we could make room. its ok,
			// we will send the packet to the empty channel
			default:
				to <- v
			}
		}

	}
}

func NB_Send[M any](c chan *M, v *M) {
	select {
	case c <- v:
	default:
		select {
		case <-c:
			c <- v
		default:
			c <- v
		}
	}
}

func NB_Receive[M any](from chan *M, v *M) {
	select {
	case v = <-from:
	default:
	}
}
