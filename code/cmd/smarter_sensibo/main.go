package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/jbarratt/smarter_sensibo/code/pkg/sensibo"
)

// round is a simple helper function that rounds to a single digit of precision
func round(val float64) float64 {
	var round float64
	digit := 10 * val
	if _, div := math.Modf(digit); div >= 0.5 {
		round = math.Ceil(digit)
	} else {
		round = math.Floor(digit)
	}
	return round / 10
}

// CToF converts temps as you'd expect
func CToF(c float64) float64 {
	return round(float64((c * 9 / 5) + 32))
}

// FToC converts temps as you'd expect
func FToC(f float64) float64 {
	return round(float64((f - 32) * 5 / 9))
}

// inActiveWindow determines if the service should be currently running
func inActiveWindow(t time.Time) bool {
	if t.Weekday() >= time.Monday && t.Weekday() <= time.Friday {
		if t.Hour() >= 5 && t.Hour() <= 16 {
			return true
		}
	}
	return false
}

// setWarming configures a sensibo client to be in a "SmartMode" configuration which warms a room
func setWarming(sm *sensibo.SmartMode) {

	sm.Enabled = true

	sm.LowTemperatureThreshold = FToC(68.0)
	sm.LowTemperatureState.On = true
	sm.LowTemperatureState.FanLevel = "strong"
	sm.LowTemperatureState.TargetTemperature = 80
	sm.LowTemperatureState.Mode = "heat"

	sm.HighTemperatureThreshold = FToC(74.0)
	sm.HighTemperatureState.On = false
}

// setCooling configures a sensibo client to be in a "SmartMode" configuration which cools a room
func setCooling(sm *sensibo.SmartMode) {
	sm.Enabled = true

	sm.LowTemperatureThreshold = FToC(71)
	sm.LowTemperatureState.On = false

	sm.HighTemperatureThreshold = FToC(74.0)
	sm.HighTemperatureState.On = true
	sm.HighTemperatureState.FanLevel = "strong"
	sm.HighTemperatureState.TargetTemperature = 65
	sm.HighTemperatureState.Mode = "cool"
}

// shutdown configures SmartMode off and also ensures the device is off
func shutdown(pod *sensibo.Pod) {
	pod.AcState.On = false
	pod.SmartMode.Enabled = false
	// the enabled thing doesn't seem to matter, so make sure both states are 'off'
	pod.SmartMode.HighTemperatureState.On = false
	pod.SmartMode.LowTemperatureState.On = false
}

// syncSensibo is what would usually be the main() method
// It's pulled out so it can be shared across CLI and Lambda invocations
func syncSensibo() {

	client := sensibo.NewClient(nil)
	err := client.LoadState()
	if err != nil {
		log.Fatal(err)
	}

	// smartMarsh, _ := json.MarshalIndent(client.State.SmartMode, "", "  ")

	loc, _ := time.LoadLocation("America/Los_Angeles")
	now := time.Now().In(loc)

	if inActiveWindow(now) {
		log.Println("In an active window")
		if CToF(client.State.Measurements.Temperature) < 65.0 {
			log.Println("Under threshold, setting warming mode")
			setWarming(&client.State.SmartMode)
		} else if CToF(client.State.Measurements.Temperature) > 73.0 {
			log.Println("Temp over threshold, entering cooling mode")
			setCooling(&client.State.SmartMode)
		} else {
			log.Printf("%0.2f F: Between temp zones, disabling smart mode\n", CToF(client.State.Measurements.Temperature))
			shutdown(&client.State)
		}
	} else {
		log.Println("Outside active window, disabling device")
		shutdown(&client.State)
	}

	err = client.PushState()
	if err != nil {
		log.Fatal(err)
	}
}

func HandleRequest(ctx context.Context, e events.CloudWatchEvent) error {
	syncSensibo()
	// print this so the metric filter can count it
	fmt.Println("SENSIBO_SETTING_SUCCESS")
	return nil
}

func main() {
	// detect if this is running from within a lambda
	_, inLambda := os.LookupEnv("AWS_LAMBDA_FUNCTION_NAME")
	if inLambda {
		lambda.Start(HandleRequest)
	} else {
		syncSensibo()
	}
}
