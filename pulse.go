package pulseaudio

import (
	"github.com/godbus/dbus"

	"errors"
	"log"
	"reflect"
	"strings"
)

const (
	DbusIf   = "org.PulseAudio.Core1"
	DbusPath = "/org/pulseaudio/core1"
)

type Client struct {
	conn   *dbus.Conn
	hooker *Hooker
}

// New creates a new pulseaudio Dbus client session.
//
func New() (*Client, error) { // chan *dbus.Signal
	addr, es := serverLookup()
	if es != nil {
		return nil, es
	}

	conn, ed := dbus.Dial(addr)
	if ed != nil {
		return nil, ed
	}

	if ea := conn.Auth(nil); ea != nil {
		conn.Close()
		return nil, ea
	}

	pulse := &Client{
		conn:   conn,
		hooker: NewHooker(),
	}

	pulse.hooker.AddCalls(PulseCalls)
	pulse.hooker.AddTypes(PulseTypes)

	return pulse, nil
}

// Register connects an object to the pulseaudio events hooks it implements.
// If the object declares any of the method in the On... interfaces list, it
// will be registered to receive those events.
//
func (pulse *Client) Register(obj interface{}) (errs []error) {
	tolisten := pulse.hooker.Register(obj)
	for _, name := range tolisten {
		e := pulse.ListenForSignal(name, []dbus.ObjectPath{})
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
	c := make(chan *dbus.Signal, 10)
	pulse.conn.Signal(c)

	for s := range c {
		pulse.DispatchSignal(s)
	}
}

// DispatchSignal forwards a signal event to the registered clients.
//
func (pulse *Client) DispatchSignal(s *dbus.Signal) {
	name := strings.TrimPrefix(string(s.Name), DbusIf+".")
	if name != s.Name { // dbus interface matched.
		if pulse.hooker.Call(name, s) {
			return // signal was defined (even if no clients are connected).
		}
	}
	log.Println("unknown signal", s.Name, s.Path)
}

//
//------------------------------------------------------------[ DBUS METHODS ]--

// ListenForSignal registers a new event to listen.
//
func (pulse *Client) ListenForSignal(name string, paths []dbus.ObjectPath) error {
	args := []interface{}{DbusIf + "." + name, paths}
	return pulse.Core().Call("ListenForSignal", 0, args...).Err
}

// StopListeningForSignal unregisters an listened event.
//
func (pulse *Client) StopListeningForSignal(name string) error {
	return pulse.Core().Call("StopListeningForSignal", 0, DbusIf+"."+name).Err
}

//
//-----------------------------------------------------[ CALLBACK INTERFACES ]--

type OnFallbackSinkUpdated interface {
	FallbackSinkUpdated(dbus.ObjectPath)
}

type OnFallbackSinkUnset interface {
	FallbackSinkUnset()
}

type OnNewSink interface {
	NewSink(dbus.ObjectPath)
}

type OnSinkRemoved interface {
	SinkRemoved(dbus.ObjectPath)
}

type OnNewPlaybackStream interface {
	NewPlaybackStream(dbus.ObjectPath)
}

type OnPlaybackStreamRemoved interface {
	PlaybackStreamRemoved(dbus.ObjectPath)
}

type OnDeviceVolumeUpdated interface {
	DeviceVolumeUpdated(dbus.ObjectPath, []uint32)
}
type OnDeviceMuteUpdated interface {
	DeviceMuteUpdated(dbus.ObjectPath, bool)
}

type OnStreamVolumeUpdated interface {
	StreamVolumeUpdated(dbus.ObjectPath, []uint32)
}

type OnStreamMuteUpdated interface {
	StreamMuteUpdated(dbus.ObjectPath, bool)
}

//
//--------------------------------------------------------[ CALLBACK METHODS ]--

// PulseCalls defines callbacks methods to call the matching object method with
// type-asserted arguments.
// Public so it can be hacked before the first Register.
//
var PulseCalls = Calls{
	"FallbackSinkUpdated":   func(m Msg) { m.O.(OnFallbackSinkUpdated).FallbackSinkUpdated(m.P) },
	"FallbackSinkUnset":     func(m Msg) { m.O.(OnFallbackSinkUnset).FallbackSinkUnset() },
	"NewSink":               func(m Msg) { m.O.(OnNewSink).NewSink(m.P) },
	"SinkRemoved":           func(m Msg) { m.O.(OnSinkRemoved).SinkRemoved(m.P) },
	"NewPlaybackStream":     func(m Msg) { m.O.(OnNewPlaybackStream).NewPlaybackStream(m.D[0].(dbus.ObjectPath)) },
	"PlaybackStreamRemoved": func(m Msg) { m.O.(OnPlaybackStreamRemoved).PlaybackStreamRemoved(m.D[0].(dbus.ObjectPath)) },
	"Device.VolumeUpdated":  func(m Msg) { m.O.(OnDeviceVolumeUpdated).DeviceVolumeUpdated(m.P, m.D[0].([]uint32)) },
	"Device.MuteUpdated":    func(m Msg) { m.O.(OnDeviceMuteUpdated).DeviceMuteUpdated(m.P, m.D[0].(bool)) },
	"Stream.VolumeUpdated":  func(m Msg) { m.O.(OnStreamVolumeUpdated).StreamVolumeUpdated(m.P, m.D[0].([]uint32)) },
	"Stream.MuteUpdated":    func(m Msg) { m.O.(OnStreamMuteUpdated).StreamMuteUpdated(m.P, m.D[0].(bool)) },
}

// PulseTypes defines interface types for events to register.
// Public so it can be hacked before the first Register.
//
var PulseTypes = map[string]reflect.Type{
	"FallbackSinkUpdated":   reflect.TypeOf((*OnFallbackSinkUpdated)(nil)).Elem(),
	"FallbackSinkUnset":     reflect.TypeOf((*OnFallbackSinkUnset)(nil)).Elem(),
	"NewSink":               reflect.TypeOf((*OnNewSink)(nil)).Elem(),
	"SinkRemoved":           reflect.TypeOf((*OnSinkRemoved)(nil)).Elem(),
	"NewPlaybackStream":     reflect.TypeOf((*OnNewPlaybackStream)(nil)).Elem(),
	"PlaybackStreamRemoved": reflect.TypeOf((*OnPlaybackStreamRemoved)(nil)).Elem(),
	"Device.VolumeUpdated":  reflect.TypeOf((*OnDeviceVolumeUpdated)(nil)).Elem(),
	"Device.MuteUpdated":    reflect.TypeOf((*OnDeviceMuteUpdated)(nil)).Elem(),
	"Stream.VolumeUpdated":  reflect.TypeOf((*OnStreamVolumeUpdated)(nil)).Elem(),
	"Stream.MuteUpdated":    reflect.TypeOf((*OnStreamMuteUpdated)(nil)).Elem(),
}

//
//------------------------------------------------------------------[ COMMON ]--

// serverLookup asks the main service for the location of the real service.
// It's the only thing the pulseaudio service do on the main session dbus.
// On my system, it returns  "unix:path=/run/user/1000/pulse/dbus-socket"
//
func serverLookup() (string, error) {
	conn, es := dbus.SessionBus()
	if es != nil {
		return "", es
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
	*dbus.Object
	prefix string
}

// NewObject creates a dbus Object with properties access methods.
//
func NewObject(conn *dbus.Conn, interf string, path dbus.ObjectPath) *Object {
	return &Object{conn.Object(interf, path), interf}
}

// Get queries an object property and return it as a raw dbus.Variant.
//
func (dev *Object) Get(property string) (dbus.Variant, error) {
	return dev.GetProperty(dev.location(property))
}

// GetValue queries an object property and set it to dest.
// dest must be of the same type as returned data for the method.
//
func (dev *Object) GetValue(name string, dest interface{}) (e error) {
	v, e := dev.Get(name)
	if e != nil {
		return e
	}
	return cast(v.Value(), dest)
}

// Set updates the given object property with value.
//
func (dev *Object) Set(property string, value interface{}) error {
	return dev.SetProperty(dev.location(property), dbus.MakeVariant(value))
}

func (dev *Object) location(property string) string {
	return dev.prefix + "." + property
}

//
//---------------------------------------------------[ GET CASTED PROPERTIES ]--

// Bool queries an object property and return it as bool.
//
func (dev *Object) Bool(name string) (val bool, e error) {
	e = dev.GetValue(name, &val)
	return val, e
}

// Uint32 queries an object property and return it as uint32.
//
func (dev *Object) Uint32(name string) (val uint32, e error) {
	e = dev.GetValue(name, &val)
	return val, e
}

// Uint64 queries an object property and return it as uint64.
//
func (dev *Object) Uint64(name string) (val uint64, e error) {
	e = dev.GetValue(name, &val)
	return val, e
}

// String queries an object property and return it as string.
//
func (dev *Object) String(name string) (val string, e error) {
	e = dev.GetValue(name, &val)
	return val, e
}

// ObjectPath queries an object property and return it as string.
//
func (dev *Object) ObjectPath(name string) (val dbus.ObjectPath, e error) {
	e = dev.GetValue(name, &val)
	return val, e
}

// ListUint32 queries an object property and return it as []uint32.
//
func (dev *Object) ListUint32(name string) (val []uint32, e error) {
	e = dev.GetValue(name, &val)
	return val, e
}

// ListString queries an object property and return it as []string.
//
func (dev *Object) ListString(name string) (val []string, e error) {
	e = dev.GetValue(name, &val)
	return val, e
}

// ListPath queries an object property and return it as []dbus.ObjectPath.
//
func (dev *Object) ListPath(name string) (val []dbus.ObjectPath, e error) {
	e = dev.GetValue(name, &val)
	return val, e
}

// MapString queries an object property and return it as map[string]string.
//
func (dev *Object) MapString(name string) (map[string]string, error) {
	var asbytes map[string][]byte
	e := dev.GetValue(name, &asbytes)
	if e != nil {
		return nil, e
	}
	val := make(map[string]string)
	for k, v := range asbytes {
		if len(v) > 0 {
			val[k] = string(v[:len(v)-1]) // remove \0 at end.
		}
	}
	return val, nil
}

func cast(src interface{}, dest interface{}) (e error) {
	switch c := dest.(type) {
	case *bool:
		*c = src.(bool)

	case *uint32:
		*c = src.(uint32)

	case *uint64:
		*c = src.(uint64)

	case *string:
		*c = src.(string)

	case *dbus.ObjectPath:
		*c = src.(dbus.ObjectPath)

	case *[]uint32:
		*c = src.([]uint32)

	case *[]string:
		*c = src.([]string)

	case *[]dbus.ObjectPath:
		*c = src.([]dbus.ObjectPath)

	case *map[string][]byte:
		*c = src.(map[string][]byte)

	default:
		e = errors.New("cast fail")
	}
	return
}

// SetProperty calls org.freedesktop.DBus.Properties.Set on the given object.
// The property name must be given in interface.member notation.
//
// TODO: Should be moved to the dbus api.
//
func (o *Object) SetProperty(p string, v dbus.Variant) error {
	idx := strings.LastIndex(p, ".")
	if idx == -1 || idx+1 == len(p) {
		return errors.New("dbus: invalid property " + p)
	}

	iface := p[:idx]
	prop := p[idx+1:]

	err := o.Call("org.freedesktop.DBus.Properties.Set", 0, iface, prop, v).Err

	if err != nil {
		return err
	}

	return nil
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

// Hook defines a list of objects indexed by the methods they implement.
// An object can be referenced multiple times.
// If an object declares all methods, he will be referenced in every field.
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
