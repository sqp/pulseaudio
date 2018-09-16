package pulseaudio

import (
	"github.com/godbus/dbus"

	"fmt"
	"os/exec"
	"reflect"
	"strings"
)

// Dbus objects paths.
const (
	DbusInterface = "org.PulseAudio.Core1"
	DbusPath      = "/org/pulseaudio/core1"
)

// Client manages a pulseaudio Dbus client session.
//
type Client struct {
	conn          *dbus.Conn
	hooker        *Hooker
	ch            chan *dbus.Signal
	unknownSignal func(*dbus.Signal)
}

// New creates a new pulseaudio Dbus client session.
//
func New() (*Client, error) { // chan *dbus.Signal
	addr, e := serverLookup()
	if e != nil {
		return nil, e
	}

	conn, e := dbus.Dial(addr)
	if e != nil {
		return nil, e
	}

	e = conn.Auth(nil)
	if e != nil {
		conn.Close()
		return nil, e
	}

	pulse := &Client{
		conn:          conn,
		hooker:        NewHooker(),
		unknownSignal: func(s *dbus.Signal) { fmt.Println("unknown signal", s.Name, s.Path) },
	}

	pulse.hooker.AddCalls(PulseCalls)
	pulse.hooker.AddTypes(PulseTypes)

	return pulse, nil
}

// Close closes the DBus connection. The client can't be reused after.
//
func (pulse *Client) Close() error {
	return pulse.conn.Close()
}

// Register connects an object to the pulseaudio events hooks it implements.
// If the object declares any of the method in the On... interfaces list, it
// will be registered to receive those events.
//
func (pulse *Client) Register(obj interface{}) (errs []error) {
	tolisten := pulse.hooker.Register(obj)
	for _, name := range tolisten {
		e := pulse.ListenForSignal(name)
		if e != nil {
			errs = append(errs, e)
		}
	}
	return errs
}

// Unregister disconnects an object from the pulseaudio events hooks.
//
func (pulse *Client) Unregister(obj interface{}) (errs []error) {
	tounlisten := pulse.hooker.Unregister(obj)
	for _, name := range tounlisten {
		e := pulse.StopListeningForSignal(name)
		if e != nil {
			errs = append(errs, e)
		}
	}
	return errs
}

// Listen awaits for pulseaudio messages and dispatch events to registered clients.
//
func (pulse *Client) Listen() {
	pulse.ch = make(chan *dbus.Signal, 10)
	pulse.conn.Signal(pulse.ch)

	for s := range pulse.ch {
		pulse.DispatchSignal(s)
	}
}

// StopListening unregisters an listened event.
//
func (pulse *Client) StopListening() {
	pulse.conn.RemoveSignal(pulse.ch)
	close(pulse.ch)
}

// DispatchSignal forwards a signal event to the registered clients.
//
func (pulse *Client) DispatchSignal(s *dbus.Signal) {
	name := strings.TrimPrefix(string(s.Name), DbusInterface+".")
	if name != s.Name { // dbus interface matched.
		if pulse.hooker.Call(name, s) {
			return // signal was defined (even if no clients are connected).
		}
	}
	pulse.unknownSignal(s)
}

// SetOnUnknownSignal sets the unknown signal logger callback. Optional
//
func (pulse *Client) SetOnUnknownSignal(call func(s *dbus.Signal)) {
	pulse.unknownSignal = call
}

//
//------------------------------------------------------------[ DBUS METHODS ]--

// ListenForSignal registers a new event to listen.
//
func (pulse *Client) ListenForSignal(name string, paths ...dbus.ObjectPath) error {
	args := []interface{}{DbusInterface + "." + name, paths}
	return pulse.Core().Call("ListenForSignal", 0, args...).Err
}

// StopListeningForSignal unregisters an listened event.
//
func (pulse *Client) StopListeningForSignal(name string) error {
	return pulse.Core().Call("StopListeningForSignal", 0, DbusInterface+"."+name).Err
}

//
//-----------------------------------------------------[ CALLBACK INTERFACES ]--

// OnFallbackSinkUpdated is an interface to the FallbackSinkUpdated method.
type OnFallbackSinkUpdated interface {
	FallbackSinkUpdated(dbus.ObjectPath)
}

// OnFallbackSinkUnset is an interface to the FallbackSinkUnset method.
type OnFallbackSinkUnset interface {
	FallbackSinkUnset()
}

// OnNewSink is an interface to the NewSink method.
type OnNewSink interface {
	NewSink(dbus.ObjectPath)
}

// OnSinkRemoved is an interface to the SinkRemoved method.
type OnSinkRemoved interface {
	SinkRemoved(dbus.ObjectPath)
}

// OnNewPlaybackStream is an interface to the NewPlaybackStream method.
type OnNewPlaybackStream interface {
	NewPlaybackStream(dbus.ObjectPath)
}

// OnPlaybackStreamRemoved is an interface to the PlaybackStreamRemoved method.
type OnPlaybackStreamRemoved interface {
	PlaybackStreamRemoved(dbus.ObjectPath)
}

// OnDeviceVolumeUpdated is an interface to the DeviceVolumeUpdated method.
type OnDeviceVolumeUpdated interface {
	DeviceVolumeUpdated(dbus.ObjectPath, []uint32)
}

// OnDeviceMuteUpdated is an interface to the DeviceMuteUpdated method.
type OnDeviceMuteUpdated interface {
	DeviceMuteUpdated(dbus.ObjectPath, bool)
}

// OnStreamVolumeUpdated is an interface to the StreamVolumeUpdated method.
type OnStreamVolumeUpdated interface {
	StreamVolumeUpdated(dbus.ObjectPath, []uint32)
}

// OnStreamMuteUpdated is an interface to the StreamMuteUpdated method.
type OnStreamMuteUpdated interface {
	StreamMuteUpdated(dbus.ObjectPath, bool)
}

// OnDeviceActivePortUpdated is an interface to the DeviceActivePortUpdated method.
type OnDeviceActivePortUpdated interface {
	DeviceActivePortUpdated(dbus.ObjectPath, dbus.ObjectPath)
}

//
//--------------------------------------------------------[ CALLBACK METHODS ]--

// PulseCalls defines callbacks methods to call the matching object method with
// type-asserted arguments.
// Public so it can be hacked before the first Register.
//
var PulseCalls = Calls{
	"FallbackSinkUpdated":      func(m Msg) { m.O.(OnFallbackSinkUpdated).FallbackSinkUpdated(m.P) },
	"FallbackSinkUnset":        func(m Msg) { m.O.(OnFallbackSinkUnset).FallbackSinkUnset() },
	"NewSink":                  func(m Msg) { m.O.(OnNewSink).NewSink(m.P) },
	"SinkRemoved":              func(m Msg) { m.O.(OnSinkRemoved).SinkRemoved(m.P) },
	"NewPlaybackStream":        func(m Msg) { m.O.(OnNewPlaybackStream).NewPlaybackStream(m.D[0].(dbus.ObjectPath)) },
	"PlaybackStreamRemoved":    func(m Msg) { m.O.(OnPlaybackStreamRemoved).PlaybackStreamRemoved(m.D[0].(dbus.ObjectPath)) },
	"Device.VolumeUpdated":     func(m Msg) { m.O.(OnDeviceVolumeUpdated).DeviceVolumeUpdated(m.P, m.D[0].([]uint32)) },
	"Device.MuteUpdated":       func(m Msg) { m.O.(OnDeviceMuteUpdated).DeviceMuteUpdated(m.P, m.D[0].(bool)) },
	"Device.ActivePortUpdated": func(m Msg) { m.O.(OnDeviceActivePortUpdated).DeviceActivePortUpdated(m.P, m.D[0].(dbus.ObjectPath)) },
	"Stream.VolumeUpdated":     func(m Msg) { m.O.(OnStreamVolumeUpdated).StreamVolumeUpdated(m.P, m.D[0].([]uint32)) },
	"Stream.MuteUpdated":       func(m Msg) { m.O.(OnStreamMuteUpdated).StreamMuteUpdated(m.P, m.D[0].(bool)) },
}

// PulseTypes defines interface types for events to register.
// Public so it can be hacked before the first Register.
//
var PulseTypes = map[string]reflect.Type{
	"FallbackSinkUpdated":      reflect.TypeOf((*OnFallbackSinkUpdated)(nil)).Elem(),
	"FallbackSinkUnset":        reflect.TypeOf((*OnFallbackSinkUnset)(nil)).Elem(),
	"NewSink":                  reflect.TypeOf((*OnNewSink)(nil)).Elem(),
	"SinkRemoved":              reflect.TypeOf((*OnSinkRemoved)(nil)).Elem(),
	"NewPlaybackStream":        reflect.TypeOf((*OnNewPlaybackStream)(nil)).Elem(),
	"PlaybackStreamRemoved":    reflect.TypeOf((*OnPlaybackStreamRemoved)(nil)).Elem(),
	"Device.VolumeUpdated":     reflect.TypeOf((*OnDeviceVolumeUpdated)(nil)).Elem(),
	"Device.MuteUpdated":       reflect.TypeOf((*OnDeviceMuteUpdated)(nil)).Elem(),
	"Device.ActivePortUpdated": reflect.TypeOf((*OnDeviceActivePortUpdated)(nil)).Elem(),
	"Stream.VolumeUpdated":     reflect.TypeOf((*OnStreamVolumeUpdated)(nil)).Elem(),
	"Stream.MuteUpdated":       reflect.TypeOf((*OnStreamMuteUpdated)(nil)).Elem(),
}

//
//------------------------------------------------------------------[ COMMON ]--

// serverLookup asks the main service for the location of the real service.
// It's the only thing the pulseaudio service do on the main session dbus.
// On my system, it returns  "unix:path=/run/user/1000/pulse/dbus-socket"
//
func serverLookup() (string, error) {
	conn, e := dbus.SessionBus()
	if e != nil {
		return "", e
	}
	srv := NewObject(conn, "org.PulseAudio1", "/org/pulseaudio/server_lookup1")
	addr, ep := srv.GetProperty("org.PulseAudio.ServerLookup1.Address")
	if ep != nil {
		return "", ep
	}
	return addr.Value().(string), nil
}

//

// The next part could (should) be moved as another package^w lib)

//
//-------------------------------------------------------------[ FACTO PROPS ]--

// Object extends the dbus Object with properties access methods.
//
type Object struct {
	dbus.BusObject
	prefix string
}

// NewObject creates a dbus Object with properties access methods.
//
func NewObject(conn *dbus.Conn, interf string, path dbus.ObjectPath) *Object {
	return &Object{conn.Object(interf, path), interf}
}

// Get queries an object property and set its value to dest.
// dest must be a pointer to the type of data returned by the method.
//
func (dev *Object) Get(property string, dest interface{}) error {
	v, e := dev.GetProperty(dev.prefix + "." + property)
	if e != nil {
		return e
	}

	switch val := v.Value().(type) {
	case bool, uint32, uint64, string, dbus.ObjectPath,
		[]uint32, []string, []dbus.ObjectPath:

		reflect.ValueOf(dest).Elem().Set(reflect.ValueOf(val))
		return nil

	case map[string][]byte:
		tmp := make(map[string]string)
		for k, v := range val {
			if len(v) > 0 {
				tmp[k] = string(v[:len(v)-1]) // remove \0 at end.
			}
		}
		reflect.ValueOf(dest).Elem().Set(reflect.ValueOf(tmp))
		return nil
	}
	return fmt.Errorf("unknown type, %T to %T", v.Value(), dest)
}

// Set updates the given object property with value.
//
func (dev *Object) Set(property string, value interface{}) error {
	return dev.SetProperty(dev.prefix+"."+property, value)
}

//
//---------------------------------------------------[ GET CASTED PROPERTIES ]--

// Bool queries an object property and return it as bool.
//
func (dev *Object) Bool(name string) (val bool, e error) {
	e = dev.Get(name, &val)
	return val, e
}

// Uint32 queries an object property and return it as uint32.
//
func (dev *Object) Uint32(name string) (val uint32, e error) {
	e = dev.Get(name, &val)
	return val, e
}

// Uint64 queries an object property and return it as uint64.
//
func (dev *Object) Uint64(name string) (val uint64, e error) {
	e = dev.Get(name, &val)
	return val, e
}

// String queries an object property and return it as string.
//
func (dev *Object) String(name string) (val string, e error) {
	e = dev.Get(name, &val)
	return val, e
}

// ObjectPath queries an object property and return it as string.
//
func (dev *Object) ObjectPath(name string) (val dbus.ObjectPath, e error) {
	e = dev.Get(name, &val)
	return val, e
}

// ListUint32 queries an object property and return it as []uint32.
//
func (dev *Object) ListUint32(name string) (val []uint32, e error) {
	e = dev.Get(name, &val)
	return val, e
}

// ListString queries an object property and return it as []string.
//
func (dev *Object) ListString(name string) (val []string, e error) {
	e = dev.Get(name, &val)
	return val, e
}

// ListPath queries an object property and return it as []dbus.ObjectPath.
//
func (dev *Object) ListPath(name string) (val []dbus.ObjectPath, e error) {
	e = dev.Get(name, &val)
	return val, e
}

// MapString queries an object property and return it as map[string]string.
//
func (dev *Object) MapString(name string) (val map[string]string, e error) {
	e = dev.Get(name, &val)
	return val, e
}

// SetProperty calls org.freedesktop.DBus.Properties.Set on the given object.
// The property name must be given in interface.member notation.
//
// TODO: Should be moved to the dbus api.
//
func (dev *Object) SetProperty(p string, val interface{}) error {
	idx := strings.LastIndex(p, ".")
	if idx == -1 || idx+1 == len(p) {
		return fmt.Errorf("dbus: invalid property %s", p)
	}

	iface := p[:idx]
	prop := p[idx+1:]
	v := dbus.MakeVariant(val)
	return dev.Call("org.freedesktop.DBus.Properties.Set", 0, iface, prop, v).Err
}

//
//-------------------------------------------------------------------[ HOOKS ]--

// Msg defines an dbus signal event message.
//
type Msg struct {
	O interface{}     // client object.
	P dbus.ObjectPath // signal path.
	D []interface{}   // signal data.
}

// Calls defines a list of event callback methods indexed by dbus method name.
//
type Calls map[string]func(Msg)

// Types defines a list of interfaces types indexed by dbus method name.
//
type Types map[string]reflect.Type

// Hooker defines a list of objects indexed by the methods they implement.
// An object can be referenced multiple times.
// If an object declares all methods, it will be referenced in every field.
//   hooker:= NewHooker()
//   hooker.AddCalls(myCalls)
//   hooker.AddTypes(myTypes)
//
//   // create a type with some of your callback methods and register it.
//   tolisten := hooker.Register(obj) // tolisten is the list of events you may have to listen.
//
//   // add the signal forwarder in your events listening loop.
//   matched := Call(signalName, dbusSignal)
//
type Hooker struct {
	Hooks map[string][]interface{}
	Calls Calls
	Types Types
}

// NewHooker handles a loosely coupled hook interface to forward dbus signals
// to registered clients.
//
func NewHooker() *Hooker {
	return &Hooker{
		Hooks: make(map[string][]interface{}),
		Calls: make(Calls),
		Types: make(Types),
	}
}

// Call forwards a Dbus event to registered clients for this event.
//
func (hook Hooker) Call(name string, s *dbus.Signal) bool {
	call, ok := hook.Calls[name]
	if !ok { // Signal name not defined.
		return false
	}
	if list, ok := hook.Hooks[name]; ok { // Hook clients found.
		for _, obj := range list {
			call(Msg{obj, s.Path, s.Body})
		}
	}
	return true
}

// Register connects an object to the events hooks it implements.
// If the object implements any of the interfaces types declared, it will be
// registered to receive the matching events.
// //
func (hook Hooker) Register(obj interface{}) (tolisten []string) {
	t := reflect.ValueOf(obj).Type()
	for name, modelType := range hook.Types {
		if t.Implements(modelType) {
			hook.Hooks[name] = append(hook.Hooks[name], obj)
			if len(hook.Hooks[name]) == 1 { // First client registered for this event. need to listen.
				tolisten = append(tolisten, name)
			}
		}
	}
	return tolisten
}

// Unregister disconnects an object from the events hooks.
//
func (hook Hooker) Unregister(obj interface{}) (tounlisten []string) {
	for name, list := range hook.Hooks {
		hook.Hooks[name] = hook.remove(list, obj)
		if len(hook.Hooks[name]) == 0 {
			delete(hook.Hooks, name)
			tounlisten = append(tounlisten, name) // No more clients, need to unlisten.
		}
	}
	return tounlisten
}

// AddCalls registers a list of callback methods.
//
func (hook Hooker) AddCalls(calls Calls) {
	for name, call := range calls {
		hook.Calls[name] = call
	}
}

// AddTypes registers a list of interfaces types.
//
func (hook Hooker) AddTypes(tests Types) {
	for name, test := range tests {
		hook.Types[name] = test
	}
}

// remove removes an object from the list if found.
//
func (hook Hooker) remove(list []interface{}, obj interface{}) []interface{} {
	for i, test := range list {
		if obj == test {
			return append(list[:i], list[i+1:]...)
		}
	}
	return list
}

//
//-------------------------------------------------------[ PULSE DBUS MODULE ]--

// LoadModule loads the PulseAudio DBus module.
//
func LoadModule() error {
	return exec.Command("pacmd", "load-module", "module-dbus-protocol").Run()
}

// UnloadModule unloads the PulseAudio DBus module.
//
func UnloadModule() error {
	return exec.Command("pacmd", "unload-module", "module-dbus-protocol").Run()
}

// ModuleIsLoaded tests if the PulseAudio DBus module is loaded.
//
func ModuleIsLoaded() (bool, error) {
	out, e := exec.Command("pacmd", "list-modules").CombinedOutput()
	return strings.Contains(string(out), "<module-dbus-protocol>"), e
}
