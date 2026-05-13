// RichEdit-based colored text display widget.
// Uses Windows RichEdit 4.1 (RICHEDIT50W class from msftedit.dll)
// to display text with per-character colors.
package main

import (
	"syscall"
	"unsafe"

	"github.com/lxn/walk"
	"github.com/lxn/win"
)

// RichEdit message and mask constants
const (
	EM_SETCHARFORMAT = win.WM_USER + 68
	EM_EXSETSEL      = win.WM_USER + 55
	EM_SETEVENTMASK  = win.WM_USER + 69
	EM_SETBKGNDCOLOR = win.WM_USER + 67

	SCF_SELECTION = 0x0001
	SCF_ALL       = 0x0004

	CFM_COLOR = 0x40000000

	ENM_CHANGE = 0x00000001
	ENM_SCROLL = 0x00000004
)

// CHARFORMAT2W matches the Windows CHARFORMAT2W structure
type CHARFORMAT2W struct {
	cbSize            uint32
	dwMask            uint32
	dwEffects         uint32
	yHeight           int32
	yOffset           int32
	crTextColor       uint32
	bCharSet          byte
	bPitchAndFamily   byte
	szFaceName        [32]uint16
	wWeight           uint16
	sSpacing          int16
	crBackColor       uint32
	lcid              uint32
	dwReserved        uint32
	sStyle            int16
	wKerning          uint16
	bUnderlineType    byte
	bAnimation        byte
	bRevAuthor        byte
	bReserved1        byte
}

var (
	msfteditOnce bool
)

func ensureRichEditLoaded() {
	if msfteditOnce {
		return
	}
	msfteditOnce = true
	syscall.LoadLibrary("msftedit.dll")
}

// RichText is a read-only colored text display widget wrapping RichEdit.
// It embeds walk.Composite to satisfy Walk's widget/layout interface,
// and manages the RichEdit control internally.
type RichText struct {
	walk.Composite
	hwndEdit win.HWND
}

// NewRichText creates a new RichText widget as a child of the given parent.
func NewRichText(parent walk.Container) (*RichText, error) {
	ensureRichEditLoaded()

	rt := new(RichText)

	if err := walk.InitWidget(
		rt,
		parent,
		"RICHEDIT50W",
		win.WS_VISIBLE|win.WS_VSCROLL|win.WS_HSCROLL|
			win.ES_MULTILINE|win.ES_READONLY|win.ES_WANTRETURN|
			win.ES_NOHIDESEL,
		0,
	); err != nil {
		return nil, err
	}

	rt.hwndEdit = rt.Handle()

	// Set font
	font, _ := walk.NewFont("Consolas", 10, 0)
	if font != nil {
		rt.SetFont(font)
	}

	// Set event mask
	win.SendMessage(rt.hwndEdit, EM_SETEVENTMASK, 0, ENM_CHANGE|ENM_SCROLL)

	// Unlimited text
	win.SendMessage(rt.hwndEdit, win.EM_EXLIMITTEXT, 0, 0x7FFFFFFF)

	return rt, nil
}

// SetReadOnly sets the read-only state of the RichEdit control.
func (rt *RichText) SetReadOnly(readOnly bool) error {
	if rt == nil || rt.hwndEdit == 0 {
		return nil
	}
	flag := uintptr(0)
	if readOnly {
		flag = 1
	}
	win.SendMessage(rt.hwndEdit, win.EM_SETREADONLY, flag, 0)
	return nil
}

// AppendText appends plain text at the end.
func (rt *RichText) AppendText(text string) {
	if rt == nil || rt.hwndEdit == 0 || text == "" {
		return
	}
	textUTF16 := syscall.StringToUTF16Ptr(text)
	// Move to end
	charCount := win.SendMessage(rt.hwndEdit, win.WM_GETTEXTLENGTH, 0, 0)
	sel := struct {
		cpMin int32
		cpMax int32
	}{int32(charCount), int32(charCount)}
	win.SendMessage(rt.hwndEdit, EM_EXSETSEL, 0, uintptr(unsafe.Pointer(&sel)))
	win.SendMessage(rt.hwndEdit, win.EM_REPLACESEL, 0, uintptr(unsafe.Pointer(textUTF16)))
	// Scroll to end
	lineCnt := win.SendMessage(rt.hwndEdit, win.EM_GETLINECOUNT, 0, 0)
	if lineCnt > 0 {
		win.SendMessage(rt.hwndEdit, win.EM_LINESCROLL, 0, lineCnt)
	}
}

// AppendColored appends text with a specific color.
func (rt *RichText) AppendColored(text string, color walk.Color) {
	if rt == nil || rt.hwndEdit == 0 || text == "" {
		return
	}

	// Get current end position
	charCount := win.SendMessage(rt.hwndEdit, win.WM_GETTEXTLENGTH, 0, 0)
	startPos := int32(charCount)

	// Insert text at end
	textUTF16 := syscall.StringToUTF16Ptr(text)
	sel := struct {
		cpMin int32
		cpMax int32
	}{startPos, startPos}
	win.SendMessage(rt.hwndEdit, EM_EXSETSEL, 0, uintptr(unsafe.Pointer(&sel)))
	win.SendMessage(rt.hwndEdit, win.EM_REPLACESEL, 0, uintptr(unsafe.Pointer(textUTF16)))

	// Select the newly inserted text
	endPos := startPos + int32(len([]rune(text)))
	sel2 := struct {
		cpMin int32
		cpMax int32
	}{startPos, endPos}
	win.SendMessage(rt.hwndEdit, EM_EXSETSEL, 0, uintptr(unsafe.Pointer(&sel2)))

	// Apply color to selection
	cf := CHARFORMAT2W{
		cbSize:      uint32(unsafe.Sizeof(CHARFORMAT2W{})),
		dwMask:      CFM_COLOR,
		crTextColor: uint32(color),
	}
	win.SendMessage(rt.hwndEdit, EM_SETCHARFORMAT, SCF_SELECTION, uintptr(unsafe.Pointer(&cf)))

	// Scroll to end
	lineCnt := win.SendMessage(rt.hwndEdit, win.EM_GETLINECOUNT, 0, 0)
	if lineCnt > 0 {
		win.SendMessage(rt.hwndEdit, win.EM_LINESCROLL, 0, lineCnt)
	}
}

// SetText replaces all content with plain text.
func (rt *RichText) SetText(text string) {
	if rt == nil || rt.hwndEdit == 0 {
		return
	}
	win.SendMessage(rt.hwndEdit, win.WM_SETTEXT, 0, uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(text))))
}

// Clear removes all text.
func (rt *RichText) Clear() {
	if rt == nil || rt.hwndEdit == 0 {
		return
	}
	rt.SetText("")
}

// SetDefaultColor sets the default text color for all content.
func (rt *RichText) SetDefaultColor(color walk.Color) {
	if rt == nil || rt.hwndEdit == 0 {
		return
	}
	cf := CHARFORMAT2W{
		cbSize:      uint32(unsafe.Sizeof(CHARFORMAT2W{})),
		dwMask:      CFM_COLOR,
		crTextColor: uint32(color),
	}
	win.SendMessage(rt.hwndEdit, EM_SETCHARFORMAT, SCF_ALL, uintptr(unsafe.Pointer(&cf)))
}

// SetBackgroundColor sets the background color.
func (rt *RichText) SetBackgroundColor(color walk.Color) {
	if rt == nil || rt.hwndEdit == 0 {
		return
	}
	win.SendMessage(rt.hwndEdit, EM_SETBKGNDCOLOR, 0, uintptr(color))
}

// GetText returns the current content.
func (rt *RichText) GetText() string {
	if rt == nil || rt.hwndEdit == 0 {
		return ""
	}
	length := win.SendMessage(rt.hwndEdit, win.WM_GETTEXTLENGTH, 0, 0)
	if length == 0 {
		return ""
	}
	buf := make([]uint16, length+1)
	win.SendMessage(rt.hwndEdit, win.WM_GETTEXT, uintptr(length+1), uintptr(unsafe.Pointer(&buf[0])))
	return syscall.UTF16ToString(buf)
}
