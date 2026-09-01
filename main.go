package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/rest-sh/restish/v2/plugin"
)

const barWidth = 20

type progress struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	State   string `json:"state"`
	Current *int64 `json:"current"`
	Total   *int64 `json:"total"`
	Message string `json:"message"`
}

type formatter struct {
	w        io.Writer
	tty      bool
	active   bool
	lastLine string
}

func main() {
	manifest := plugin.Manifest{
		Name:              "progress",
		Version:           "0.1.0",
		Description:       "Render streamed progress records as a terminal progress bar",
		RestishAPIVersion: 2,
		Hooks:             []string{"formatter"},
		FormatterNames:    []string{"progress"},
	}
	if plugin.HandleStartupFlags(os.Stdout, manifest, nil) {
		return
	}

	f := &formatter{w: os.Stdout}
	dec := plugin.NewDecoder(os.Stdin)
	for {
		var req plugin.FormatterRequest
		if err := dec.ReadMessage(&req); err != nil {
			fail(fmt.Errorf("read formatter request: %w", err))
		}
		if req.Type != "formatter" || req.Format != "progress" {
			fail(fmt.Errorf("unexpected formatter request %q/%q", req.Type, req.Format))
		}
		if err := f.Handle(req); err != nil {
			fail(err)
		}
		if req.Event == "end" {
			return
		}
	}
}

func (f *formatter) Handle(req plugin.FormatterRequest) error {
	switch req.Event {
	case "start", "item":
		if req.Color {
			f.tty = true
		}
		if req.Response.Body == nil {
			return nil
		}
		p, err := decodeProgress(req.Response.Body)
		if err != nil {
			return err
		}
		return f.write(p)
	case "end":
		if f.tty && f.active {
			_, err := fmt.Fprintln(f.w)
			f.active = false
			return err
		}
		return nil
	default:
		return fmt.Errorf("unsupported formatter event %q", req.Event)
	}
}

func decodeProgress(value any) (progress, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return progress{}, fmt.Errorf("encode progress record: %w", err)
	}
	var p progress
	if err := json.Unmarshal(raw, &p); err != nil {
		return progress{}, fmt.Errorf("progress formatter expects an object with integer current and total fields: %w", err)
	}
	if p.Label == "" {
		p.Label = p.ID
	}
	if p.Label == "" {
		return progress{}, fmt.Errorf("progress formatter requires label or id")
	}
	if p.State == "" {
		p.State = "running"
	}
	if p.Current != nil && *p.Current < 0 || p.Total != nil && *p.Total < 0 {
		return progress{}, fmt.Errorf("progress formatter requires non-negative current and total")
	}
	if (p.Current == nil) != (p.Total == nil) {
		return progress{}, fmt.Errorf("progress formatter requires current and total together")
	}
	if p.Total != nil && *p.Total == 0 {
		return progress{}, fmt.Errorf("progress formatter requires total greater than zero")
	}
	if p.Total != nil && *p.Current > *p.Total {
		return progress{}, fmt.Errorf("progress formatter requires current no greater than total")
	}
	return p, nil
}

func (f *formatter) write(p progress) error {
	line := render(p)
	if line == f.lastLine {
		return nil
	}
	f.lastLine = line
	if !f.tty {
		_, err := fmt.Fprintln(f.w, line)
		return err
	}
	if _, err := fmt.Fprintf(f.w, "\r\x1b[2K%s", line); err != nil {
		return err
	}
	f.active = true
	if terminalState(p.State) {
		_, err := fmt.Fprintln(f.w)
		f.active = false
		return err
	}
	return nil
}

func render(p progress) string {
	var parts []string
	parts = append(parts, p.Label)
	if p.Total != nil {
		total := *p.Total
		current := min(*p.Current, total)
		filled := 0
		if total > 0 {
			filled = int(current * barWidth / total)
		}
		parts = append(parts, "["+strings.Repeat("#", filled)+strings.Repeat("-", barWidth-filled)+"]")
		unit := "steps"
		if total == 1 {
			unit = "step"
		}
		parts = append(parts, fmt.Sprintf("%d%% (%d/%d %s)", current*100/total, current, total, unit))
	}
	parts = append(parts, p.State)
	line := strings.Join(parts, " ")
	if p.Message != "" {
		line += ": " + p.Message
	}
	return line
}

func terminalState(state string) bool {
	switch state {
	case "success", "failure", "failed", "error", "cancelled", "canceled":
		return true
	default:
		return false
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
