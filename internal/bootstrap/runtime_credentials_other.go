//go:build !unix && !windows

package bootstrap

import "os"

func credentialRedirected(info os.FileInfo) bool { return true }

func openCredentialFile(root *os.Root, name string) (*os.File, error) {
	return nil, errRuntimeCredential
}
