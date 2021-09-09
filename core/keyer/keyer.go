package keyer

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"text/template"

	"github.com/ftl/hamradio/callsign"

	"github.com/ftl/hellocontest/core"
)

// View represents the visual parts of the keyer.
type View interface {
	ShowMessage(...interface{})
	SetPattern(int, string)
	ShowKeyerSpeed(int)
}

// CWClient defines the interface used by the Keyer to output the CW.
type CWClient interface {
	Connect() error
	IsConnected() bool
	Speed(int)
	Send(text string)
	Abort()
}

// KeyerValueProvider provides the variable values for the Keyer templates on demand.
type KeyerValueProvider func() core.KeyerValues

type KeyerListener interface {
	KeyerChanged(core.Keyer)
}

type KeyerListenerFunc func(core.Keyer)

func (f KeyerListenerFunc) KeyerChanged(keyer core.Keyer) {
	f(keyer)
}

type Writer interface {
	WriteKeyer(core.Keyer) error
}

// New returns a new Keyer that has no patterns or templates defined yet.
func New(settings core.Settings, client CWClient, keyer core.Keyer) *Keyer {
	result := &Keyer{
		writer:          new(nullWriter),
		stationCallsign: settings.Station().Callsign,
		spPatterns:      make(map[int]string),
		spTemplates:     make(map[int]*template.Template),
		runPatterns:     make(map[int]string),
		runTemplates:    make(map[int]*template.Template),
		client:          client,
		values:          noValues,
	}
	result.setWorkmode(core.Run)
	result.SetKeyer(keyer)
	if result.client == nil {
		result.client = new(nullClient)
	}
	return result
}

type Keyer struct {
	writer     Writer
	view       View
	client     CWClient
	values     KeyerValueProvider
	savedKeyer core.Keyer

	listeners []interface{}

	stationCallsign callsign.Callsign
	wpm             int
	spPatterns      map[int]string
	spTemplates     map[int]*template.Template
	runPatterns     map[int]string
	runTemplates    map[int]*template.Template
	patterns        *map[int]string
	templates       *map[int]*template.Template
}

func (k *Keyer) setWorkmode(workmode core.Workmode) {
	switch workmode {
	case core.SearchPounce:
		k.patterns = &k.spPatterns
		k.templates = &k.spTemplates
	case core.Run:
		k.patterns = &k.runPatterns
		k.templates = &k.runTemplates
	}
}

func (k *Keyer) SetWriter(writer Writer) {
	if writer == nil {
		k.writer = new(nullWriter)
		return
	}
	k.writer = writer
}

func (k *Keyer) SetKeyer(keyer core.Keyer) {
	k.savedKeyer = keyer
	k.wpm = keyer.WPM
	for i, pattern := range keyer.SPMacros {
		k.spPatterns[i] = pattern
		k.spTemplates[i], _ = template.New("").Parse(pattern)
	}
	for i, pattern := range keyer.RunMacros {
		k.runPatterns[i] = pattern
		k.runTemplates[i], _ = template.New("").Parse(pattern)
	}
	k.showPatterns()
}

func (k *Keyer) SetView(view View) {
	k.view = view
	k.showPatterns()
}

func (k *Keyer) showPatterns() {
	if k.view == nil {
		return
	}
	for i, pattern := range *k.patterns {
		k.view.SetPattern(i, pattern)
	}
}

func (k *Keyer) WorkmodeChanged(workmode core.Workmode) {
	k.setWorkmode(workmode)
	k.showPatterns()
}

func (k *Keyer) StationChanged(station core.Station) {
	k.stationCallsign = station.Callsign
}

func (k *Keyer) SetValues(values KeyerValueProvider) {
	k.values = values
}

func (k *Keyer) Save() {
	keyer, modified := k.getKeyerSettings()
	if !modified {
		return
	}
	k.savedKeyer = keyer
	k.writer.WriteKeyer(keyer)
}

func (k *Keyer) KeyerSettings() core.Keyer {
	keyer, _ := k.getKeyerSettings()
	return keyer
}

func (k *Keyer) getKeyerSettings() (core.Keyer, bool) {
	var keyer core.Keyer
	keyer.WPM = k.wpm
	keyer.SPMacros = make([]string, len(k.spPatterns))
	for i := range keyer.SPMacros {
		pattern, ok := k.spPatterns[i]
		if !ok {
			continue
		}
		keyer.SPMacros[i] = pattern
	}
	keyer.RunMacros = make([]string, len(k.runPatterns))
	for i := range keyer.RunMacros {
		pattern, ok := k.runPatterns[i]
		if !ok {
			continue
		}
		keyer.RunMacros[i] = pattern
	}

	modified := (fmt.Sprintf("%v", keyer) != fmt.Sprintf("%v", k.savedKeyer))
	return keyer, modified
}

func (k *Keyer) EnterSpeed(speed int) {
	k.wpm = speed
	if !k.client.IsConnected() {
		return
	}
	log.Printf("speed entered: %d", speed)
	k.client.Speed(k.wpm)
}

func (k *Keyer) IncreaseSpeed() {
	k.EnterSpeed(k.wpm + 1)
	k.view.ShowKeyerSpeed(k.wpm)
}

func (k *Keyer) DecreaseSpeed() {
	k.EnterSpeed(k.wpm - 1)
	k.view.ShowKeyerSpeed(k.wpm)
}

func (k *Keyer) EnterPattern(index int, pattern string) {
	(*k.patterns)[index] = pattern
	var err error
	(*k.templates)[index], err = template.New("").Parse(pattern)
	if err != nil {
		k.view.ShowMessage(err)
	} else {
		k.view.ShowMessage()
	}
}

func (k *Keyer) GetPattern(index int) string {
	return (*k.patterns)[index]
}

func (k *Keyer) GetText(index int) (string, error) {
	buffer := bytes.NewBufferString("")
	template, ok := (*k.templates)[index]
	if !ok {
		return "", nil
	}
	err := template.Execute(buffer, k.fillins())
	if err != nil {
		return "", err
	}
	return buffer.String(), nil
}

func (k *Keyer) fillins() map[string]string {
	values := k.values()
	return map[string]string{
		"MyCall":    k.stationCallsign.String(),
		"MyReport":  softcut(values.MyReport.String()),
		"MyNumber":  softcut(values.MyNumber.String()),
		"MyXchange": values.MyXchange,
		"TheirCall": values.TheirCall,
	}
}

func (k *Keyer) Send(index int) {
	message, err := k.GetText(index)
	if err != nil {
		k.view.ShowMessage(err)
		return
	}
	k.send(message)
}

func (k *Keyer) SendQuestion(q string) {
	s := strings.TrimSpace(q) + "?"
	k.send(s)
}

func (k *Keyer) send(s string) {
	if !k.client.IsConnected() {
		err := k.client.Connect()
		if err != nil {
			k.view.ShowMessage(err)
			k.emitStatusChanged(false)
			return
		}
		k.emitStatusChanged(true)
		k.client.Speed(k.wpm)
	}

	log.Printf("sending %s", s)
	k.client.Send(s)
}

func (k *Keyer) Stop() {
	if !k.client.IsConnected() {
		return
	}
	log.Println("abort sending")
	k.client.Abort()
}

func (k *Keyer) Notify(listener interface{}) {
	k.listeners = append(k.listeners, listener)
}

func (k *Keyer) emitStatusChanged(available bool) {
	log.Printf("cw status changed, notifying %d listeners", len(k.listeners))
	for _, listener := range k.listeners {
		if serviceStatusListener, ok := listener.(core.ServiceStatusListener); ok {
			serviceStatusListener.StatusChanged(core.CWDaemonService, available)
		}
	}
}

func (k *Keyer) emitKeyerChanged() {
	keyer := core.Keyer{
		WPM: k.wpm,
		// TODO add patterns here
	}
	for _, listener := range k.listeners {
		if keyerListener, ok := listener.(KeyerListener); ok {
			keyerListener.KeyerChanged(keyer)
		}
	}
}

// softcut replaces 0 and 9 with their "cut" counterparts t and n.
func softcut(s string) string {
	cuts := map[string]string{
		"0": "t",
		"9": "n",
	}
	result := s
	for digit, cut := range cuts {
		result = strings.Replace(result, digit, cut, -1)
	}
	return result
}

// cut replaces digits with the "cut" counterparts. (see http://wiki.bavarian-contest-club.de/wiki/Contest-FAQ#Was_sind_.22Cut_Numbers.22.3F)
func cut(s string) string {
	cuts := map[string]string{
		"0": "t",
		"1": "a",
		"2": "u",
		"3": "v",
		"5": "e",
		"7": "g",
		"8": "d",
		"9": "n",
	}
	result := s
	for digit, cut := range cuts {
		result = strings.Replace(result, digit, cut, -1)
	}
	return result
}

func noValues() core.KeyerValues {
	return core.KeyerValues{}
}

type nullWriter struct{}

func (w *nullWriter) WriteKeyer(core.Keyer) error { return nil }

type nullClient struct{}

func (*nullClient) Connect() error    { return nil }
func (*nullClient) IsConnected() bool { return true }
func (*nullClient) Speed(int)         {}
func (*nullClient) Send(text string)  {}
func (*nullClient) Abort()            {}
