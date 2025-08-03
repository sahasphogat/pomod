package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"time"
)

const (
	socketPath    = "/tmp/pomod.sock"
	workDuration  = 50 * 60
	breakDuration = 10 * 60
)

type State struct {
	Mode       string
	Running    bool
	TimeLeft   int
	ActiveTime int
	StartTime  time.Time
}

var state = State{
	Mode:     "work",
	TimeLeft: workDuration,
}

func main() {
	os.Remove(socketPath)
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		fmt.Println("Error creating socket:", err)
		return
	}
	defer ln.Close()

	go autoFinishLoop()

	for {
		conn, err := ln.Accept()
		if err == nil {
			go handleConn(conn)
		}
	}
}

func handleConn(c net.Conn) {
	defer c.Close()
	buf := make([]byte, 128)
	n, _ := c.Read(buf)
	cmd := string(buf[:n])

	switch cmd {
	case "toggle":
		if state.Running {
			elapsed := int(time.Since(state.StartTime).Seconds())
			state.TimeLeft -= elapsed
			if state.TimeLeft < 0 {
				state.TimeLeft = 0
			}
			state.ActiveTime += elapsed
			state.Running = false
			runHook("paused")
		} else {
			state.StartTime = time.Now()
			state.Running = true
			runHook("resumed")
		}

	case "finish":
		if state.Running {
			elapsed := int(time.Since(state.StartTime).Seconds())
			state.TimeLeft -= elapsed
			if state.TimeLeft < 0 {
				state.TimeLeft = 0
			}
			state.ActiveTime += elapsed
			state.Running = false
			runHook("paused")
		}
		logSession()
		if state.Mode == "work" {
			runHook("session_finished")
			switchMode()
		} else {
			runHook("break_finished")
			switchMode()
		}

	case "status":
		timeLeft := state.TimeLeft
		if state.Running {
			elapsed := int(time.Since(state.StartTime).Seconds())
			timeLeft = state.TimeLeft - elapsed
			if timeLeft < 0 {
				timeLeft = 0
			}
		}
		out := map[string]interface{}{
			"mode":      state.Mode,
			"running":   state.Running,
			"time_left": timeLeft,
		}
		j, _ := json.Marshal(out)
		c.Write(j)

	default:
		c.Write([]byte("unknown command"))
	}
}

func autoFinishLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if state.Running {
			elapsed := int(time.Since(state.StartTime).Seconds())
			if elapsed >= state.TimeLeft {
				state.ActiveTime += state.TimeLeft
				state.TimeLeft = 0
				state.Running = false
				logSession()
				if state.Mode == "work" {
					runHook("session_finished")
					switchMode()
				} else {
					runHook("break_finished")
					switchMode()
				}
			}
		}
	}
}

func switchMode() {
	if state.Mode == "work" {
		state.Mode = "break"
		state.TimeLeft = breakDuration
	} else {
		state.Mode = "work"
		state.TimeLeft = workDuration
	}
	state.ActiveTime = 0
	state.Running = false
}

func logSession() {
	if state.ActiveTime < 60 {
		return
	}
	entry := map[string]interface{}{
		"type":     state.Mode,
		"start":    state.StartTime.Add(-time.Duration(state.ActiveTime) * time.Second).Format(time.RFC3339),
		"end":      time.Now().Format(time.RFC3339),
		"duration": state.ActiveTime,
	}
	usr, _ := user.Current()
	logDir := filepath.Join(usr.HomeDir, ".local", "share", "pomod")
	os.MkdirAll(logDir, 0755)
	f, err := os.OpenFile(filepath.Join(logDir, "log.jsonl"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		j, _ := json.Marshal(entry)
		f.Write(j)
		f.Write([]byte("\n"))
		f.Close()
	}
}

func runHook(name string) {
	usr, _ := user.Current()
	hookPath := filepath.Join(usr.HomeDir, ".local", "share", "pomod", "hooks", name)

	if _, err := os.Stat(hookPath); err == nil {
		cmd := exec.Command(hookPath)
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
		cmd.Stdin = nil
		cmd.Start()
	}
}

