package logbook

import (
	"log"
	"math"
	"sort"

	"github.com/ftl/hamradio/callsign"
	"github.com/pkg/errors"

	"github.com/ftl/hellocontest/core"
)

// New creates a new empty logbook.
func New(clock core.Clock) *Logbook {
	return &Logbook{
		clock:             clock,
		qsos:              make([]core.QSO, 0, 1000),
		view:              &nullView{},
		rowAddedListeners: make([]RowAddedListener, 0),
	}
}

// Load creates a new log and loads it with the entries from the given reader.
func Load(clock core.Clock, reader Reader) (*Logbook, error) {
	log.Print("Loading QSOs")
	logbook := &Logbook{
		clock:             clock,
		view:              &nullView{},
		rowAddedListeners: make([]RowAddedListener, 0),
	}

	var err error
	logbook.qsos, err = reader.ReadAll()
	if err != nil {
		return nil, err
	}

	logbook.myLastNumber = lastNumber(logbook.qsos)
	return logbook, nil
}

func lastNumber(qsos []core.QSO) int {
	lastNumber := 0
	for _, qso := range qsos {
		lastNumber = int(math.Max(float64(lastNumber), float64(qso.MyNumber)))
	}
	return lastNumber
}

type Logbook struct {
	clock           core.Clock
	qsos            []core.QSO
	myLastNumber    int
	ignoreSelection bool

	view                 View
	rowAddedListeners    []RowAddedListener
	rowSelectedListeners []RowSelectedListener
}

// View represents the visual part of the log.
type View interface {
	UpdateAllRows([]core.QSO)
	RowAdded(core.QSO)
	SelectRow(int)
}

// Reader reads log entries.
type Reader interface {
	ReadAll() ([]core.QSO, error)
}

// Writer writes log entries.
type Writer interface {
	Write(core.QSO) error
}

// Store allows to read and write log entries.
type Store interface {
	Reader
	Writer
	Clear() error
}

// RowAddedListener is notified when a new row is added to the log.
type RowAddedListener func(core.QSO) error

func (l RowAddedListener) Write(qso core.QSO) error {
	return l(qso)
}

// RowSelectedListener is notified when a row is selected in the log view.
type RowSelectedListener func(core.QSO)

func (l *Logbook) SetView(view View) {
	l.ignoreSelection = true

	if view == nil {
		l.view = &nullView{}
		return
	}

	l.view = view
	l.view.UpdateAllRows(l.qsos)
	l.ignoreSelection = false
}

func (l *Logbook) OnRowAdded(listener RowAddedListener) {
	l.rowAddedListeners = append(l.rowAddedListeners, listener)
}

func (l *Logbook) ClearRowAddedListeners() {
	l.rowAddedListeners = make([]RowAddedListener, 0)
}

func (l *Logbook) emitRowAdded(qso core.QSO) {
	for _, listener := range l.rowAddedListeners {
		err := listener(qso)
		if err != nil {
			log.Printf("Error on rowAdded: %T, %v", listener, err)
		}
	}
}

func (l *Logbook) OnRowSelected(listener RowSelectedListener) {
	l.rowSelectedListeners = append(l.rowSelectedListeners, listener)
}

func (l *Logbook) ClearRowSelectedListeners() {
	l.rowSelectedListeners = make([]RowSelectedListener, 0)
}

func (l *Logbook) emitRowSelected(qso core.QSO) {
	for _, listener := range l.rowSelectedListeners {
		listener(qso)
	}
}

func (l *Logbook) Select(i int) {
	if i < 0 || i >= len(l.qsos) {
		log.Printf("invalid QSO index %d", i)
		return
	}
	if l.ignoreSelection {
		return
	}
	qso := l.qsos[i]
	l.emitRowSelected(qso)
}

func (l *Logbook) SelectQSO(qso core.QSO) {
	log.Printf("select qso #%d", qso.MyNumber)
	index, ok := l.indexOf(qso)
	if !ok {
		log.Print("qso not found")
		return
	}

	l.view.SelectRow(index)
}

func (l *Logbook) indexOf(qso core.QSO) (int, bool) {
	for i := len(l.qsos) - 1; i >= 0; i-- {
		if l.qsos[i].MyNumber == qso.MyNumber {
			return i, true
		}
	}
	return -1, false
}

func (l *Logbook) SelectLastQSO() {
	if len(l.qsos) == 0 {
		return
	}
	l.view.SelectRow(len(l.qsos) - 1)
}

func (l *Logbook) NextNumber() core.QSONumber {
	return core.QSONumber(l.myLastNumber + 1)
}

func (l *Logbook) LastBand() core.Band {
	if len(l.qsos) == 0 {
		return core.NoBand
	}
	return l.qsos[len(l.qsos)-1].Band
}

func (l *Logbook) LastMode() core.Mode {
	if len(l.qsos) == 0 {
		return core.NoMode
	}
	return l.qsos[len(l.qsos)-1].Mode
}

func (l *Logbook) Log(qso core.QSO) {
	l.ignoreSelection = true
	defer func() { l.ignoreSelection = false }()

	qso.LogTimestamp = l.clock.Now()
	l.qsos = append(l.qsos, qso)
	l.myLastNumber = int(math.Max(float64(l.myLastNumber), float64(qso.MyNumber)))
	l.view.RowAdded(qso)
	l.emitRowAdded(qso)
	log.Printf("QSO added: %s", qso.String())
}

func (l *Logbook) Find(callsign callsign.Callsign) (core.QSO, bool) {
	checkedNumbers := make(map[core.QSONumber]bool)
	for i := len(l.qsos) - 1; i >= 0; i-- {
		qso := l.qsos[i]
		if checkedNumbers[qso.MyNumber] {
			continue
		}
		checkedNumbers[qso.MyNumber] = true

		if callsign == qso.Callsign {
			return qso, true
		}
	}
	return core.QSO{}, false
}

func (l *Logbook) FindAll(callsign callsign.Callsign, band core.Band, mode core.Mode) []core.QSO {
	checkedNumbers := make(map[core.QSONumber]bool)
	result := make([]core.QSO, 0)
	for i := len(l.qsos) - 1; i >= 0; i-- {
		qso := l.qsos[i]
		if checkedNumbers[qso.MyNumber] {
			continue
		}
		checkedNumbers[qso.MyNumber] = true

		if callsign != qso.Callsign {
			continue
		}
		if band != core.NoBand && band != qso.Band {
			continue
		}
		if mode != core.NoMode && mode != qso.Mode {
			continue
		}
		result = append(result, qso)
	}
	return result
}

func (l *Logbook) QsosOrderedByMyNumber() []core.QSO {
	return byMyNumber(l.qsos)
}

func (l *Logbook) UniqueQsosOrderedByMyNumber() []core.QSO {
	return byMyNumber(unique(l.qsos))
}

func byMyNumber(qsos []core.QSO) []core.QSO {
	result := make([]core.QSO, len(qsos))
	copy(result, qsos)
	sort.Slice(result, func(i, j int) bool {
		if result[i].MyNumber == result[j].MyNumber {
			return result[i].LogTimestamp.Before(result[j].LogTimestamp)
		}
		return result[i].MyNumber < result[j].MyNumber
	})
	return result
}

func unique(qsos []core.QSO) []core.QSO {
	index := make(map[core.QSONumber]core.QSO)
	for _, qso := range qsos {
		former, ok := index[qso.MyNumber]
		if !ok || qso.LogTimestamp.After(former.LogTimestamp) {
			index[qso.MyNumber] = qso
		}
	}

	result := make([]core.QSO, len(index))
	i := 0
	for _, qso := range index {
		result[i] = qso
		i++
	}
	return result
}

func (l *Logbook) WriteAll(writer Writer) error {
	for _, qso := range l.qsos {
		err := writer.Write(qso)
		if err != nil {
			return errors.Wrapf(err, "cannot write QSO %v", qso)
		}
	}
	return nil
}

type nullView struct{}

func (d *nullView) UpdateAllRows([]core.QSO) {}
func (d *nullView) RowAdded(core.QSO)        {}
func (d *nullView) SelectRow(int)            {}
