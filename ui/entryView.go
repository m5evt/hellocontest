package ui

import (
	"fmt"
	"log"
	"strings"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"

	"github.com/ftl/hellocontest/core"
)

// EntryController controls the entry of QSO data.
type EntryController interface {
	GotoNextField() core.EntryField
	TabNextField() core.EntryField
	SetActiveField(core.EntryField)

	ToggleWorkmode()

	Enter(string)
	SendQuestion()
	StopTX()

	KeyerInc()
	KeyerDec()

	FButton(fkey int)
	EscapeStateMachine()

	Log()
	Clear()
}

type entryView struct {
	controller EntryController

	ignoreInput bool

	entryRoot    *gtk.Grid
	utc          *gtk.Label
	frequency    *gtk.Label
	callsign     *gtk.Entry
	theirReport  *gtk.Entry
	theirXchange *gtk.Entry
	band         *gtk.Label
	mode         *gtk.ComboBoxText
	logButton    *gtk.Button
	clearButton  *gtk.Button
	messageLabel *gtk.Label
	cwspeedLabel *gtk.Label
	wmLabel      *gtk.Label
}

func setupEntryView(builder *gtk.Builder) *entryView {
	result := new(entryView)

	result.entryRoot = getUI(builder, "entryGrid").(*gtk.Grid)
	result.utc = getUI(builder, "utcLabel").(*gtk.Label)
	result.callsign = getUI(builder, "callsignEntry").(*gtk.Entry)
	result.theirReport = getUI(builder, "theirReportEntry").(*gtk.Entry)
	result.theirXchange = getUI(builder, "theirXchangeEntry").(*gtk.Entry)
	result.band = getUI(builder, "bandLabel").(*gtk.Label)
	result.mode = getUI(builder, "modeCombo").(*gtk.ComboBoxText)
	result.logButton = getUI(builder, "logButton").(*gtk.Button)
	result.clearButton = getUI(builder, "clearButton").(*gtk.Button)
	result.messageLabel = getUI(builder, "messageLabel").(*gtk.Label)
	result.cwspeedLabel = getUI(builder, "cwspeedLabel").(*gtk.Label)
	result.wmLabel = getUI(builder, "workmodeLabe").(*gtk.Label)

	result.addEntryEventHandlers(&result.callsign.Widget)
	result.addEntryEventHandlers(&result.theirReport.Widget)
	result.addEntryEventHandlers(&result.theirXchange.Widget)
	result.addEntryEventHandlers(&result.mode.Widget)

	result.logButton.Connect("clicked", result.onLogButtonClicked)
	result.clearButton.Connect("clicked", result.onClearButtonClicked)

	setupModeCombo(result.mode)

	addStyleClass(&result.messageLabel.Widget, "message")
	return result
}

func setupModeCombo(combo *gtk.ComboBoxText) {
	combo.RemoveAll()
	for _, value := range core.Modes {
		combo.Append(value.String(), value.String())
	}
	combo.SetActive(0)
}

func (v *entryView) SetEntryController(controller EntryController) {
	v.controller = controller
}

func (v *entryView) addEntryEventHandlers(w *gtk.Widget) {
	w.Connect("key_press_event", v.onEntryKeyPress)
	w.Connect("focus_in_event", v.onEntryFocusIn)
	w.Connect("focus_out_event", v.onEntryFocusOut)
	w.Connect("changed", v.onEntryChanged)
}

func (v *entryView) onEntryKeyPress(_ interface{}, event *gdk.Event) bool {
	keyEvent := gdk.EventKeyNewFromEvent(event)
	switch keyEvent.KeyVal() {
	case gdk.KEY_Tab:
		v.controller.TabNextField()
		return true
	case gdk.KEY_Return:
		v.controller.GotoNextField()
		return true
	case gdk.KEY_question:
		v.controller.SendQuestion()
		return true
	case gdk.KEY_plus:
		v.controller.ToggleWorkmode()
		return true
	case gdk.KEY_Page_Up:
		v.controller.KeyerInc()
		return true
	case gdk.KEY_Page_Down:
		v.controller.KeyerDec()
		return true
	case gdk.KEY_F1:
		v.controller.FButton(0)
		return true
	case gdk.KEY_F2:
		v.controller.FButton(1)
		return true
	case gdk.KEY_F3:
		v.controller.FButton(2)
		return true
	case gdk.KEY_F4:
		v.controller.FButton(3)
		return true
	case gdk.KEY_Escape:
		v.controller.EscapeStateMachine()
		return true
	default:
		return false
	}
}

func (v *entryView) onEntryFocusIn(widget interface{}, _ *gdk.Event) bool {
	var field core.EntryField
	switch w := widget.(type) {
	case *gtk.Entry:
		field = v.widgetToField(&w.Widget)
	case *gtk.ComboBoxText:
		field = v.widgetToField(&w.Widget)
	default:
		field = core.OtherField
	}
	v.controller.SetActiveField(field)
	return false
}

func (v *entryView) onEntryFocusOut(widget interface{}, _ *gdk.Event) bool {
	if entry, ok := widget.(*gtk.Entry); ok {
		entry.SelectRegion(0, 0)
	}
	return false
}

func (v *entryView) onEntryChanged(widget interface{}) bool {
	if v.controller == nil {
		return false
	}
	if v.ignoreInput {
		return false
	}

	switch w := widget.(type) {
	case *gtk.Entry:
		text, err := w.GetText()
		if err != nil {
			return false
		}
		w.SetText(strings.TrimSpace(strings.ToUpper(text)))
		v.controller.Enter(text)
	case *gtk.ComboBoxText:
		activeField := v.widgetToField(&w.Widget)
		v.controller.SetActiveField(activeField)
		text := w.GetActiveText()
		v.controller.Enter(text)
	}

	return false
}

func (v *entryView) onLogButtonClicked(button *gtk.Button) bool {
	v.controller.Log()
	return true
}

func (v *entryView) onClearButtonClicked(button *gtk.Button) bool {
	v.controller.Clear()
	return true
}

func (v *entryView) setTextWithoutChangeEvent(f func(string), value string) {
	v.ignoreInput = true
	defer func() { v.ignoreInput = false }()
	f(value)
}

func (v *entryView) SetUTC(text string) {
	runAsync(func() {
		v.utc.SetText(text)
	})
}

func (v *entryView) SetFrequency(frequency core.Frequency) {
	runAsync(func() {
		//v.frequency.SetText(fmt.Sprintf("%7.2f kHz", frequency/1000.0))
	})
}

func (v *entryView) SetCallsign(text string) {
	v.setTextWithoutChangeEvent(v.callsign.SetText, text)
}

func (v *entryView) SetTheirReport(text string) {
	v.setTextWithoutChangeEvent(v.theirReport.SetText, text)
}

func (v *entryView) SetTheirXchange(text string) {
	v.setTextWithoutChangeEvent(v.theirXchange.SetText, text)
}

func (v *entryView) SetBand(text string) {
	runAsync(func() {
		v.band.SetText(text)
	})
}

func (v *entryView) SetMode(text string) {
	runAsync(func() {
		v.setTextWithoutChangeEvent(func(s string) { v.mode.SetActiveID(s) }, text)
	})
}

func (v *entryView) EnableExchangeFields(theirNumber, theirXchange bool) {
	v.theirXchange.SetSensitive(theirXchange)
}

func (v *entryView) SetActiveField(field core.EntryField) {
	widget := v.fieldToWidget(field)
	widget.GrabFocus()
}

func (v *entryView) fieldToWidget(field core.EntryField) *gtk.Widget {
	switch field {
	case core.CallsignField:
		return &v.callsign.Widget
	case core.TheirReportField:
		return &v.theirReport.Widget
	case core.TheirXchangeField:
		return &v.theirXchange.Widget
	case core.BandField:
		return &v.band.Widget
	case core.ModeField:
		return &v.mode.Widget
	case core.OtherField:
		return &v.callsign.Widget
	default:
		log.Fatalf("Unknown entry field %d", field)
	}
	panic("this is never reached")
}

func (v *entryView) widgetToField(widget *gtk.Widget) core.EntryField {
	name, _ := widget.GetName()
	switch name {
	case "callsignEntry":
		return core.CallsignField
	case "theirReportEntry":
		return core.TheirReportField
	case "theirXchangeEntry":
		return core.TheirXchangeField
	case "bandCombo":
		return core.BandField
	case "modeCombo":
		return core.ModeField
	default:
		return core.OtherField
	}
}

func (v *entryView) SetDuplicateMarker(duplicate bool) {
	if duplicate {
		addStyleClass(&v.callsign.Widget, "duplicate")
	} else {
		removeStyleClass(&v.callsign.Widget, "duplicate")
	}
}

func (v *entryView) SetEditingMarker(editing bool) {
	if editing {
		addStyleClass(&v.callsign.Widget, "editing")
	} else {
		removeStyleClass(&v.callsign.Widget, "editing")
	}
}

func (v *entryView) ShowMessage(args ...interface{}) {
	v.messageLabel.SetText(fmt.Sprint(args...))
}

func (v *entryView) ClearMessage() {
	v.messageLabel.SetText("")
}

func (v *entryView) ShowKeyerSpeed(speed int) {
	v.cwspeedLabel.SetText(fmt.Sprintf("%2d", speed))
}

func (v *entryView) ShowWorkmode(text string) {
	v.wmLabel.SetText(text)
}
