package pulseaudio_test

import (
	"github.com/sqp/pulseaudio"

	"testing"
)

func TestLoadModule(t *testing.T) {
	isLoaded, e := pulseaudio.ModuleIsLoaded()
	if e != nil {
		t.Error(e)
		return
	}
	if isLoaded {
		return
	}
	e = pulseaudio.LoadModule()
	if e != nil {
		t.Error(e)
	}
	if !isLoaded {
		e = pulseaudio.UnloadModule()
		if e != nil {
			t.Error(e)
		}
	}
}
