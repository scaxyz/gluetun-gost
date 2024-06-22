package main

import (
	_ "embed"
	"flag"
	"gluetun-gost/internal"
	"log/slog"
	"os"
	"os/exec"
)

//go:embed iptables/post-rules.txt
var postRules string

//go:embed routes.txt
var routes string

var fDebug = flag.Bool("debug", false, "enable debug logging")
var d = flag.Bool("d", false, "enable debug logging")
var execute = flag.Bool("x", false, "execute changes")
var debug = false

func main() {
	flag.Parse()

	if *fDebug || *d || os.Getenv("GG_DEBUG") == "true" {
		debug = true
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	if !(*execute) {
		slog.Warn("dry mode, no changes will be made. Use -x to execute")
	}

	err := savePostRules()
	if err != nil {
		slog.Error("failed to save iptables rules", slog.Any("err", err))
		os.Exit(1)
	}

	err = internal.AdjustRoutes(routes, debug, *execute, "GOST_NET", "GOST_SERVER")
	if err != nil {
		slog.Error("failed to adjust routes", slog.Any("err", err))
		os.Exit(1)
	}

	if debug {
		slog.Debug("starting gluetun-entrypoint")
	}

	if !(*execute) {
		os.Exit(0)
	}

	cmd := exec.Command("/gluetun-entrypoint")
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err = cmd.Run()

	if err != nil {
		slog.Error("gluetun-entrypoint failed", slog.Any("err", err))
		os.Exit(1)
	}

}

func savePostRules() error {

	concretRules := internal.ReplaceEnv((postRules), "GOST_NET")

	if debug {
		slog.Debug("writing post-rules.txt", slog.String("rules", concretRules))
	}

	if !(*execute) {
		return nil
	}

	err := os.MkdirAll("/iptables", 0755)
	if err != nil {
		return err
	}

	err = os.WriteFile("/iptables/post-rules.txt", []byte(concretRules), 0644)
	if err != nil {
		return err
	}

	return nil
}
