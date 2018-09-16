package pulseaudio_test

import (
	"github.com/godbus/dbus"
	"github.com/sqp/pulseaudio"

	"fmt"
	"log"
	"strconv"
	"time"
)

//
//--------------------------------------------------------------------[ MAIN ]--

// Create a pulse dbus service with 2 clients, listen to events,
// then use some properties.
//
func Example() {
	// Load pulseaudio DBus module if needed. This module is mandatory, but it
	// can also be configured in system files. See package doc.
	isLoaded, e := pulseaudio.ModuleIsLoaded()
	testFatal(e, "test pulse dbus module is loaded")
	if !isLoaded {
		e = pulseaudio.LoadModule()
		testFatal(e, "load pulse dbus module")

		defer pulseaudio.UnloadModule() // has error to test
	}

	// Connect to the pulseaudio dbus service.
	pulse, e := pulseaudio.New()
	testFatal(e, "connect to the pulse service")
	defer pulse.Close() // has error to test

	// Create and register a first client.
	app := &AppPulse{}
	pulse.Register(app)
	defer pulse.Unregister(app) // has errors to test

	// Create and register a second client (optional).
	two := &ClientTwo{pulse}
	pulse.Register(two)
	defer pulse.Unregister(two) // has errors to test

	// Listen to registered events.
	go pulse.Listen()
	defer pulse.StopListening()

	// Use some properties.
	GetProps(pulse)
	SetProps(pulse)

	// Output:
	// two: device mute updated /org/pulseaudio/core1/sink0 true
	// sink muted
	// two: device mute updated /org/pulseaudio/core1/sink0 false
	// sink restored
}

//
//--------------------------------------------------------------[ CLIENT ONE ]--

// AppPulse is a client that connects 6 callbacks.
//
type AppPulse struct{}

// NewSink is called when a sink is added.
//
func (ap *AppPulse) NewSink(path dbus.ObjectPath) {
	log.Println("one: new sink", path)
}

// SinkRemoved is called when a sink is removed.
//
func (ap *AppPulse) SinkRemoved(path dbus.ObjectPath) {
	log.Println("one: sink removed", path)
}

// NewPlaybackStream is called when a playback stream is added.
//
func (ap *AppPulse) NewPlaybackStream(path dbus.ObjectPath) {
	log.Println("one: new playback stream", path)
}

// PlaybackStreamRemoved is called when a playback stream is removed.
//
func (ap *AppPulse) PlaybackStreamRemoved(path dbus.ObjectPath) {
	log.Println("one: playback stream removed", path)
}

// DeviceVolumeUpdated is called when the volume has changed on a device.
//
func (ap *AppPulse) DeviceVolumeUpdated(path dbus.ObjectPath, values []uint32) {
	log.Println("one: device volume updated", path, values)
}

// DeviceActiveCardUpdated is called when active card has changed on a device.
// i.e. headphones injected.
func (ap *AppPulse) DeviceActiveCardUpdated(path dbus.ObjectPath, port dbus.ObjectPath) {
	log.Println("one: device active card updated", path, port)
}

// StreamVolumeUpdated is called when the volume has changed on a stream.
//
func (ap *AppPulse) StreamVolumeUpdated(path dbus.ObjectPath, values []uint32) {
	log.Println("one: stream volume", path, values)
}

//
//--------------------------------------------------------------[ CLIENT TWO ]--

// ClientTwo is a client that also connects some callbacks (3).
//
type ClientTwo struct {
	*pulseaudio.Client
}

// DeviceVolumeUpdated is called when the volume has changed on a device.
//
func (two *ClientTwo) DeviceVolumeUpdated(path dbus.ObjectPath, values []uint32) {
	log.Println("two: device volume updated", path, values)
}

// DeviceMuteUpdated is called when the output has been (un)muted.
//
func (two *ClientTwo) DeviceMuteUpdated(path dbus.ObjectPath, state bool) {
	fmt.Println("two: device mute updated", path, state)
}

// DeviceActivePortUpdated is called when the port has changed on a device.
// Like a cable connected.
//
func (two *ClientTwo) DeviceActivePortUpdated(path, path2 dbus.ObjectPath) {
	log.Println("two: device port updated", path, path2)
}

//
//----------------------------------------------[ GET OBJECTS AND PROPERTIES ]--

// GetProps is an example to show how to get properties.
func GetProps(client *pulseaudio.Client) {
	// Get the list of streams from the Core and show some informations about them.
	// You better handle errors that were not checked here for code clarity.

	// Get the list of playback streams from the core.
	streams, _ := client.Core().ListPath("PlaybackStreams") // []ObjectPath
	for _, stream := range streams {

		// Get the device to query properties for the stream referenced by his path.
		dev := client.Stream(stream)

		// Get some informations about this stream.
		mute, _ := dev.Bool("Mute")               // bool
		vols, _ := dev.ListUint32("Volume")       // []uint32
		latency, _ := dev.Uint64("Latency")       // uint64
		sampleRate, _ := dev.Uint32("SampleRate") // uint32
		log.Println("stream", volumeText(mute, vols), "latency", latency, "sampleRate", sampleRate)

		props, e := dev.MapString("PropertyList") // map[string]string
		testFatal(e, "get device PropertyList")
		log.Println(props)

		// Get the client associated with the stream.
		devcltpath, _ := dev.ObjectPath("Client") // ObjectPath
		devclt := client.Client(devcltpath)
		devcltdrv, _ := devclt.String("Driver") // string
		log.Println("device client driver", devcltdrv)
	}
}

// SetProps is an example to show how to set properties.
// Toggles twice the mute state of the first sink device.
func SetProps(client *pulseaudio.Client) {
	sinks, e := client.Core().ListPath("Sinks")
	testFatal(e, "get list of sinks")

	if len(sinks) == 0 {
		fmt.Println("no sinks to test")
		return
	}

	dev := client.Device(sinks[0]) // Only use the first sink for the test.

	var muted bool
	e = dev.Get("Mute", &muted) // Get is a generic method to get properties.
	testFatal(e, "get sink muted state")

	e = dev.Set("Mute", !muted)
	testFatal(e, "set sink muted state")

	<-time.After(time.Millisecond * 100)
	fmt.Println("sink muted")

	e = dev.Set("Mute", muted) // For properties tagged RW in the doc.
	testFatal(e, "set sink muted state")

	<-time.After(time.Millisecond * 100)
	fmt.Println("sink restored")
}

//
//------------------------------------------------------------------[ COMMON ]--

func volumeText(mute bool, vals []uint32) string {
	if mute {
		return "muted"
	}
	vol := int(volumeAverage(vals)) * 100 / 65535
	return " " + strconv.Itoa(vol) + "% "
}

func volumeAverage(vals []uint32) uint32 {
	var vol uint32
	if len(vals) > 0 {
		for _, cur := range vals {
			vol += cur
		}
		vol /= uint32(len(vals))
	}
	return vol
}

func testFatal(e error, msg string) {
	if e != nil {
		log.Fatalln(msg+":", e)
	}
}
