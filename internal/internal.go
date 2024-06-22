package internal

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"slices"
	"strings"
)

type WriteTrigger struct {
	writer  io.Writer
	trigger func()
}

func NewWriteTrigger(writer io.Writer, trigger func()) *WriteTrigger {
	return &WriteTrigger{writer: writer, trigger: trigger}
}

func (w *WriteTrigger) Write(p []byte) (n int, err error) {
	w.trigger()
	return w.writer.Write(p)
}

func MustGetEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic(key + " not set in env")
	}
	return value
}

func ReplaceEnv(data string, envs ...string) string {
	for _, env := range envs {
		envA := fmt.Sprintf("${%s}", env)
		envB := fmt.Sprintf("$%s", env)
		data = strings.ReplaceAll(data, envA, MustGetEnv(env))
		data = strings.ReplaceAll(data, envB, MustGetEnv(env))
	}

	return data
}

func AdjustRoutes(routes string, debug bool, execute bool, envs ...string) error {

	concretRoutes := ReplaceEnv((routes), envs...)

	for _, route := range strings.Split(concretRoutes, "\n") {
		if route == "" {
			continue
		}
		fields := strings.Fields(route)
		isDelete := slices.Contains(fields, "del")
		if debug {
			slog.Debug("adjusting routes", slog.String("cmd", route))
		}

		if !execute {
			continue
		}
		cmd := exec.Command(fields[0], fields[1:]...)

		outputs, err := cmd.CombinedOutput()
		if err != nil {
			if !isDelete {
				return fmt.Errorf("failed to execute %s err='%w' output='%s'", route, err, string(outputs))
			}
		}

		if debug {
			slog.Debug("adjusting routes",
				slog.String("output", string(outputs)),
				slog.String("cmd", route),
				slog.Int("exit-code", cmd.ProcessState.ExitCode()))
		}

	}

	return nil
}
