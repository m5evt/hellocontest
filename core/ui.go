package core

// LogView represents the visual part of the log.
type LogView interface {
	SetLog(Log)

	UpdateAllRows([]QSO)
	RowAdded(QSO)
}

// EntryController controls the entry of QSO data.
type EntryController interface {
	SetView(EntryView)

	GotoNextField() EntryField
	GetActiveField() EntryField
	SetActiveField(EntryField)

	BandSelected(string)
	ModeSelected(string)

	Log()
	Reset()
}

// EntryView represents the visual part of the QSO data entry.
type EntryView interface {
	SetEntryController(EntryController)

	GetCallsign() string
	SetCallsign(string)
	GetTheirReport() string
	SetTheirReport(string)
	GetTheirNumber() string
	SetTheirNumber(string)
	GetBand() string
	SetBand(text string)
	GetMode() string
	SetMode(text string)
	GetMyReport() string
	SetMyReport(string)
	GetMyNumber() string
	SetMyNumber(string)

	SetActiveField(EntryField)
	SetDuplicateMarker(bool)
	ShowError(error)
	ClearError()
}

// EntryField represents an entry field in the visual part.
type EntryField int

// The entry fields.
const (
	CallsignField EntryField = iota
	TheirReportField
	TheirNumberField
	MyReportField
	MyNumberField
	OtherField
)
