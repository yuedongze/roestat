package roest

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// MQTTBroker is the MQTT-over-secure-WebSocket endpoint used by the web app.
const MQTTBroker = "wss://client.roestcoffee.com:8083/mqtt"

// LiveEvent is a roast event (type 0 = charge, 1 = drop, 4 = dry end, 5 = FC).
type LiveEvent struct {
	Msec int `json:"msec"`
	Type int `json:"type"`
}

// LivePhases builds the roast-phase breakdown for a roast in progress from the
// live event list and the current elapsed time. Unlike Log.Phases it is lenient:
// it returns only the phases reached so far, with the current phase running up to
// nowMsec (or the drop event, once dropped), so the bar grows as the roast
// advances. ok is false before any elapsed time exists.
func LivePhases(events []LiveEvent, nowMsec int) ([]RoastPhase, bool) {
	dryEnd, haveDry := liveEventMsec(events, 4)
	firstCrack, haveFC := liveEventMsec(events, 5)
	end := nowMsec
	if drop, ok := liveEventMsec(events, 1); ok && drop > 0 {
		end = drop
	}
	return buildPhases(dryEnd, firstCrack, end, haveDry, haveFC)
}

// liveEventMsec returns the elapsed-msec of the first event of the given type.
func liveEventMsec(events []LiveEvent, typ int) (int, bool) {
	for _, e := range events {
		if e.Type == typ {
			return e.Msec, true
		}
	}
	return 0, false
}

// SensorReading is one raw sensor value from the temperature_sensors array.
type SensorReading struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}

// LiveData mirrors a REST datapoint but arrives over MQTT ~once per second.
type LiveData struct {
	Msec               int             `json:"msec"`
	BT                 float64         `json:"bt"`
	ET                 float64         `json:"et"`
	Target             float64         `json:"target"`
	Fan                float64         `json:"fan"`
	Heat               float64         `json:"heat"`
	RPM                float64         `json:"rpm"`
	DrumTemp           float64         `json:"drum_temp"`
	InletTemp          float64         `json:"inlet_temp"`
	AirPressure        float64         `json:"air_pressure"`
	Crack              int             `json:"crack"`
	PCBTemperature     float64         `json:"pcb_temperature"`
	PCBHumidity        float64         `json:"pcb_humidity"`
	TemperatureSensors []SensorReading `json:"temperature_sensors"`
}

// LivePayload is a full live message published to roest/p2000/<machine_id>/v2.
type LivePayload struct {
	BatchUUID       string      `json:"batch_uuid"`
	ChargeTimestamp int64       `json:"charge_timestamp"`
	ProfileID       string      `json:"profile_id"`
	FirmwareVersion string      `json:"firmware_version"`
	Events          []LiveEvent `json:"events"`
	Data            LiveData    `json:"data"`
}

// LiveClient is a subscription to one machine's live MQTT feed. Received
// payloads are delivered on Messages; the buffer drops rather than blocks if
// the consumer falls behind.
type LiveClient struct {
	client   mqtt.Client
	Messages chan LivePayload
}

// ConnectLive connects to the broker and subscribes to the machine's topic.
func ConnectLive(m Machine) (*LiveClient, error) {
	lc := &LiveClient{Messages: make(chan LivePayload, 64)}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(MQTTBroker)
	opts.SetClientID(fmt.Sprintf("roestat-%d-%d", m.ID, os.Getpid()))
	opts.SetUsername(m.MQTTConfig.Username)
	opts.SetPassword(m.MQTTConfig.SubscribePassword)
	opts.SetTLSConfig(&tls.Config{InsecureSkipVerify: true}) // wss on :8083
	opts.SetConnectTimeout(15 * time.Second)
	opts.SetKeepAlive(60 * time.Second)
	opts.SetAutoReconnect(true)

	client := mqtt.NewClient(opts)
	if tok := client.Connect(); tok.Wait() && tok.Error() != nil {
		return nil, fmt.Errorf("mqtt connect: %w", tok.Error())
	}

	handler := func(_ mqtt.Client, msg mqtt.Message) {
		var p LivePayload
		if err := json.Unmarshal(sanitizeJSON(msg.Payload()), &p); err != nil {
			return
		}
		select {
		case lc.Messages <- p:
		default: // consumer behind; drop this sample
		}
	}
	if tok := client.Subscribe(m.MQTTConfig.Topic, 0, handler); tok.Wait() && tok.Error() != nil {
		client.Disconnect(100)
		return nil, fmt.Errorf("mqtt subscribe: %w", tok.Error())
	}

	lc.client = client
	return lc, nil
}

// Close disconnects from the broker.
func (lc *LiveClient) Close() {
	if lc.client != nil {
		lc.client.Disconnect(250)
	}
}
