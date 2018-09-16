/*
Package pulseaudio controls a pulseaudio server through its Dbus interface.

This is a pure go binding for the pulseaudio Dbus interface.

Note that you will have to enable the dbus module of your pulseaudio server.
This can now be done with the LoadModule function.

or by adding this line in /etc/pulse/default.pa
  load-module module-dbus-protocol

(if system-wide daemon is used, instead edit /etc/pulse/system.pa )


Registering methods to listen to signals

Create a type that declares any methods matching the pulseaudio interface.
  Methods will be detected when you register the object:         Register(myobject)
  Methods will start to receive events when you start the loop:  Listen()

Instead of declaring a big interface and forcing clients to provide every method
they don't need, this API use a flexible interface system.
The callback interface is provided by a list of single method interfaces and
only those needed will have to be implemented.

You can register multiple clients at any time on the same pulseaudio session.
This allow you to split the callback logic of your program and have some parts
(un)loadable like a client GUI.

See types On... for the list of callback methods that can be used.


Create a client object with some callback methods to register:
	type Client struct {
		// The reference to the object isn't needed for the callbacks to work,
		// but it will certainly be useful at some point in your code.
		*pulseaudio.Client
	}

	func (cl *Client) NewPlaybackStream(path dbus.ObjectPath) {
		log.Println("NewPlaybackStream", path)
	}

	func (cl *Client) PlaybackStreamRemoved(path dbus.ObjectPath) {
		log.Println("PlaybackStreamRemoved", path)
	}

	func (cl *Client) DeviceVolumeUpdated(path dbus.ObjectPath, values []uint32) {
		log.Println("device volume", path, values)
	}

	func (cl *Client) StreamVolumeUpdated(path dbus.ObjectPath, values []uint32) {
		log.Println("stream volume", path, values)
	}


Register your object and start the listening loop:
	pulse, e := pulseaudio.New()
	if e != nil {
		log.Panicln("connect", e)
	}

	client := &Client{pulse}
	pulse.Register(client)

	pulse.Listen()

Get properties

There are way too many properties to have a dedicated method for each of them.
  Object  Object implementing the property you need.
    Core
    Device
    Stream
  Name    Name of the property
  Type    Type of the property.
    bool          Bool
    uint32        Uint32
    uint64        Uint64
    string        String
    ObjectPath    Path
    []uint32      ListUint32
    []string      ListString
    []ObjectPath  ListPath

First you need to get the object implementing the property you need.
Then you will have to call the method matching the type of returned data for the
property you want to get. See the example.

Set properties

Properties with the tag RW can also be set.

Pulseaudio Dbus documentation

http://www.freedesktop.org/wiki/Software/PulseAudio/Documentation/Developer/Clients/DBus/

Dbus documentation was copied to provide some useful informations here.
Still valid in august 2018.

Check the upstream source for updates or more informations.

*/
package pulseaudio

import "github.com/godbus/dbus"

// Core controls the pulseaudio core.
//
// Properties list:
//   Boolean
//     !IsLocal   This per-client property can be used to find out whether the client is connected to a local server.
//
//   Uint32
//     !InterfaceRevision        The "major" version of the main D-Bus interface is
//                               embedded in the interface name: org.PulseAudio.Core1.
//                               When changes are made that break compatibility between
//                               old clients and new servers the major version is
//                               incremented. This property tells the "minor" version,
//                               that is, when new features are added to the interface,
//                               this version number is incremented so that new clients
//                               can check if the server they talk to supports the new
//                               features. This documentation defines revision 0.
//                               Extensions are versioned separately (i.e. they have
//                               their own major and minor version numbers).
//     !DefaultSampleFormat  RW  The default sample format that is used when
//                               initializing a device and the configuration information
//                               doesn't specify the desired sample format.
//     !DefaultSampleRate    RW  The default sample rate that is used when initializing
//                               a device and the configuration information doesn't
//                               specify the desired sample rate.
//
//   String
//     Name        The server name. At the time of writing no competing implementations
//                 have appeared, so the expected name is "pulseaudio".
//     Version     The server version string, for example "0.9.17".
//     Username    The username that the server is running under.
//     Hostname    The hostname of the machine the server is running on.
//
//   ObjectPath
//     !FallbackSink    RW  When a new playback stream is created and there is no other
//                          policy about to which sink the stream should be connected,
//                          the fallback sink is selected. This property doesn't exist
//                          if there's no sink selected as the fallback sink.
//     !FallbackSource  RW  When a new record stream is created and there is no other
//                          policy about to which source the stream should be connected,
//                          the fallback source is selected. This property doesn't
//                          exist if there's no source selected as the fallback source.
//     !MyClient            This property has a different value for each client: it
//                          tells the reading client the client object that is
//                          assigned to its connection.
//
//   ListUint32
//     !DefaultChannels  RW  The default channel map that is used when initializing a
//                           device and the configuration information doesn't specify
//                           the desired channel map. The default channel count can be
//                           inferred from this. The channel map is expressed as a
//                           list of channel positions,
//
//   ListString
//     Extensions    All available server extension interfaces. Each extension interface
//                   defines an unique string that clients can search from this array.
//                   The string should contain a version part so that if backward
//                   compatibility breaking changes are made to the interface,
//                   old clients don't detect the new interface at all,
//                   or both old and new interfaces can be provided.
//                   The string is specific to the D-Bus interface of the extension,
//                   so if an extension module offers access through both the C API
//                   and D-Bus, the interfaces can be updated independently.
//                   The strings are intended to follow the structure and restrictions
//                   of D-Bus interface names, but that is not enforced.
//                   The clients should treat the strings as opaque identifiers.
//
//   ListPath
//     Cards              All currently available cards.
//     Sinks              All currently available sinks.
//     Sources            All currently available sources.
//     !PlaybackStreams   All current playback streams.
//     !RecordStreams     All current record streams.
//     Samples            All currently loaded samples.
//     Modules            All currently loaded modules.
//     Clients            All currently connected clients.
//
func (pulse *Client) Core() *Object {
	return NewObject(pulse.conn, DbusInterface, DbusPath)
}

// Device controls a pulseaudio device.
//
// Methods list:
//   Suspend          Suspends or unsuspends the device.
//     bool:            True to suspend, false to unsuspend.
//   !GetPortByName   Find the device port with the given name.
//     string:          Port name
//     out: ObjectPath: Device port object
//
// Properties list:
//   Boolean
//     !HasFlatVolume                  Whether or not the device is configured to use the
//                                    "flat volume" logic, that is, the device volume follows
//                                    the maximum volume of all connected streams.
//                                    Currently this is not implemented for sources, so for
//                                    them this property is always false.
//     !HasConvertibleToDecibelVolume  If this is true, the volume values of the Volume property
//                                     can be converted to decibels with pa_sw_volume_to_dB().
//                                     If you want to avoid the C API, the function does
//                                     the conversion as follows:
//                                       If input = 0, then output = -200.0
//                                       Otherwise output = 20 * log10((input / 65536)3)
//     Mute     RW                     Whether or not the device is currently muted.
//     !HasHardwareVolume              Whether or not the device volume controls the hardware volume.
//     !HasHardwareMute                Whether or not muting the device controls the hardware mute state.
//     !HasDynamicLatency              Whether or not the device latency can be adjusted
//                                     according to the needs of the connected streams.
//     !IsHardwareDevice               Whether or not this device is a hardware device.
//     !IsNetworkDevice                Whether or not this device is a network device.
//
//   Uint32
//     Index          The device index. Sink and source indices are separate, so it's perfectly
//                    normal to have two devices with the same index: the other device is a
//                    sink and the other is a source.
//     !SampleFormat  The sample format of the device.
//     !SampleRate    The sample rate of the device.
//     !BaseVolume    The volume level at which the device doesn't perform any
//                    amplification or attenuation.
//     !VolumeSteps   If the device doesn't support arbitrary volume values, this property
//                    tells the number of possible volume values.
//                    Otherwise this property has value 65537.
//     State          The current state of the device.
//
//   Uint64
//     !ConfiguredLatency  The latency in microseconds that device has been configured to.
//     Latency             The length of queued audio in the device buffer. Not all devices
//                         support latency querying; in those cases this property does not exist.
//
//   String
//     Name    The device name.
//     Driver  The driver that implements the device object.
//            This is usually expressed as a source code file name, for example "module-alsa-card.c".
//
//   ObjectPath
//     !OwnerModule      The module that owns this device. It's not guaranteed that any module
//                       claims ownership; in such case this property does not exist.
//     Card              The card that this device belongs to. Not all devices are part of cards;
//                       in those cases this property does not exist.
//     !ActivePort  RW   The currently active device port.
//                       This property doesn't exist if the device does not have any ports.
//
//   ListUint32
//     Channels    The channel map of the device. The channel count can be inferred from this.
//                 The channel map is expressed as a list of channel positions.
//     Volume  RW  The volume of the device. The array is matched against the Channels property:
//                 the first array element is the volume of the first channel in the Channels
//                 property, and so on. There are two ways to adjust the volume:
//                   You can either adjust the overall volume by giving a single-value array,
//                   or you can precisely control the individual channels by passing an array
//                   containing a value for each channel.
//
//   ListPath
//     Ports     All available device ports. May be empty.
//
//
//   MapString
//     !PropertyList       The device's property list.
//
func (pulse *Client) Device(sink dbus.ObjectPath) *Object {
	return NewObject(pulse.conn, DbusInterface+".Device", sink)
}

// Stream controls a pulseaudio stream.
//
// Methods list:
//   Kill    Kills the stream.
//   Move    Moves the stream to another device.
//     ObjectPath: The device to move to.
//
// Properties list:
//
//  Boolean
//    !VolumeWritable   Whether or not the Volume property can be set. Note that read-only volumes can still change, clients just can't control them.
//    Mute  RW        Whether or not the stream is currently muted. Record streams don't currently support muting, so this property exists for playback streams only for now.
//
//  Uint32
//    Index     The stream index. Playback and record stream indices are separate, so it's perfectly normal to have two streams with the same index: the other stream is a playback stream and the other is a record stream.
//    !SampleFormat   The sample format of the stream. See [[Software/PulseAudio/Documentation/Developer/Clients/DBus/Enumerations]] for the list of possible values.
//    !SampleRate     The sample rate of the stream.
//
//  Uint64
//    !BufferLatency      The length of buffered audio in microseconds that is not at the device yet/anymore.
//    !DeviceLatency      The length of buffered audio in microseconds at the device.
//
//  String
//    Driver   The driver that implements the stream object. This is usually expressed as a source code file name, for example "protocol-native.c".
//    !ResampleMethod     The resampling algorithm that is used to convert the stream audio data to/from the device's sample rate.
//
//  ObjectPath
//    !OwnerModule   The module that owns this stream. It's not guaranteed that any module claims ownership; in such case this property does not exist.
//    Client    The client whose stream this is. Not all streams are created by clients, in those cases this property does not exist.
//    Device   The device this stream is connected to.
//
//  ListUint32
//    Channels   The channel map of the stream. The channel count can be inferred from this. The channel map is expressed as a list of channel positions, see [[Software/PulseAudio/Documentation/Developer/Clients/DBus/Enumerations]] for the list of possible channel position values.
//    Volume  RW   The volume of the stream. The array is matched against the Channels property: the first array element is the volume of the first channel in the Channels property, and so on.
//                 There are two ways to adjust the volume. You can either adjust the overall volume by giving a single-value array, or you can precisely control the individual channels by passing an array containing a value for each channel.
//                 The volume can only be written if VolumeWritable is true.
//
//  MapString
//    !PropertyList   The stream's property list.
//
func (pulse *Client) Stream(sink dbus.ObjectPath) *Object {
	return NewObject(pulse.conn, DbusInterface+".Stream", sink)
}

// Client controls a pulseaudio client.
//
func (pulse *Client) Client(sink dbus.ObjectPath) *Object {
	return NewObject(pulse.conn, DbusInterface+".Client", sink)
}
