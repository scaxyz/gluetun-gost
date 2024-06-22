package main

import (
	_ "embed"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"slices"
	"strings"
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

	err := savePostRules()
	if err != nil {
		slog.Error("failed to save iptables rules", slog.Any("err", err))
		os.Exit(1)
	}

	err = adjustRoutes()
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

func mustGetEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic(key + " not set in env")
	}
	return value
}

func replaceEnv(data string, envs ...string) string {
	for _, env := range envs {
		envA := fmt.Sprintf("${%s}", env)
		envB := fmt.Sprintf("$%s", env)
		data = strings.ReplaceAll(data, envA, mustGetEnv(env))
		data = strings.ReplaceAll(data, envB, mustGetEnv(env))
	}

	return data
}

func savePostRules() error {

	concretRules := replaceEnv((postRules), "GOST_NET")

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

func adjustRoutes() error {

	concretRoutes := replaceEnv((routes), "GOST_NET", "GOST_SERVER")

	for _, route := range strings.Split(concretRoutes, "\n") {
		if route == "" {
			continue
		}
		fields := strings.Fields(route)
		isDelete := slices.Contains(fields, "del")
		if debug {
			slog.Debug("adjusting routes", slog.String("cmd", route))
		}

		if !(*execute) {
			continue
		}
		cmd := exec.Command(fields[0], fields[1:]...)

		outputs, err := cmd.CombinedOutput()
		if err != nil && !isDelete {
			return fmt.Errorf("failed to execute %s err='%w' output='%s'", route, err, string(outputs))
		}

	}

	return nil
}
