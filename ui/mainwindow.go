package ui

import (
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/ftl/hellocontest/core"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

type mainWindow struct {
	styleProvider *gtk.CssProvider

	window       *gtk.ApplicationWindow
	callsign     *gtk.Entry
	theirReport  *gtk.Entry
	theirNumber  *gtk.Entry
	theirXchange *gtk.Entry
	band         *gtk.ComboBoxText
	mode         *gtk.ComboBoxText
	myReport     *gtk.Entry
	myNumber     *gtk.Entry
	myXchange    *gtk.Entry
	logButton    *gtk.Button
	resetButton  *gtk.Button
	messageLabel *gtk.Label

	menuFileNew    *gtk.MenuItem
	menuFileOpen   *gtk.MenuItem
	menuFileSaveAs *gtk.MenuItem
	menuFileQuit   *gtk.MenuItem

	qsoView *gtk.TreeView
	qsoList *gtk.ListStore

	ignoreComboChange bool

	log   core.Log
	app   core.AppController
	entry core.EntryController
}

func setupMainWindow(builder *gtk.Builder, application *gtk.Application) *mainWindow {
	result := new(mainWindow)

	result.styleProvider = setupStyleProvider()

	result.window = getUI(builder, "mainWindow").(*gtk.ApplicationWindow)
	result.window.SetApplication(application)
	result.window.SetDefaultSize(500, 500)

	result.menuFileNew = getUI(builder, "menuFileNew").(*gtk.MenuItem)
	result.menuFileOpen = getUI(builder, "menuFileOpen").(*gtk.MenuItem)
	result.menuFileSaveAs = getUI(builder, "menuFileSaveAs").(*gtk.MenuItem)
	result.menuFileQuit = getUI(builder, "menuFileQuit").(*gtk.MenuItem)

	result.callsign = getUI(builder, "callsignEntry").(*gtk.Entry)
	result.theirReport = getUI(builder, "theirReportEntry").(*gtk.Entry)
	result.theirNumber = getUI(builder, "theirNumberEntry").(*gtk.Entry)
	result.theirXchange = getUI(builder, "theirXchangeEntry").(*gtk.Entry)
	result.band = getUI(builder, "bandCombo").(*gtk.ComboBoxText)
	result.mode = getUI(builder, "modeCombo").(*gtk.ComboBoxText)
	result.myReport = getUI(builder, "myReportEntry").(*gtk.Entry)
	result.myNumber = getUI(builder, "myNumberEntry").(*gtk.Entry)
	result.myXchange = getUI(builder, "myXchangeEntry").(*gtk.Entry)
	result.logButton = getUI(builder, "logButton").(*gtk.Button)
	result.resetButton = getUI(builder, "resetButton").(*gtk.Button)
	result.messageLabel = getUI(builder, "errorLabel").(*gtk.Label)

	result.addEntryTraversal(result.callsign)
	result.addEntryTraversal(result.theirReport)
	result.addEntryTraversal(result.theirNumber)
	result.addEntryTraversal(result.theirXchange)
	result.addEntryTraversal(result.myReport)
	result.addEntryTraversal(result.myNumber)
	result.addEntryTraversal(result.myXchange)
	result.addOtherWidgetTraversal(&result.band.Widget)
	result.band.Connect("changed", result.onBandChanged)
	result.addOtherWidgetTraversal(&result.mode.Widget)
	result.mode.Connect("changed", result.onModeChanged)
	result.logButton.Connect("clicked", result.onLogButtonClicked)
	result.resetButton.Connect("clicked", result.onResetButtonClicked)

	setupBandCombo(result.band)
	setupModeCombo(result.mode)
	result.qsoView = getUI(builder, "qsoView").(*gtk.TreeView)
	result.qsoList = setupQsoView(result.qsoView)

	result.addStyleProvider(&result.myNumber.Widget)

	result.menuFileNew.Connect("activate", result.onNew)
	result.menuFileOpen.Connect("activate", result.onOpen)
	result.menuFileSaveAs.Connect("activate", result.onSaveAs)
	result.menuFileQuit.Connect("activate", result.onQuit)

	return result
}

func setupStyleProvider() *gtk.CssProvider {
	provider, err := gtk.CssProviderNew()
	if err != nil {
		log.Fatalf("Cannot create CSS provider: %v", err)
	}
	provider.LoadFromData(`
		.duplicate {background-color: #FF0000; color: #FFFFFF;}
	`)
	return provider
}

func (w *mainWindow) addStyleProvider(widget *gtk.Widget) {
	context, err := widget.GetStyleContext()
	if err != nil {
		log.Printf("Cannot get style context: %v", err)
		return
	}
	context.AddProvider(w.styleProvider, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)
}

func addStyleClass(widget *gtk.Widget, class string) {
	doWithStyle(widget, func(style *gtk.StyleContext) {
		style.AddClass(class)
	})
}

func removeStyleClass(widget *gtk.Widget, class string) {
	doWithStyle(widget, func(style *gtk.StyleContext) {
		style.RemoveClass(class)
	})
}

func doWithStyle(widget *gtk.Widget, do func(*gtk.StyleContext)) error {
	style, err := widget.GetStyleContext()
	if err != nil {
		return err
	}
	do(style)
	return nil
}

func setupBandCombo(combo *gtk.ComboBoxText) {
	combo.RemoveAll()
	for _, value := range core.Bands {
		combo.Append(value.String(), value.String())
	}
	combo.SetActive(0)
}

func setupModeCombo(combo *gtk.ComboBoxText) {
	combo.RemoveAll()
	for _, value := range core.Modes {
		combo.Append(value.String(), value.String())
	}
	combo.SetActive(0)
}

const (
	qsoColumnUTC int = iota
	qsoColumnCallsign
	qsoColumnBand
	qsoColumnMode
	qsoColumnMyReport
	qsoColumnMyNumber
	qsoColumnMyXchange
	qsoColumnTheirReport
	qsoColumnTheirNumber
	qsoColumnTheirXchange
)

func setupQsoView(qsoView *gtk.TreeView) *gtk.ListStore {
	qsoView.AppendColumn(createColumn("UTC", qsoColumnUTC))
	qsoView.AppendColumn(createColumn("Callsign", qsoColumnCallsign))
	qsoView.AppendColumn(createColumn("Band", qsoColumnBand))
	qsoView.AppendColumn(createColumn("Mode", qsoColumnMode))
	qsoView.AppendColumn(createColumn("My RST", qsoColumnMyReport))
	qsoView.AppendColumn(createColumn("My #", qsoColumnMyNumber))
	qsoView.AppendColumn(createColumn("My XChg", qsoColumnMyXchange))
	qsoView.AppendColumn(createColumn("Th RST", qsoColumnTheirReport))
	qsoView.AppendColumn(createColumn("Th #", qsoColumnTheirNumber))
	qsoView.AppendColumn(createColumn("Th XChg", qsoColumnTheirXchange))

	qsoList, err := gtk.ListStoreNew(glib.TYPE_STRING, glib.TYPE_STRING, glib.TYPE_STRING, glib.TYPE_STRING, glib.TYPE_STRING, glib.TYPE_STRING, glib.TYPE_STRING, glib.TYPE_STRING, glib.TYPE_STRING, glib.TYPE_STRING)
	if err != nil {
		log.Fatalf("Cannot create QSO list store: %v", err)
	}
	qsoView.SetModel(qsoList)
	return qsoList
}

func createColumn(title string, id int) *gtk.TreeViewColumn {
	cellRenderer, err := gtk.CellRendererTextNew()
	if err != nil {
		log.Fatalf("Cannot create text cell renderer for column %s: %v", title, err)
	}

	column, err := gtk.TreeViewColumnNewWithAttribute(title, cellRenderer, "text", id)
	if err != nil {
		log.Fatalf("Cannot create column %s: %v", title, err)
	}
	return column
}

func (w *mainWindow) addEntryTraversal(entry *gtk.Entry) {
	entry.Connect("key_press_event", w.onEntryKeyPress)
	entry.Connect("focus_in_event", w.onEntryFocusIn)
	entry.Connect("focus_out_event", w.onEntryFocusOut)
}

func (w *mainWindow) onNew() {
	w.app.New()
}

func (w *mainWindow) onOpen() {
	w.app.Open()
}

func (w *mainWindow) onSaveAs() {
	w.app.SaveAs()
}

func (w *mainWindow) onQuit() {
	if app, err := w.window.GetApplication(); err != nil {
		log.Printf("Cannot quit application: %v", err)
	} else {
		app.Quit()
	}
}

func (w *mainWindow) onEntryKeyPress(widget interface{}, event *gdk.Event) bool {
	keyEvent := gdk.EventKeyNewFromEvent(event)
	switch keyEvent.KeyVal() {
	case gdk.KEY_Tab:
		w.entry.GotoNextField()
		return true
	case gdk.KEY_space:
		w.entry.GotoNextField()
		return true
	case gdk.KEY_Return:
		w.entry.Log()
		return true
	case gdk.KEY_Escape:
		w.entry.Reset()
		return true
	default:
		return false
	}
}

func (w *mainWindow) onEntryFocusIn(entry *gtk.Entry, event *gdk.Event) bool {
	entryField := w.entryToField(entry)
	w.entry.SetActiveField(entryField)
	return false
}

func (w *mainWindow) onEntryFocusOut(entry *gtk.Entry, event *gdk.Event) bool {
	entry.SelectRegion(0, 0)
	return false
}

func (w *mainWindow) addOtherWidgetTraversal(widget *gtk.Widget) {
	widget.Connect("key_press_event", w.onOtherWidgetKeyPress)
}

func (w *mainWindow) onOtherWidgetKeyPress(widget interface{}, event *gdk.Event) bool {
	keyEvent := gdk.EventKeyNewFromEvent(event)
	switch keyEvent.KeyVal() {
	case gdk.KEY_Tab:
		w.entry.SetActiveField(core.CallsignField)
		w.SetActiveField(core.CallsignField)
		return true
	case gdk.KEY_space:
		w.entry.SetActiveField(core.CallsignField)
		w.SetActiveField(core.CallsignField)
		return true
	default:
		return false
	}
}

func (w *mainWindow) onBandChanged(widget *gtk.ComboBoxText) bool {
	if w.entry != nil && !w.ignoreComboChange {
		w.entry.BandSelected(widget.GetActiveText())
	}
	return false
}

func (w *mainWindow) onModeChanged(widget *gtk.ComboBoxText) bool {
	if w.entry != nil && !w.ignoreComboChange {
		w.entry.ModeSelected(widget.GetActiveText())
	}
	return false
}

func (w *mainWindow) onLogButtonClicked(button *gtk.Button) bool {
	w.entry.Log()
	return true
}

func (w *mainWindow) onResetButtonClicked(button *gtk.Button) bool {
	w.entry.Reset()
	return true
}

func (w *mainWindow) Show() {
	w.window.ShowAll()
}

func (w *mainWindow) SetEntryController(entry core.EntryController) {
	w.entry = entry
}

func (w *mainWindow) GetCallsign() string {
	text, err := w.callsign.GetText()
	if err != nil {
		log.Printf("Error getting text: %v", err)
	}
	return text
}

func (w *mainWindow) SetCallsign(text string) {
	w.callsign.SetText(text)
}

func (w *mainWindow) GetTheirReport() string {
	text, err := w.theirReport.GetText()
	if err != nil {
		log.Printf("Error getting text: %v", err)
	}
	return text
}

func (w *mainWindow) SetTheirReport(text string) {
	w.theirReport.SetText(text)
}

func (w *mainWindow) GetTheirNumber() string {
	text, err := w.theirNumber.GetText()
	if err != nil {
		log.Printf("Error getting text: %v", err)
	}
	return text
}

func (w *mainWindow) SetTheirNumber(text string) {
	w.theirNumber.SetText(text)
}

func (w *mainWindow) GetTheirXchange() string {
	text, err := w.theirXchange.GetText()
	if err != nil {
		log.Printf("Error getting text: %v", err)
	}
	return text
}

func (w *mainWindow) SetTheirXchange(text string) {
	w.theirXchange.SetText(text)
}

func (w *mainWindow) GetBand() string {
	return w.band.GetActiveText()
}

func (w *mainWindow) SetBand(text string) {
	w.ignoreComboChange = true
	defer func() { w.ignoreComboChange = false }()
	w.band.SetActiveID(text)
}

func (w *mainWindow) GetMode() string {
	return w.mode.GetActiveText()
}

func (w *mainWindow) SetMode(text string) {
	w.ignoreComboChange = true
	defer func() { w.ignoreComboChange = false }()
	w.mode.SetActiveID(text)
}

func (w *mainWindow) GetMyReport() string {
	text, err := w.myReport.GetText()
	if err != nil {
		log.Printf("Error getting text: %v", err)
	}
	return text
}

func (w *mainWindow) SetMyReport(text string) {
	w.myReport.SetText(text)
}

func (w *mainWindow) GetMyNumber() string {
	text, err := w.myNumber.GetText()
	if err != nil {
		log.Printf("Error getting text: %v", err)
	}
	return text
}

func (w *mainWindow) SetMyNumber(text string) {
	w.myNumber.SetText(text)
}

func (w *mainWindow) GetMyXchange() string {
	text, err := w.myXchange.GetText()
	if err != nil {
		log.Printf("Error getting text: %v", err)
	}
	return text
}

func (w *mainWindow) SetMyXchange(text string) {
	w.myXchange.SetText(text)
}

func (w *mainWindow) SetActiveField(field core.EntryField) {
	entry := w.fieldToEntry(field)
	entry.GrabFocus()
}

func (w *mainWindow) fieldToEntry(field core.EntryField) *gtk.Entry {
	switch field {
	case core.CallsignField:
		return w.callsign
	case core.TheirReportField:
		return w.theirReport
	case core.TheirNumberField:
		return w.theirNumber
	case core.TheirXchangeField:
		return w.theirXchange
	case core.MyReportField:
		return w.myReport
	case core.MyNumberField:
		return w.myNumber
	case core.MyXchangeField:
		return w.myXchange
	case core.OtherField:
		return w.callsign
	default:
		log.Fatalf("Unknown entry field %d", field)
	}
	panic("this is never reached")
}

func (w *mainWindow) entryToField(entry *gtk.Entry) core.EntryField {
	name, _ := entry.GetName()
	switch name {
	case "callsignEntry":
		return core.CallsignField
	case "theirReportEntry":
		return core.TheirReportField
	case "theirNumberEntry":
		return core.TheirNumberField
	case "theirXchangeEntry":
		return core.TheirXchangeField
	case "myReportEntry":
		return core.MyReportField
	case "myNumberEntry":
		return core.MyNumberField
	case "myXchangeEntry":
		return core.MyXchangeField
	default:
		return core.OtherField
	}
}

func (w *mainWindow) SetDuplicateMarker(duplicate bool) {
	if duplicate {
		addStyleClass(&w.myNumber.Widget, "duplicate")
	} else {
		removeStyleClass(&w.myNumber.Widget, "duplicate")
	}
}

func (w *mainWindow) ShowMessage(err error) {
	w.messageLabel.SetText(err.Error())
}

func (w *mainWindow) ClearMessage() {
	w.messageLabel.SetText("")
}

func (w *mainWindow) SetLog(log core.Log) {
	w.log = log
}

func (w *mainWindow) UpdateAllRows(qsos []core.QSO) {
	w.qsoList.Clear()
	for _, qso := range qsos {
		w.RowAdded(qso)
	}
}

func (w *mainWindow) RowAdded(qso core.QSO) {
	newRow := w.qsoList.Append()
	err := w.qsoList.Set(newRow,
		[]int{
			qsoColumnUTC,
			qsoColumnCallsign,
			qsoColumnBand,
			qsoColumnMode,
			qsoColumnMyReport,
			qsoColumnMyNumber,
			qsoColumnMyXchange,
			qsoColumnTheirReport,
			qsoColumnTheirNumber,
			qsoColumnTheirXchange,
		},
		[]interface{}{
			qso.Time.In(time.UTC).Format("15:04"),
			qso.Callsign.String(),
			qso.Band.String(),
			qso.Mode.String(),
			qso.MyReport.String(),
			qso.MyNumber.String(),
			qso.MyXchange,
			qso.TheirReport.String(),
			qso.TheirNumber.String(),
			qso.TheirXchange,
		})
	if err != nil {
		log.Printf("Cannot add QSO row %s: %v", qso.String(), err)
	}
	path, err := w.qsoList.GetPath(newRow)
	if err != nil {
		log.Printf("Cannot get path for list item: %s", err)
	}
	w.qsoView.SetCursorOnCell(path, w.qsoView.GetColumn(1), nil, false)
}

func (w *mainWindow) SetAppController(app core.AppController) {
	w.app = app
}

func (w *mainWindow) ShowFilename(filename string) {
	w.window.SetTitle(fmt.Sprintf("Hello Contest %s", filepath.Base(filename)))
}

func (w *mainWindow) SelectOpenFile(title string, patterns ...string) (string, bool, error) {
	dlg, err := gtk.FileChooserDialogNewWith1Button(title, &w.window.Window, gtk.FILE_CHOOSER_ACTION_OPEN, "Open", gtk.RESPONSE_ACCEPT)
	if err != nil {
		errors.Wrap(err, "cannot create a file selection dialog to open a file")
	}
	defer dlg.Destroy()

	if len(patterns) > 0 {
		filter, err := gtk.FileFilterNew()
		if err != nil {
			return "", false, errors.Wrap(err, "cannot create a file selection dialog to open a file")
		}
		for _, pattern := range patterns {
			filter.AddPattern(pattern)
		}
		dlg.SetFilter(filter)
	}

	result := dlg.Run()
	if result != int(gtk.RESPONSE_ACCEPT) {
		return "", false, nil
	}

	return dlg.GetFilename(), true, nil
}

func (w *mainWindow) SelectSaveFile(title string, patterns ...string) (string, bool, error) {
	dlg, err := gtk.FileChooserDialogNewWith1Button(title, &w.window.Window, gtk.FILE_CHOOSER_ACTION_SAVE, "Save", gtk.RESPONSE_ACCEPT)
	if err != nil {
		return "", false, errors.Wrap(err, "cannot create a file selection dialog to save a file")
	}
	defer dlg.Destroy()

	dlg.SetDoOverwriteConfirmation(true)

	if len(patterns) > 0 {
		filter, err := gtk.FileFilterNew()
		if err != nil {
			return "", false, errors.Wrap(err, "cannot create a file selection dialog to save a file")
		}
		for _, pattern := range patterns {
			filter.AddPattern(pattern)
		}
		dlg.SetFilter(filter)
	}

	result := dlg.Run()
	if result != int(gtk.RESPONSE_ACCEPT) {
		return "", false, nil
	}

	return dlg.GetFilename(), true, nil
}

func (w *mainWindow) ShowInfoDialog(format string, a ...interface{}) {
	dlg := gtk.MessageDialogNew(w.window, gtk.DIALOG_MODAL, gtk.MESSAGE_INFO, gtk.BUTTONS_OK, format, a...)
	defer dlg.Destroy()
	dlg.Run()
}

func (w *mainWindow) ShowErrorDialog(format string, a ...interface{}) {
	dlg := gtk.MessageDialogNew(w.window, gtk.DIALOG_MODAL, gtk.MESSAGE_ERROR, gtk.BUTTONS_OK, format, a...)
	defer dlg.Destroy()
	dlg.Run()
}
