// Build formatted output panes for history and other views.
package main

import (
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

// FormattedOutput creates a Composite containing a RichText display widget
// pre-populated with syntax-highlighted log output.
// Use as a drop-in replacement for readonly TextEdit views.
func FormattedOutput(content string, minHeight int) Widget {
	return newFormattedPane(content, minHeight, true)
}

// FormattedScript creates a Composite containing a RichText display widget
// pre-populated with shell-syntax-highlighted script content.
func FormattedScript(content string, minHeight int) Widget {
	return newFormattedPane(content, minHeight, false)
}

// FormattedCodeByType creates a Composite containing a RichText display widget
// with syntax highlighting based on file type (sh, py, json, yaml).
func FormattedCodeByType(content string, fileType string, minHeight int) Widget {
	return newFormattedPaneByType(content, fileType, minHeight)
}

func newFormattedPane(content string, minHeight int, isLog bool) Widget {
	var comp *walk.Composite
	return Composite{
		AssignTo: &comp,
		Layout:   VBox{MarginsZero: true},
		MinSize:  Size{0, minHeight},
		Children: []Widget{},
		OnBoundsChanged: func() {
			// Create RichText child on first size calculation
			if comp != nil && comp.Children().Len() == 0 {
				rt, err := NewRichText(comp)
				if err == nil {
					rt.SetReadOnly(true)
					if isLog {
						FormatLogOutput(rt, content)
					} else {
						FormatScriptContent(rt, content)
					}
				}
			}
		},
	}
}

func newFormattedPaneByType(content string, fileType string, minHeight int) Widget {
	var comp *walk.Composite
	return Composite{
		AssignTo: &comp,
		Layout:   VBox{MarginsZero: true},
		MinSize:  Size{0, minHeight},
		Children: []Widget{},
		OnBoundsChanged: func() {
			// Create RichText child on first size calculation
			if comp != nil && comp.Children().Len() == 0 {
				rt, err := NewRichText(comp)
				if err == nil {
					rt.SetReadOnly(true)
					FormatCodeByType(rt, content, fileType)
				}
			}
		},
	}
}
