package worker

const maxAttempts = 3

func AttemptsBeforeFailure() int {
	return maxAttempts
}
