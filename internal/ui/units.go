package ui

// tempUnit is the temperature unit used for every displayed value. It's toggled
// with the "f" key and read at render time, so flipping it reunits all views.
type tempUnit int

const (
	unitCelsius tempUnit = iota
	unitFahrenheit
)

// currentUnit is process-wide display state. The TUI re-renders from it every
// frame; toggleUnit therefore reunits every gauge, chart and table at once.
var currentUnit = unitCelsius

// toggleUnit flips between Celsius and Fahrenheit.
func toggleUnit() {
	if currentUnit == unitCelsius {
		currentUnit = unitFahrenheit
	} else {
		currentUnit = unitCelsius
	}
}

// convTemp converts an absolute temperature in °C to the current unit.
func convTemp(c float64) float64 {
	if currentUnit == unitFahrenheit {
		return c*9/5 + 32
	}
	return c
}

// convDelta converts a temperature difference/rate in °C to the current unit —
// no offset, since a 10°C rise is an 18°F rise. Used for Rate of Rise.
func convDelta(c float64) float64 {
	if currentUnit == unitFahrenheit {
		return c * 9 / 5
	}
	return c
}

// tempSuffix is the unit label for absolute temperatures ("°C"/"°F").
func tempSuffix() string {
	if currentUnit == unitFahrenheit {
		return "°F"
	}
	return "°C"
}

// rorSuffix is the unit label for Rate of Rise ("°C/min"/"°F/min").
func rorSuffix() string {
	if currentUnit == unitFahrenheit {
		return "°F/min"
	}
	return "°C/min"
}
