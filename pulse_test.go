package pulseaudio_test

import (
	"github.com/godbus/dbus"

	"github.com/sqp/pulseaudio"

	"fmt"
	"log"
	"testing"
)

func TestLoadModule(t *testing.T) {
	isLoaded, e := pulseaudio.ModuleIsLoaded()
	testFatal(e, "test pulse dbus module is loaded")

	if isLoaded {
		coverFailPath(t)

		e = pulseaudio.UnloadModule()
		testFatal(e, "unload pulse dbus module")

		e = pulseaudio.LoadModule()
		testFatal(e, "load pulse dbus module")
		return
	}

	e = pulseaudio.LoadModule()
	testFatal(e, "load pulse dbus module")

	coverFailPath(t)

	e = pulseaudio.UnloadModule()
	testFatal(e, "unload pulse dbus module")
}

func coverFailPath(t *testing.T) {
	pulse, e := pulseaudio.New()
	testFatal(e, "new pulse")
	defer pulse.Close()

	pulse.DispatchSignal(&dbus.Signal{Name: pulseaudio.DbusInterface + ".Invalid"})
	pulse.DispatchSignal(&dbus.Signal{Name: "Invalid"})

	pulse.SetOnUnknownSignal(func(s *dbus.Signal) { log.Println("unknown signal", s.Name, s.Path) })
	pulse.DispatchSignal(&dbus.Signal{Name: pulseaudio.DbusInterface + ".Invalid"})
	pulse.DispatchSignal(&dbus.Signal{Name: "Invalid"})

	version, e := pulse.Core().String("Version")
	testFatal(e, "get core version")
	log.Println("CoreVersion", version)

	exts, e := pulse.Core().ListString("Extensions")
	testFatal(e, "get core extensions")
	log.Println("CoreExtensions", exts)

	sinks, e := pulse.Core().ListPath("Sinks")
	testFatal(e, "get list of sinks")

	if len(sinks) == 0 {
		fmt.Println("no sinks to test")
		return
	}

	dev := pulse.Device(sinks[0])
	dev.SetProperty("willfail", nil)
}
