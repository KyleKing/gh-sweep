package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewModel(t *testing.T) {
	m := NewModel()

	if m.ready {
		t.Error("Expected model to not be ready initially")
	}

	if m.width != 0 {
		t.Errorf("Expected width to be 0, got %d", m.width)
	}
}

func TestModelInit(t *testing.T) {
	m := NewModel()
	cmd := m.Init()

	if cmd != nil {
		t.Error("Expected Init() to return nil")
	}
}

func TestModelUpdate(t *testing.T) {
	m := NewModel()

	// Test window size message
	msg := tea.WindowSizeMsg{Width: 100, Height: 50}
	updated, _ := m.Update(msg)
	updatedModel := updated.(Model)

	if !updatedModel.ready {
		t.Error("Expected model to be ready after WindowSizeMsg")
	}

	if updatedModel.width != 100 {
		t.Errorf("Expected width to be 100, got %d", updatedModel.width)
	}

	if updatedModel.height != 50 {
		t.Errorf("Expected height to be 50, got %d", updatedModel.height)
	}
}

func TestModelView(t *testing.T) {
	m := NewModel()

	// Test not ready state
	view := m.View()
	if !strings.Contains(view, "Initializing") {
		t.Error("Expected view to show 'Initializing' when not ready")
	}

	// Test ready state
	m.ready = true
	m.width = 100
	m.height = 50

	view = m.View()
	if !strings.Contains(view, "gh-sweep") {
		t.Error("Expected view to contain 'gh-sweep'")
	}
}
