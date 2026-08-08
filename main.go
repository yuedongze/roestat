// Command roestat is a terminal UI for ROEST coffee roasters: a live view of
// the current roast and a browsable history of past roasts.
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/yuedongze/roestat/internal/roest"
	"github.com/yuedongze/roestat/internal/ui"
)

func main() {
	loadDotenv(".env")

	client, err := roest.NewClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, "roestat: "+err.Error())
		fmt.Fprintln(os.Stderr, "set ROEST_CLIENT_ID and ROEST_CLIENT_SECRET (e.g. in .env)")
		os.Exit(1)
	}

	p := tea.NewProgram(ui.NewApp(client), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "roestat: "+err.Error())
		os.Exit(1)
	}
}

// loadDotenv is a minimal .env loader mirroring live_roast.py: it sets keys
// only if not already present in the environment (real env wins) and strips
// surrounding quotes. Missing file is not an error.
func loadDotenv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.Trim(strings.TrimSpace(val), `"'`)
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, val)
		}
	}
}
