package ui

import (
	"fmt"
	"strings"

	"github.com/gotk3/gotk3/gtk"

	"github.com/ftl/hellocontest/core"
)

type callinfoView struct {
	callsignLabel   *gtk.Label
	dxccLabel       *gtk.Label
	valueLabel      *gtk.Label
	supercheckLabel *gtk.Label
}

func setupCallinfoView(builder *gtk.Builder) *callinfoView {
	result := new(callinfoView)

	result.callsignLabel = getUI(builder, "callsignLabel").(*gtk.Label)
	result.dxccLabel = getUI(builder, "dxccLabel").(*gtk.Label)
	result.valueLabel = getUI(builder, "valueLabel").(*gtk.Label)
	result.supercheckLabel = getUI(builder, "supercheckLabel").(*gtk.Label)

	return result
}

func (v *callinfoView) SetCallsign(callsign string, worked, duplicate bool) {
	if v == nil {
		return
	}

	normalized := strings.ToUpper(strings.TrimSpace(callsign))
	if normalized == "" {
		v.callsignLabel.SetMarkup("-")
		return
	}

	// see https://developer.gnome.org/pango/stable/pango-Markup.html for reference
	attributes := make([]string, 0)
	if duplicate {
		attributes = append(attributes, "background='red' foreground='white'")
	} else if worked {
		attributes = append(attributes, "foreground='cyan'")
	}
	attributeString := strings.Join(attributes, " ")

	renderedCallsign := fmt.Sprintf("<span %s>%s</span>", attributeString, normalized)
	v.callsignLabel.SetMarkup(renderedCallsign)
}

func (v *callinfoView) SetDXCC(name, continent string, itu, cq int, arrlCompliant bool) {
	if v == nil {
		return
	}

	if name == "" {
		v.dxccLabel.SetMarkup("")
		return
	}

	text := fmt.Sprintf("%s, %s", name, continent)
	if itu != 0 {
		text += fmt.Sprintf(", ITU %d", itu)
	}
	if cq != 0 {
		text += fmt.Sprintf(", CQ %d", cq)
	}
	if name != "" && !arrlCompliant {
		text += fmt.Sprintf(", <span foreground='red' font-weight='heavy'>not ARRL compliant</span>")
	}

	v.dxccLabel.SetMarkup(text)
}

func (v *callinfoView) SetValue(points, multis int) {
	if v == nil {
		return
	}

	var pointsMarkup string
	switch {
	case points < 1:
		pointsMarkup = "foreground='silver'"
	case points > 1:
		pointsMarkup = "font-weight='heavy' foreground='cyan'"
	}

	var multisMarkup string
	switch {
	case points < 1:
		multisMarkup = "foreground='silver'"
	case multis > 1:
		multisMarkup = "font-weight='heavy' foreground='cyan'"
	}

	text := fmt.Sprintf("<span %s>%d points</span> / <span %s>%d multis</span>", pointsMarkup, points, multisMarkup, multis)

	v.valueLabel.SetMarkup(text)
}

func (v *callinfoView) SetSupercheck(callsigns []core.AnnotatedCallsign) {
	if v == nil {
		return
	}

	var text string
	for _, callsign := range callsigns {
		// see https://developer.gnome.org/pango/stable/pango-Markup.html for reference
		attributes := make([]string, 0)
		switch {
		case callsign.Duplicate:
			attributes = append(attributes, "foreground='red'")
		case callsign.Worked:
			attributes = append(attributes, "foreground='cyan'")
		case (callsign.Points == 0) && (callsign.Multis == 0):
			attributes = append(attributes, "foreground='silver'")
		}
		if callsign.ExactMatch {
			attributes = append(attributes, "foreground='cyan' font-weight='heavy' font-size='large'")
		}
		attributeString := strings.Join(attributes, " ")

		renderedCallsign := fmt.Sprintf("<span %s>%s</span>", attributeString, callsign.Callsign)

		if text != "" {
			text += "  "
		}
		text += renderedCallsign
	}
	v.supercheckLabel.SetMarkup(text)
}
