package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/rest-sh/restish/v2/plugin"
	"github.com/schollz/progressbar/v3"
)

type barStyle struct {
	Width int
	Color string
	Fill  string
	Head  string
	Empty string
	Start string
	End   string
}

func defaultBarStyle() barStyle {
	return barStyle{Width: 24, Color: "cyan", Fill: "█", Head: "█", Empty: "░"}
}

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
	style    barStyle
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

	style, err := barStyleFromEnv()
	if err != nil {
		fail(err)
	}
	f := &formatter{w: os.Stdout, style: style}
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
	line, err := render(p, f.style, f.tty)
	if err != nil {
		return err
	}
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

func render(p progress, style barStyle, color bool) (string, error) {
	if p.Total == nil {
		return progressDescription(p), nil
	}
	theme := progressbar.Theme{
		Saucer:        style.Fill,
		SaucerHead:    style.Head,
		SaucerPadding: style.Empty,
		BarStart:      style.Start,
		BarEnd:        style.End,
	}
	if color {
		theme.Saucer = "[" + style.Color + "]" + theme.Saucer + "[reset]"
		theme.SaucerHead = "[" + style.Color + "]" + theme.SaucerHead + "[reset]"
	}
	bar := progressbar.NewOptions64(*p.Total,
		progressbar.OptionSetWriter(io.Discard),
		progressbar.OptionSetWidth(style.Width),
		progressbar.OptionSetTheme(theme),
		progressbar.OptionEnableColorCodes(color),
		progressbar.OptionSetPredictTime(false),
		progressbar.OptionSetElapsedTime(false),
		progressbar.OptionShowDescriptionAtLineEnd(),
		progressbar.OptionSetDescription(progressDescription(p)),
	)
	if err := bar.Set64(*p.Current); err != nil {
		return "", fmt.Errorf("render progress: %w", err)
	}
	return strings.TrimSpace(strings.TrimPrefix(bar.String(), "\r")), nil
}

func progressDescription(p progress) string {
	parts := []string{p.Label}
	if p.Total != nil {
		unit := "steps"
		if *p.Total == 1 {
			unit = "step"
		}
		parts = append(parts, fmt.Sprintf("%d/%d %s", *p.Current, *p.Total, unit))
	}
	parts = append(parts, p.State)
	description := strings.Join(parts, "  ")
	if p.Message != "" {
		description += ": " + p.Message
	}
	return description
}

func barStyleFromEnv() (barStyle, error) {
	style := defaultBarStyle()
	if value := os.Getenv("RSH_PROGRESS_WIDTH"); value != "" {
		width, err := strconv.Atoi(value)
		if err != nil || width < 1 || width > 200 {
			return barStyle{}, fmt.Errorf("RSH_PROGRESS_WIDTH must be an integer from 1 to 200")
		}
		style.Width = width
	}
	for name, target := range map[string]*string{
		"RSH_PROGRESS_FILL":  &style.Fill,
		"RSH_PROGRESS_HEAD":  &style.Head,
		"RSH_PROGRESS_EMPTY": &style.Empty,
		"RSH_PROGRESS_START": &style.Start,
		"RSH_PROGRESS_END":   &style.End,
	} {
		if value, ok := os.LookupEnv(name); ok {
			*target = value
		}
	}
	if color := os.Getenv("RSH_PROGRESS_COLOR"); color != "" {
		if !validColor(color) {
			return barStyle{}, fmt.Errorf("RSH_PROGRESS_COLOR must be black, blue, cyan, green, magenta, red, white, or yellow")
		}
		style.Color = color
	}
	if style.Fill == "" || style.Head == "" || style.Empty == "" {
		return barStyle{}, fmt.Errorf("RSH_PROGRESS_FILL, RSH_PROGRESS_HEAD, and RSH_PROGRESS_EMPTY cannot be empty")
	}
	return style, nil
}

func validColor(color string) bool {
	switch color {
	case "black", "blue", "cyan", "green", "magenta", "red", "white", "yellow":
		return true
	default:
		return false
	}
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
