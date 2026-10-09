//go:build !windows

package agentexec

func createPrivateLogDirectory(string) error { return nil }

func verifyPrivateLogDirectory(string) error { return nil }

func verifyPrivateLogFile(string) error { return nil }
