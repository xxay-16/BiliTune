//go:build !production

package main

import (
	"os"
	"strconv"
)

// Opt-in CDP endpoint for testing this application's isolated WebView in dev builds.
func browserDebugArgs() []string {
	port, err := strconv.Atoi(os.Getenv("BILITUNE_DEBUG_PORT"))
	if err != nil || port < 1024 || port > 65535 {
		return nil
	}
	return []string{"--remote-debugging-port=" + strconv.Itoa(port)}
}
