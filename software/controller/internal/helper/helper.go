package helper

func Contains(slice []int, num int) bool {
	for _, item := range slice {
		if item == num {
			return true
		}
	}
	return false
}
