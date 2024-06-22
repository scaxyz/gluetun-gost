package main

import (
	"bytes"
	_ "embed"
	"flag"
	"fmt"
	"gluetun-gost/internal"
	"log/slog"
	"os"
	"os/exec"
	"sync"
)

var fDebug = flag.Bool("debug", false, "enable debug logging")
var d = flag.Bool("d", false, "enable debug logging")
var execute = flag.Bool("x", false, "execute changes")

//go:embed routes.txt
var routes string

var debug = false

func main() {
	flag.Parse()

	if *fDebug || *d || os.Getenv("GC_DEBUG") == "true" {
		debug = true
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	if !(*execute) {
		slog.Warn("dry mode, no changes will be made. Use -x to execute")
	}

	server := internal.MustGetEnv("GOST_SERVER")
	gostNet := internal.MustGetEnv("GOST_CLIENT")
	port := internal.MustGetEnv("GOST_PORT")

	uri := fmt.Sprintf("tun://:0/%s:%s?net=%s&name=gost0", server, port, gostNet)

	cmd := exec.Command("gost", fmt.Sprintf("-L=%s", uri))

	outTrigger := internal.NewWriteTrigger(os.Stdout, adjustRoutes)
	errTrigger := internal.NewWriteTrigger(os.Stderr, adjustRoutes)

	cmd.Stderr = errTrigger
	cmd.Stdout = outTrigger

	err := cmd.Run()
	if err != nil {
		slog.Error("failed to run gost", slog.Any("err", err))
		os.Exit(1)
	}
}

func getGostIp() (string, error) {

	cmd := exec.Command("ip", "a", "show", "dev", "gost0")

	outputs, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	_, rest, ok := bytes.Cut(outputs, []byte("inet "))
	if !ok {
		return "", fmt.Errorf("ip a show dev gost0 has no inet address: %s", string(outputs))
	}

	ipAddress, _, ok := bytes.Cut(rest, []byte("/"))
	if !ok {
		return "", fmt.Errorf("ip a show dev gost0 has no inet address: %s", string(outputs))
	}

	return string(ipAddress), nil
}

var adjustRouteMx = sync.Mutex{}

func adjustRoutes() {
	adjustRouteMx.Lock()
	defer adjustRouteMx.Unlock()

	ip, err := getGostIp()
	if err != nil {
		slog.Error("failed to get own gost0 ip", slog.Any("err", err))
		return
	}

	if debug {
		slog.Debug("own gost0 ip", slog.String("ip", ip))
	}

	err = os.Setenv("GOST_CLIENT", ip)
	if err != nil {
		slog.Error("failed to set GOST_CLIENT env", slog.Any("err", err))
		return
	}

	err = internal.AdjustRoutes(routes, debug, *execute, "GOST_CLIENT", "GOST_SERVER")
	if err != nil {
		slog.Error("failed to adjust routes", slog.Any("err", err))
	}

	slog.Info("adjusted routes", "ip", ip)
}
