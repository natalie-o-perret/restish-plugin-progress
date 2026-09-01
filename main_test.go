package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/rest-sh/restish/v2/plugin"
)

func TestFormatterStreamsProgress(t *testing.T) {
	var out bytes.Buffer
	f := &formatter{w: &out}
	for _, body := range []any{
		map[string]any{"id": "deploy", "current": 1, "total": 2, "message": "first"},
		map[string]any{"id": "deploy", "current": 1, "total": 2, "message": "first"},
		map[string]any{"id": "deploy", "state": "success", "current": 2, "total": 2},
	} {
		if err := f.Handle(plugin.FormatterRequest{Event: "item", Response: plugin.FormatterResponse{Body: body}}); err != nil {
			t.Fatal(err)
		}
	}
	want := "deploy [##########----------] 50% (1/2 steps) running: first\n" +
		"deploy [####################] 100% (2/2 steps) success\n"
	if got := out.String(); got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestRenderUsesSingularStep(t *testing.T) {
	current, total := int64(1), int64(1)
	want := "workflow [####################] 100% (1/1 step) success"
	if got := render(progress{Label: "workflow", State: "success", Current: &current, Total: &total}); got != want {
		t.Fatalf("render() = %q, want %q", got, want)
	}
}

func TestFormatterRedrawsTTYLine(t *testing.T) {
	var out bytes.Buffer
	f := &formatter{w: &out}
	if err := f.Handle(plugin.FormatterRequest{Event: "start", Color: true}); err != nil {
		t.Fatal(err)
	}
	if err := f.Handle(plugin.FormatterRequest{Event: "item", Response: plugin.FormatterResponse{Body: map[string]any{"label": "workflow"}}}); err != nil {
		t.Fatal(err)
	}
	if err := f.Handle(plugin.FormatterRequest{Event: "end"}); err != nil {
		t.Fatal(err)
	}
	if got, want := out.String(), "\r\x1b[2Kworkflow running\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestDecodeProgressRejectsIncompleteCount(t *testing.T) {
	_, err := decodeProgress(map[string]any{"label": "workflow", "current": 1})
	if err == nil || !strings.Contains(err.Error(), "current and total together") {
		t.Fatalf("error = %v", err)
	}
}
