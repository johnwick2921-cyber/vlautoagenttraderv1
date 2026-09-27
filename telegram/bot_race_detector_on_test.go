//go:build race

package telegram

// raceDetectorOn: this test binary was built with -race (the go command sets
// the "race" build tag). Tests that prove nothing without the detector skip
// when it is off.
const raceDetectorOn = true
