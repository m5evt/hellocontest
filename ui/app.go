package ui

import (
	"log"

	"github.com/ftl/hellocontest/core"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

// Run the application
func Run(args []string) {
	var err error
	app := &application{id: "ft.hellocontest"}
	app.app, err = gtk.ApplicationNew(app.id, glib.APPLICATION_FLAGS_NONE)
	if err != nil {
		log.Fatal("Cannot create application: ", err)
	}

	app.app.Connect("startup", app.startup)
	app.app.Connect("activate", app.activate)
	app.app.Connect("shutdown", app.shutdown)

	app.app.Run(args)
}

type application struct {
	id         string
	app        *gtk.Application
	builder    *gtk.Builder
	mainWindow *mainWindow
	clock      core.Clock
	log        core.Log
	entry      core.EntryController
}

func (app *application) startup() {
}

func (app *application) activate() {
	app.builder = setupBuilder()

	app.mainWindow = setupMainWindow(app.builder, app.app)
	app.mainWindow.Show()

	app.clock = core.NewClock()
	app.log = core.NewLog(app.clock)
	app.log.SetView(app.mainWindow)
	app.entry = core.NewEntryController(app.clock, app.log)
	app.entry.SetView(app.mainWindow)
}

func (app *application) shutdown() {
}

func setupBuilder() *gtk.Builder {
	builder, err := gtk.BuilderNew()
	if err != nil {
		log.Fatal("Cannot create builder: ", err)
	}

	builder.AddFromFile("ui/glade/contest.glade")

	return builder
}
