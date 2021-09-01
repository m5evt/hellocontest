package entry

import (
	"testing"
	"time"

	"github.com/ftl/hamradio/callsign"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/ftl/hellocontest/core"
	"github.com/ftl/hellocontest/core/clock"
	"github.com/ftl/hellocontest/core/mocked"
)

func TestEntryController_Clear(t *testing.T) {
	_, log, qsoList, _, controller, _ := setupEntryTest()
	log.Activate()
	log.On("NextNumber").Return(core.QSONumber(1)).Once()
	qsoList.Activate()
	qsoList.On("SelectLastQSO").Once()

	controller.Clear()

	assert.Equal(t, controller.input.myReport, "599")
	assert.Equal(t, controller.input.myNumber, "001")
	assert.Equal(t, controller.input.myXchange, "")
	assert.Equal(t, controller.input.callsign, "")
	assert.Equal(t, controller.input.theirNumber, "")
	assert.Equal(t, controller.input.theirXchange, "")
	assert.Equal(t, controller.input.band, "160m")
	assert.Equal(t, controller.input.mode, "CW")
}

func TestEntryController_ClearView(t *testing.T) {
	_, log, qsoList, view, controller, _ := setupEntryTest()
	log.Activate()
	log.On("NextNumber").Once().Return(core.QSONumber(1))
	qsoList.Activate()
	qsoList.On("SelectLastQSO").Once()

	view.Activate()
	view.On("SetMyReport", "599").Once()
	view.On("SetMyNumber", "001").Once()
	view.On("SetMyXchange", "").Once()
	view.On("SetFrequency", mock.Anything).Once()
	view.On("SetCallsign", "").Once()
	view.On("SetTheirReport", "599").Once()
	view.On("SetTheirXchange", "").Once()
	view.On("SetBand", "160m").Once()
	view.On("SetMode", "CW").Once()
	view.On("SetActiveField", core.CallsignField).Once()
	view.On("SetDuplicateMarker", false).Once()
	view.On("SetEditingMarker", false).Once()
	view.On("ClearMessage").Once()

	controller.Clear()

	view.AssertExpectations(t)
}

func TestEntryController_SetLastSelectedBandAndModeOnClear(t *testing.T) {
	_, log, qsoList, _, controller, _ := setupEntryTest()
	log.Activate()
	log.On("NextNumber").Once().Return(core.QSONumber(1))
	qsoList.Activate()
	qsoList.On("SelectLastQSO").Once()

	controller.SetActiveField(core.BandField)
	controller.Enter("30m")
	controller.SetActiveField(core.ModeField)
	controller.Enter("RTTY")
	controller.Clear()

	assert.Equal(t, "30m", controller.input.band)
	assert.Equal(t, core.Band30m, controller.selectedBand)
	assert.Equal(t, "RTTY", controller.input.mode)
	assert.Equal(t, core.ModeRTTY, controller.selectedMode)
}

func TestEntryController_GotoNextField(t *testing.T) {
	_, _, _, view, controller, config := setupEntryTest()

	assert.Equal(t, core.CallsignField, controller.activeField, "callsign should be active at start")

	testCases := []struct {
		enterTheirNumber, enterTheirXchange bool
		active, next                        core.EntryField
	}{
		{true, true, core.CallsignField, core.TheirReportField},
		{false, true, core.TheirReportField, core.TheirXchangeField},
		{false, false, core.TheirReportField, core.CallsignField},
		{true, true, core.TheirXchangeField, core.CallsignField},
		{true, true, core.MyReportField, core.CallsignField},
		{true, true, core.MyNumberField, core.CallsignField},
		{true, true, core.OtherField, core.CallsignField},
	}
	view.Activate()
	view.On("Callsign").Return("").Maybe()
	view.On("EnableExchangeFields", mock.Anything, mock.Anything).Times(len(testCases))
	view.On("SetActiveField", mock.Anything).Times(len(testCases))
	for _, tc := range testCases {
		//	config.enterTheirNumber = tc.enterTheirNumber
		config.enterTheirXchange = tc.enterTheirXchange
		controller.ContestChanged(config.Contest())
		controller.SetActiveField(tc.active)
		actual := controller.GotoNextField()
		assert.Equal(t, tc.next, actual)
		assert.Equal(t, tc.next, controller.activeField)
	}

	view.AssertExpectations(t)
}

func TestEntryController_EnterNewCallsign(t *testing.T) {
	_, _, qsoList, view, controller, _ := setupEntryTest()
	qsoList.Activate()
	qsoList.On("FindDuplicateQSOs", mock.Anything, mock.Anything, mock.Anything).Return([]core.QSO{})

	view.Activate()
	view.On("SetDuplicateMarker", false).Once()
	view.On("ClearMessage").Once()
	view.On("SetActiveField", core.TheirReportField).Once()

	controller.Enter("DL1ABC")
	controller.GotoNextField()

	assert.Equal(t, "DL1ABC", controller.input.callsign)

	qsoList.AssertExpectations(t)
	view.AssertExpectations(t)
}

func TestEntryController_EnterDuplicateCallsign(t *testing.T) {
	_, _, qsoList, view, controller, _ := setupEntryTest()

	dl1abc, _ := callsign.Parse("DL1ABC")
	qso := core.QSO{
		Callsign:    dl1abc,
		Band:        core.Band20m,
		Mode:        core.ModeSSB,
		TheirReport: core.RST("599"),
		MyReport:    core.RST("559"),
		MyNumber:    1,
	}

	qsoList.Activate()
	qsoList.On("FindDuplicateQSOs", dl1abc, core.Band160m, core.ModeCW).Return([]core.QSO{qso}).Twice()

	view.Activate()
	view.On("SetDuplicateMarker", true).Once()
	view.On("ShowMessage", mock.Anything).Once()
	view.On("SetActiveField", core.TheirReportField).Once()

	controller.Enter("DL1ABC")
	controller.GotoNextField()

	assert.False(t, controller.editing)
	qsoList.AssertExpectations(t)
	view.AssertExpectations(t)
}

func TestEntryController_EnterFrequency(t *testing.T) {
	_, _, _, view, controller, _ := setupEntryTest()

	view.Activate()
	view.On("SetCallsign", "").Once()
	view.On("SetFrequency", core.Frequency(7028000)).Once()

	controller.Enter("7028")
	controller.Log()

	assert.Equal(t, "", controller.input.callsign)

	view.AssertExpectations(t)
}

func TestEntryController_LogNewQSO(t *testing.T) {
	clock, log, qsoList, _, controller, _ := setupEntryTest()

	dl1abc, _ := callsign.Parse("DL1ABC")
	qso := core.QSO{
		Callsign:     dl1abc,
		Time:         clock.Now(),
		Band:         core.Band40m,
		Mode:         core.ModeCW,
		TheirReport:  core.RST("559"),
		TheirXchange: "012",
		MyReport:     core.RST("579"),
		MyNumber:     1,
		MyXchange:    "myx",
	}

	log.Activate()
	log.On("NextNumber").Return(core.QSONumber(1))
	log.On("Log", qso).Once()
	qsoList.Activate()
	qsoList.On("FindDuplicateQSOs", dl1abc, mock.Anything, mock.Anything).Return([]core.QSO{})
	qsoList.On("SelectLastQSO").Twice()

	controller.Clear()
	controller.SetActiveField(core.BandField)
	controller.Enter("40m")
	controller.SetActiveField(core.ModeField)
	controller.Enter("CW")
	controller.SetActiveField(core.MyReportField)
	controller.Enter("579")
	controller.SetActiveField(core.MyXchangeField)
	controller.Enter("myx")
	controller.GotoNextField()

	controller.Enter("DL1ABC")
	controller.GotoNextField()
	controller.Enter("559")
	controller.GotoNextField()
	controller.Enter("012")

	controller.Log()

	log.AssertExpectations(t)
	qsoList.AssertExpectations(t)
	assert.Equal(t, core.CallsignField, controller.activeField)
}

func TestEntryController_LogWithWrongCallsign(t *testing.T) {
	_, log, _, view, controller, _ := setupEntryTest()
	log.Activate()

	view.Activate()
	view.On("SetActiveField", core.CallsignField).Once()
	view.On("ShowMessage", mock.Anything).Once()

	controller.Enter("DL")
	controller.Log()

	view.AssertExpectations(t)
	log.AssertNotCalled(t, "Log", mock.Anything)
	assert.Equal(t, core.CallsignField, controller.activeField)
}

func TestEntryController_LogWithInvalidTheirReport(t *testing.T) {
	_, log, _, view, controller, _ := setupEntryTest()

	controller.SetActiveField(core.BandField)
	controller.Enter("40m")
	controller.SetActiveField(core.ModeField)
	controller.Enter("CW")
	controller.GotoNextField()
	controller.Enter("DL1ABC")
	controller.GotoNextField()
	controller.Enter("000")

	log.Activate()
	view.Activate()
	view.On("SetActiveField", core.TheirReportField).Once()
	view.On("ShowMessage", mock.Anything).Once()

	controller.Log()

	view.AssertExpectations(t)
	log.AssertNotCalled(t, "Log", mock.Anything)
	assert.Equal(t, core.TheirReportField, controller.activeField)
}

func TestEntryController_LogWithoutMandatoryTheirXchange(t *testing.T) {
	_, log, _, view, controller := setupEntryWithOnlyExchangeTest()

	controller.SetActiveField(core.BandField)
	controller.Enter("40m")
	controller.SetActiveField(core.ModeField)
	controller.Enter("CW")
	controller.GotoNextField()
	controller.Enter("DL1ABC")
	controller.GotoNextField()
	controller.Enter("559")

	log.Activate()
	view.Activate()
	view.On("SetActiveField", core.TheirXchangeField).Once()
	view.On("ShowMessage", mock.Anything).Once()

	controller.Log()

	view.AssertExpectations(t)
	log.AssertNotCalled(t, "Log", mock.Anything)
	assert.Equal(t, core.TheirXchangeField, controller.activeField)
}

func TestEntryController_LogWithInvalidMyReport(t *testing.T) {
	_, log, _, view, controller, _ := setupEntryTest()

	controller.SetActiveField(core.BandField)
	controller.Enter("40m")
	controller.SetActiveField(core.ModeField)
	controller.Enter("CW")
	controller.SetActiveField(core.MyReportField)
	controller.Enter("000")
	controller.GotoNextField()
	controller.Enter("DL1ABC")
	controller.GotoNextField()
	controller.Enter("559")
	controller.GotoNextField()
	controller.Enter("1")

	log.Activate()
	view.Activate()
	view.On("SetActiveField", core.MyReportField).Once()
	view.On("ShowMessage", mock.Anything).Once()

	controller.Log()

	view.AssertExpectations(t)
	log.AssertNotCalled(t, "Log", mock.Anything)
	assert.Equal(t, core.MyReportField, controller.activeField)
}

func TestEntryController_EnterCallsignCheckForDuplicateAndShowMessage(t *testing.T) {
	clock, _, qsoList, view, controller, _ := setupEntryTest()
	qsoList.Activate()
	view.Activate()

	dl1ab, _ := callsign.Parse("DL1AB")
	dl1abc, _ := callsign.Parse("DL1ABC")
	qso := core.QSO{
		Callsign:    dl1ab,
		Time:        clock.Now(),
		TheirReport: core.RST("559"),
		MyReport:    core.RST("579"),
		MyNumber:    12,
	}

	qsoList.On("FindDuplicateQSOs", dl1ab, mock.Anything, mock.Anything).Once().Return([]core.QSO{qso})
	view.On("ShowMessage", mock.Anything).Once()
	controller.Enter("DL1AB")
	view.AssertExpectations(t)

	qsoList.On("FindDuplicateQSOs", dl1abc, mock.Anything, mock.Anything).Once().Return([]core.QSO{})
	view.On("ClearMessage").Once()
	controller.Enter("DL1ABC")
	view.AssertExpectations(t)
}

func TestEntryController_LogDuplicateQSO(t *testing.T) {
	clock, log, qsoList, _, controller, _ := setupEntryTest()

	dl1abc, _ := callsign.Parse("DL1ABC")
	qso := core.QSO{
		Callsign:     dl1abc,
		Time:         clock.Now().Add(-1 * time.Minute),
		Band:         core.Band40m,
		Mode:         core.ModeCW,
		TheirReport:  core.RST("559"),
		TheirXchange: "012",
		MyReport:     core.RST("579"),
		MyNumber:     1,
		MyXchange:    "myx",
	}
	dupe := core.QSO{
		Callsign:     dl1abc,
		Time:         clock.Now(),
		Band:         core.Band40m,
		Mode:         core.ModeCW,
		TheirReport:  core.RST("569"),
		TheirXchange: "012",
		MyReport:     core.RST("579"),
		MyNumber:     2,
		MyXchange:    "myx",
	}

	log.Activate()
	log.On("NextNumber").Return(core.QSONumber(2))
	log.On("Log", dupe).Once()
	qsoList.Activate()
	qsoList.On("FindDuplicateQSOs", dl1abc, mock.Anything, mock.Anything).Return([]core.QSO{qso})
	qsoList.On("SelectLastQSO").Twice()

	controller.Clear()
	controller.SetActiveField(core.BandField)
	controller.Enter("40m")
	controller.SetActiveField(core.ModeField)
	controller.Enter("CW")
	controller.SetActiveField(core.MyReportField)
	controller.Enter("579")
	controller.SetActiveField(core.MyXchangeField)
	controller.Enter("myx")
	controller.GotoNextField()

	controller.Enter("DL1ABC")
	controller.GotoNextField()
	controller.Enter("569")
	controller.GotoNextField()
	controller.Enter("012")

	controller.Log()

	log.AssertExpectations(t)
	qsoList.AssertExpectations(t)
	assert.Equal(t, core.CallsignField, controller.activeField)
}

func TestEntryController_SelectRowForEditing(t *testing.T) {
	clock, _, _, view, controller, _ := setupEntryTest()
	view.Activate()

	dl1abc, _ := callsign.Parse("DL1ABC")
	qso := core.QSO{
		Band:         core.Band80m,
		Mode:         core.ModeCW,
		Callsign:     dl1abc,
		Time:         clock.Now(),
		TheirReport:  core.RST("559"),
		TheirXchange: "A01",
		MyReport:     core.RST("579"),
		MyNumber:     34,
		MyXchange:    "B36",
	}

	view.On("SetBand", "80m").Once()
	view.On("SetMode", "CW").Once()
	view.On("SetCallsign", "DL1ABC").Once()
	view.On("SetTheirReport", "559").Once()
	view.On("SetTheirXchange", "A01").Once()
	view.On("SetMyReport", "579").Once()
	view.On("SetMyNumber", "034").Once()
	view.On("SetMyXchange", "B36").Once()
	view.On("SetActiveField", core.CallsignField).Once()
	view.On("SetEditingMarker", true).Once()

	controller.QSOSelected(qso)

	assertQSOInput(t, qso, controller)

	view.AssertExpectations(t)
}

func TestEntryController_EditQSO(t *testing.T) {
	clock, log, _, _, controller, _ := setupEntryTest()

	dl1abc, _ := callsign.Parse("DL1ABC")
	dl2abc, _ := callsign.Parse("DL2ABC")
	qso := core.QSO{
		Band:         core.Band80m,
		Mode:         core.ModeCW,
		Callsign:     dl1abc,
		Time:         clock.Now(),
		TheirReport:  core.RST("559"),
		TheirXchange: "A01",
		MyReport:     core.RST("579"),
		MyNumber:     34,
		MyXchange:    "B36",
	}
	changedQSO := qso
	changedQSO.Callsign = dl2abc
	changedQSO.TheirXchange = "B02"

	controller.QSOSelected(qso)
	controller.SetActiveField(core.CallsignField)
	controller.Enter("DL2ABC")
	controller.SetActiveField(core.TheirXchangeField)
	controller.Enter("B02")

	log.Activate()
	log.On("Log", changedQSO).Once()
	log.On("NextNumber").Return(core.QSONumber(35))
	controller.Log()

	log.AssertExpectations(t)
}

// Helpers

func setupEntryTest() (core.Clock, *mocked.Log, *mocked.QSOList, *mocked.EntryView, *Controller, *testSettings) {
	now := time.Date(2006, time.January, 2, 15, 4, 5, 6, time.UTC)
	clock := clock.Static(now)
	log := new(mocked.Log)
	qsoList := new(mocked.QSOList)
	view := new(mocked.EntryView)
	settings := &testSettings{myCall: "DL0ABC", enterTheirNumber: true, enterTheirXchange: true}
	controller := NewController(settings, clock, qsoList, testIgnoreAsync)
	controller.SetLogbook(log)
	controller.SetView(view)

	return clock, log, qsoList, view, controller, settings
}

func setupEntryWithOnlyNumberTest() (core.Clock, *mocked.Log, *mocked.QSOList, *mocked.EntryView, *Controller) {
	now := time.Date(2006, time.January, 2, 15, 4, 5, 6, time.UTC)
	clock := clock.Static(now)
	log := new(mocked.Log)
	qsoList := new(mocked.QSOList)
	view := new(mocked.EntryView)
	settings := &testSettings{myCall: "DL0ABC", enterTheirNumber: true, enterTheirXchange: false}
	controller := NewController(settings, clock, qsoList, testIgnoreAsync)
	controller.SetLogbook(log)
	controller.SetView(view)

	return clock, log, qsoList, view, controller
}

func setupEntryWithOnlyExchangeTest() (core.Clock, *mocked.Log, *mocked.QSOList, *mocked.EntryView, *Controller) {
	now := time.Date(2006, time.January, 2, 15, 4, 5, 6, time.UTC)
	clock := clock.Static(now)
	log := new(mocked.Log)
	qsoList := new(mocked.QSOList)
	view := new(mocked.EntryView)
	settings := &testSettings{myCall: "DL0ABC", enterTheirNumber: false, enterTheirXchange: true, requireTheirXchange: true}
	controller := NewController(settings, clock, qsoList, testIgnoreAsync)
	controller.SetLogbook(log)
	controller.SetView(view)

	return clock, log, qsoList, view, controller
}

func assertQSOInput(t *testing.T, qso core.QSO, controller *Controller) {
	assert.Equal(t, qso.Callsign.String(), controller.input.callsign, "callsign")
	assert.Equal(t, qso.TheirNumber.String(), controller.input.theirNumber, "their number")
	assert.Equal(t, qso.TheirXchange, controller.input.theirXchange, "their Xchange")
	assert.Equal(t, qso.MyReport.String(), controller.input.myReport, "my report")
	assert.Equal(t, qso.MyNumber.String(), controller.input.myNumber, "my number")
	assert.Equal(t, qso.MyXchange, controller.input.myXchange, "my Xchange")
	assert.Equal(t, qso.Band.String(), controller.input.band, "input band")
	assert.Equal(t, qso.Band, controller.selectedBand, "selected band")
	assert.Equal(t, qso.Mode.String(), controller.input.mode, "input mode")
	assert.Equal(t, qso.Mode, controller.selectedMode, "selected mode")
}

type testSettings struct {
	myCall              string
	enterTheirNumber    bool
	enterTheirXchange   bool
	requireTheirXchange bool
}

func (s *testSettings) Station() core.Station {
	return core.Station{
		Callsign: callsign.MustParse(s.myCall),
	}
}

func (s *testSettings) Contest() core.Contest {
	return core.Contest{
		EnterTheirNumber:    s.enterTheirNumber,
		EnterTheirXchange:   s.enterTheirXchange,
		RequireTheirXchange: s.requireTheirXchange,
	}
}

func testIgnoreAsync(f func()) {}
