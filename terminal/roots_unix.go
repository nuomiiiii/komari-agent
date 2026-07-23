//go:build !windows

package terminal

import "os"

func filesystemRoots() []string { return []string{"/"} }
func pathSeparator() string     { return string(os.PathSeparator) }
