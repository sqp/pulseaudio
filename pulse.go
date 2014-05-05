package pulseaudio

import (
	"github.com/guelfey/go.dbus" // imported as dbus.
	// "github.com/guelfey/go.dbus/introspect"

	"errors"
	"log"
	"strings"
)

const (
	DbusIf   = "org.PulseAudio.Core1"
	DbusPath = "/org/pulseaudio/core1"
)

type Core struct {
	dbusCore  *dbus.Object
	conn      *dbus.Conn
	err       []error
	hooks     Hooks
	Callbacks Callbacks
}

// NewCore creates a pulseaudio Dbus session.
//
func NewCore() (*Core, error) { // chan *dbus.Signal

	addr := "unix:path=/run/user/1000/pulse/dbus-socket"

	conn, e := dbus.Dial(addr)
	if e != nil {
		return nil, e
	}
	if e = conn.Auth(nil); e != nil {
		conn.Close()
		return nil, e
	}

	pulse := &Core{
		conn:      conn,
		dbusCore:  conn.Object(DbusIf, DbusPath),
		hooks:     make(Hooks),
		Callbacks: DefaultCallbacks(),
	}

	if pulse.dbusCore == nil {
		conn.Close()
		return nil, errors.New("no Dbus interface")
	}

	pulse.ListenForSignal("NewSink", []dbus.ObjectPath{})
	pulse.ListenForSignal("SinkRemoved", []dbus.ObjectPath{})
	pulse.ListenForSignal("NewPlaybackStream", []dbus.ObjectPath{})
	pulse.ListenForSignal("PlaybackStreamRemoved", []dbus.ObjectPath{})

	pulse.ListenForSignal("Stream.VolumeUpdated", []dbus.ObjectPath{})
	pulse.ListenForSignal("Device.VolumeUpdated", []dbus.ObjectPath{})

	return pulse, nil
}

// ListenForSignal registers a new event to listen.
//
func (pulse *Core) ListenForSignal(name string, paths []dbus.ObjectPath) {
	args := []interface{}{DbusIf + "." + name, paths}
	pulse.testErr(pulse.dbusCore.Call("ListenForSignal", 0, args...).Err)
}

//
//----------------------------------------------------------[ LOOP & SIGNALS ]--

// Listen awaits for pulseaudio messages and dispatch events to registered clients.
//
func (pulse *Core) Listen() {
	c := make(chan *dbus.Message, 10)
	pulse.conn.Eavesdrop(c)

	for msg := range c {
		switch msg.Type {
		case dbus.TypeSignal:
			s := msg.ToSignal()
			pulse.DispatchSignal(s)

			// 	case dbus.TypeMethodCall:
		}
	}
}

// DispatchSignal forwards a signal event to the registered clients.
//
func (pulse *Core) DispatchSignal(s *dbus.Signal) {
	if strings.HasPrefix(string(s.Name), DbusIf+".") {
		name := s.Name[len(DbusIf)+1:]

		if call, ok := pulse.Callbacks[name]; ok { // Signal name defined.
			if hooks, ok := pulse.hooks[name]; ok { // Hook clients found.
				for _, uncast := range hooks {
					call(s, uncast)
				}
				return
			}
		}
	}

	log.Println("unknown signal", s.Name, s.Path)
}

//
//-----------------------------------------------------[ CALLBACK INTERFACES ]--

type DefineNewPlaybackStream interface {
	NewPlaybackStream(dbus.ObjectPath)
}

type DefineDeviceVolumeUpdated interface {
	DeviceVolumeUpdated(dbus.ObjectPath, []uint32)
}

type DefineStreamVolumeUpdated interface {
	StreamVolumeUpdated(dbus.ObjectPath, []uint32)
}

//
//--------------------------------------------------------[ CALLBACK METHODS ]--

// DefaultCallbacks provides callbacks for events to forward to an uncasted object.
// The signal arguments will be casted to their usefull type so they can be used
// directly by clients.
//
func DefaultCallbacks() Callbacks {
	return Callbacks{
		"NewPlaybackStream": func(s *dbus.Signal, uncast interface{}) {
			uncast.(DefineNewPlaybackStream).NewPlaybackStream(s.Path)
		},

		"Device.VolumeUpdated": func(s *dbus.Signal, uncast interface{}) {
			uncast.(DefineDeviceVolumeUpdated).DeviceVolumeUpdated(s.Path, s.Body[0].([]uint32))
		},

		"Stream.VolumeUpdated": func(s *dbus.Signal, uncast interface{}) {
			uncast.(DefineStreamVolumeUpdated).StreamVolumeUpdated(s.Path, s.Body[0].([]uint32))
		},
	}
}

// Register connects an object to the pulseaudio events hooks it implements.
// If the object declares any of the method in the Define interfaces list, it
// will be registered to receive those events.
//
func (pulse *Core) Register(obj interface{}) {
	if _, ok := obj.(DefineNewPlaybackStream); ok {
		pulse.hooks.Append("NewPlaybackStream", obj)
	}

	if _, ok := obj.(DefineDeviceVolumeUpdated); ok {
		pulse.hooks.Append("Device.VolumeUpdated", obj)
	}

	if _, ok := obj.(DefineStreamVolumeUpdated); ok {
		pulse.hooks.Append("Stream.VolumeUpdated", obj)
	}
}

//
//-------------------------------------------------------------------[ HOOKS ]--

// Callback defines an event callback method.
//
type Callback func(*dbus.Signal, interface{})

// Callback defines a list of event callback methods indexed by dbus method name.
//
type Callbacks map[string]Callback

// Hook defines a list of objects indexed by the methods they implement.
// An object can be references multiple times.
// If an object declares all methods, he will be referenced in every field.
//
type Hooks map[string][]interface{}

// Append adds an object to the list for the given index name.
//
func (hook Hooks) Append(name string, obj interface{}) {
	hook[name] = append(hook[name], obj)
}

//
//--------------------------------------------------------------[ PROPERTIES ]--

// Name returns the service name.
//
func (pulse *Core) Name() string {
	uncast, e := pulse.GetProperty("Name")
	pulse.testErr(e)
	return uncast.(string)
}

// Sinks
//
func (pulse *Core) Sinks() []dbus.ObjectPath {
	uncast, e := pulse.GetProperty("Sinks")
	pulse.testErr(e)
	return uncast.([]dbus.ObjectPath)
}

func (pulse *Core) PlaybackStreams() []dbus.ObjectPath {
	uncast, e := pulse.GetProperty("PlaybackStreams")
	pulse.testErr(e)
	return uncast.([]dbus.ObjectPath)
}

func (pulse *Core) Volume(sink dbus.ObjectPath) []uint32 {
	obj := pulse.conn.Object(DbusIf+".Device", sink)

	uncast, e := obj.GetProperty("org.PulseAudio.Core1.Device.Volume")
	if pulse.testErr(e) {
		return []uint32{}
	}
	return uncast.Value().([]uint32)
}

func (pulse *Core) GetProperty(name string) (interface{}, error) {
	v, e := pulse.dbusCore.GetProperty("org.PulseAudio.Core1." + name)
	return v.Value(), e
}

// e = sink.SetProperty("org.PulseAudio.Core1.Device.Volume", dbus.MakeVariant([]uint32{32000, 32000}))
// log.Err(e, "SetVolume")

//
//------------------------------------------------------------------[ ERRORS ]--

func (pulse *Core) Failed() bool {
	return len(pulse.err) > 0
}

func (pulse *Core) testErr(e error) bool {
	if e != nil {
		pulse.err = append(pulse.err, e)
	}
	return e != nil
}

//
//-------------------------------------------------------------------[ CLIENT ]--

// AppPulse is a client that connects 3 callbacks.
type AppPulse struct{}

func (ap *AppPulse) NewPlaybackStream(path dbus.ObjectPath) {
	log.Println("one: new stream", path)
}

func (ap *AppPulse) DeviceVolumeUpdated(path dbus.ObjectPath, values []uint32) {
	log.Println("one: device volume", path, values)
}

func (ap *AppPulse) StreamVolumeUpdated(path dbus.ObjectPath, values []uint32) {
	log.Println("one: stream volume", path, values)
}

// ClientTwo is a client that connects only one callback.
type ClientTwo struct{}

func (ap *ClientTwo) DeviceVolumeUpdated(path dbus.ObjectPath, values []uint32) {
	log.Println("two: volume updated", path)
}

// Create a pulse dbus service with 2 clients.
func main() {
	pulse, e := NewCore()
	if e != nil {
		log.Panicln("connect", e)
	}

	app := &AppPulse{}
	pulse.Register(app)

	two := &ClientTwo{}
	pulse.Register(two)

	// log.Println(pulse.Name())
	// log.Println(pulse.Sinks())
	// log.Println(pulse.PlaybackStreams())
	// log.Println(pulse.Volume(dbus.ObjectPath(DbusPath + "/sink0")))

	pulse.Listen()
}

//

// introspect

// s, ei := introspect.Call(pulse.dbusCore)
// log.Err(ei, "intro")
// for _, interf := range s.Interfaces {
// 	log.Println(interf.Name)
// 	for _, sig := range interf.Methods {
// 		log.Println("  method", sig)
// 	}

// 	log.Println(interf.Name)
// 	for _, sig := range interf.Signals {
// 		log.Println("  signal", sig)
// 	}
// }

//

//
//------------------------------------------------------------------[ COMMON ]--

// func MsgToSignal(msg *dbus.Message) *dbus.Signal {

// 	log.Println(msg.Headers[dbus.FieldSender])

// 	iface := msg.Headers[dbus.FieldInterface].String()
// 	member := msg.Headers[dbus.FieldMember].String()
// 	// sender := msg.Headers[dbus.FieldSender].String()
// 	return &dbus.Signal{
// 		// Sender: sender,
// 		Path: dbus.ObjectPath(msg.Headers[dbus.FieldPath].String()),
// 		Name: iface + "." + member,
// 		Body: msg.Body,
// 	}
// }

// func (pulse *Core) call(name string, s *dbus.Signal, call func(*dbus.Signal, interface{})) {
// 	for _, uncast := range pulse.hooks[name] {
// 		call(s, uncast)
// 	}
// }
