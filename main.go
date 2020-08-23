package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/kelvins/sunrisesunset"
)

const baseURL = "https://api.lifx.com/v1/lights/"

//LIFX base structs
type coordinates struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type dusk struct {
	ColorStart   string `json:"colorStart"`
	ColorEnd     string `json:"colorEnd"`
	Steps        int    `json:"steps"`
	Duration     int    `json:"duration"`
	TurnOffRange int    `json:"turnOffRange"`
}

type device struct {
	Coordinates coordinates `json:"coordinates"`
	ID          string      `json:"id"`
	Name        string      `json:"name"`
}

// configuration json
type config struct {
	Token        string   `json:"token"`
	DefaultColor string   `json:"defaultColor"`
	Dusk         dusk     `json:"dusk"`
	Devices      []device `json:"devices"`
}

// LIFX "state"
type state struct {
	Selector   string  `json:"selector"`
	Power      string  `json:"power"`
	Color      string  `json:"color"`
	Brightness float32 `json:"brightness"`
	Duration   float32 `json:"duration"`
	Fast       bool    `json:"fast"`
}

// LIFX "multiple states" format (up to 50)
type states struct {
	States []state `json:"states"`
}

var myConfig config

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/bulb/{selector}/{action}", BulbHandler).Methods("GET")
	r.HandleFunc("/test/{text}", TextHandler).Methods("GET")
	http.Handle("/", r)

	srv := &http.Server{
		Handler: r,
		Addr:    "127.0.0.1:8000",
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 30 * time.Second,
		ReadTimeout:  30 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}

func initialiseConfig(configType string) {
	var myConfig config
	var myDevice device
	var myCoordinates coordinates

	if configType == "new" {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter LIFX token: ")
		token, _ := reader.ReadString('\n')
		myConfig.Token = strings.Replace(token, "\n", "", -1)

		fmt.Print("Enter default light color (Kelvins): ")
		defaultColor, _ := reader.ReadString('\n')
		defaultColor = strings.Replace(defaultColor, "\n", "", -1)
		_, errDefaultColor := strconv.Atoi(strings.Replace(defaultColor, "\n", "", -1))
		errorLog(errDefaultColor, "Default color must be an integer")
		myConfig.DefaultColor = defaultColor

		fmt.Print("Enter dusk light color to start with (Kelvins): ")
		duskColorStart, _ := reader.ReadString('\n')
		_, errColorStart := strconv.Atoi(strings.Replace(duskColorStart, "\n", "", -1))
		errorLog(errColorStart, "Dusk color must be an integer")
		myConfig.Dusk.ColorStart = strings.Replace(duskColorStart, "\n", "", -1)

		fmt.Print("Enter dusk light color to end with (Kelvins): ")
		duskColorEnd, _ := reader.ReadString('\n')
		_, errColorEnd := strconv.Atoi(strings.Replace(duskColorEnd, "\n", "", -1))
		errorLog(errColorEnd, "Dusk color must be an integer")
		myConfig.Dusk.ColorEnd = strings.Replace(duskColorEnd, "\n", "", -1)

		fmt.Print("Enter dusk steps: ")
		duskSteps, _ := reader.ReadString('\n')
		mySteps, errSteps := strconv.Atoi(strings.Replace(duskSteps, "\n", "", -1))
		errorLog(errSteps, "Dusk stepts must be an integer")
		myConfig.Dusk.Steps = mySteps

		fmt.Print("Enter dusk duration (minutes): ")
		duskDuration, _ := reader.ReadString('\n')
		myDuration, errDuration := strconv.Atoi(strings.Replace(duskDuration, "\n", "", -1))
		errorLog(errDuration, "Dusk duration must be an integer")
		myConfig.Dusk.Duration = myDuration

		fmt.Print("Enter dusk turn off range (minutes): ")
		turnOffRange, _ := reader.ReadString('\n')
		myRange, errRange := strconv.Atoi(strings.Replace(turnOffRange, "\n", "", -1))
		errorLog(errRange, "Dusk stepts turn off range be an integer")
		myConfig.Dusk.TurnOffRange = myRange

		myConfig.setConfig()
	} else if configType == "device" {
		myConfig.getConfig()

		reader := bufio.NewReader(os.Stdin)

		fmt.Print("Enter device or group name: ")
		mySelector, _ := reader.ReadString('\n')
		myDevice.Name = strings.Replace(mySelector, "\n", "", -1)

		fmt.Print("Enter device or group ID: ")
		myID, _ := reader.ReadString('\n')
		myDevice.ID = strings.Replace(myID, "\n", "", -1)

		fmt.Print("Enter device or group latitude: ")
		myLat, _ := reader.ReadString('\n')
		myLatitude, errLatitude := strconv.ParseFloat(strings.Replace(myLat, "\n", "", -1), 64)
		errorLog(errLatitude, "Latitude must be a float")
		myCoordinates.Latitude = myLatitude

		fmt.Print("Enter device or group longitude: ")
		myLng, _ := reader.ReadString('\n')
		myLongitude, errLongitude := strconv.ParseFloat(strings.Replace(myLng, "\n", "", -1), 64)
		errorLog(errLongitude, "Longitude must be a float")
		myCoordinates.Longitude = myLongitude

		myDevice.Coordinates = myCoordinates

		myConfig.Devices = append(myConfig.Devices, myDevice)

		myConfig.setConfig()
	} else {
		fmt.Println("Parameter not valid")
	}
}

func (myConfig *config) getConfig() {
	data, err := ioutil.ReadFile("./config.json")
	if err != nil {
		errorLog(err, "No configuration file found")
		panic(err)
	}
	err = json.Unmarshal(data, &myConfig)
	if err != nil {
		errorLog(err, "Configuration file not valid")
		panic(err)
	}
}

func (myConfig *config) setConfig() {
	data, err := json.Marshal(myConfig)

	err = ioutil.WriteFile("config.json", data, 0644)
	if err != nil {
		errorLog(err, "SetConfig error")
		return
	}
}

func takeAction(action string, myDevice device) {
	switch action {
	case "state":
		allStates := getState()
		fmt.Println(allStates)
	case "toggle":
		toggle(myDevice.ID)
	case "on":
		setPower(myDevice.ID, "on", 1)
	case "off":
		setPower(myDevice.ID, "off", 0)
	case "dusk":
		startDusk(myDevice)
	case "duskBasic":
		startDuskBasic(myDevice)
	case "duskBeta":
		startDuskBeta(myDevice)
	default:
		fmt.Println("No action found for " + action)
	}
}

func setPower(selector string, power string, brightness float32) {
	myState := state{selector, power, "keilvin:" + myConfig.DefaultColor, brightness, 0, false}
	setState(myState)
}

func toggle(selector string) {
	client := &http.Client{}

	req, err := http.NewRequest("POST", baseURL+selector+"/toggle", nil)
	if err != nil {
		errorLog(err, "Bad 'toggle' format")
	}

	req.Header.Set("Authorization", "Bearer "+myConfig.Token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		errorLog(err, "Bad 'toggle' request")
	}

	defer resp.Body.Close()
}

func startDusk(myDevice device) {
	p := sunrisesunset.Parameters{
		Latitude:  myDevice.Coordinates.Latitude,
		Longitude: myDevice.Coordinates.Longitude,
		UtcOffset: 2.0,
		Date:      time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.UTC),
	}

	_, sunset, err := p.GetSunriseSunset()

	if err != nil {
		errorLog(err, "Start dusk error")
	}

	if sunset.After(time.Now()) && sunset.Before(time.Now().Local().Add(time.Minute*time.Duration(1))) {
		durationStep := float32(myConfig.Dusk.Duration * 60 / myConfig.Dusk.Steps)
		myColorStart, _ := strconv.ParseInt(myConfig.Dusk.ColorStart, 10, 0)
		myColorEnd, _ := strconv.ParseInt(myConfig.Dusk.ColorEnd, 10, 0)

		for n := 1; n <= myConfig.Dusk.Steps; n++ {
			brightnessLevel := float32(n*2) / 100
			newState := state{myDevice.ID, "on", "kelvin:" + strconv.FormatInt(myColorStart+(myColorEnd-myColorStart)*int64(n)/int64(myConfig.Dusk.Steps), 10), brightnessLevel, durationStep, true}
			setState(newState)
			if n == myConfig.Dusk.Steps {
				midnight := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Add(time.Hour*time.Duration(24)).Day(), 0, 0, 0, 0, time.Local)
				diff := midnight.Sub(time.Now())
				seconds := int(diff.Seconds())
				durationStep = float32(seconds - myConfig.Dusk.TurnOffRange*30 + rand.Intn(myConfig.Dusk.TurnOffRange*60))
			}
			time.Sleep(time.Duration(durationStep * 1000000000))
		}

		setPower(myDevice.ID, "off", 0)
	}
}

func startDuskBasic(myDevice device) {
	p := sunrisesunset.Parameters{
		Latitude:  myDevice.Coordinates.Latitude,
		Longitude: myDevice.Coordinates.Longitude,
		UtcOffset: 2.0,
		Date:      time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.UTC),
	}

	_, sunset, err := p.GetSunriseSunset()

	if err != nil {
		errorLog(err, "Start dusk basic error")
	}

	if sunset.After(time.Now()) && sunset.Before(time.Now().Local().Add(time.Minute*time.Duration(1))) {
		newState := state{myDevice.ID, "on", "kelvin:" + myConfig.Dusk.ColorEnd, 1, float32(myConfig.Dusk.Duration * 60), true}
		setState(newState)
	}
}

func startDuskBeta(myDevice device) {
	p := sunrisesunset.Parameters{
		Latitude:  myDevice.Coordinates.Latitude,
		Longitude: myDevice.Coordinates.Longitude,
		UtcOffset: 2.0,
		Date:      time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.UTC),
	}

	_, sunset, err := p.GetSunriseSunset()

	if err != nil {
		errorLog(err, "Start dusk beta error")
	}

	if sunset.After(time.Now()) && sunset.Before(time.Now().Local().Add(time.Minute*time.Duration(1))) {
		var duskStates []state
		var brightnessLevel float32
		durationStep := float32(myConfig.Dusk.Duration * 60 / myConfig.Dusk.Steps)
		myColorStart, _ := strconv.ParseInt(myConfig.Dusk.ColorStart, 10, 0)
		myColorEnd, _ := strconv.ParseInt(myConfig.Dusk.ColorEnd, 10, 0)

		for n := 1; n <= 8; n++ {
			brightnessLevel = float32(n) / 10
			newState := state{myDevice.ID, "on", "kelvin:" + strconv.FormatInt(myColorStart+(myColorEnd-myColorStart)*int64(n)/int64(myConfig.Dusk.Steps), 10), brightnessLevel, durationStep, true}
			if n == myConfig.Dusk.Steps {
				midnight := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Add(time.Hour*time.Duration(24)).Day(), 0, 0, 0, 0, time.Local)
				diff := midnight.Sub(time.Now())
				seconds := int(diff.Seconds())
				durationStep = float32(seconds - myConfig.Dusk.TurnOffRange*30 + rand.Intn(myConfig.Dusk.TurnOffRange*60))
			}

			duskStates = append(duskStates, newState)
		}

		midnight := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Add(time.Hour*time.Duration(24)).Day(), 0, 0, 0, 0, time.Local)
		diff := midnight.Sub(time.Now())
		seconds := int(diff.Seconds())
		durationStep = float32(seconds - myConfig.Dusk.TurnOffRange*30 + rand.Intn(myConfig.Dusk.TurnOffRange*60))

		stateOn := state{myDevice.ID, "on", "kelvin:" + myConfig.Dusk.ColorEnd, brightnessLevel, durationStep, true}
		duskStates = append(duskStates, stateOn)

		stateOff := state{myDevice.ID, "off", "kelvin:2700", 0, 0, true}
		duskStates = append(duskStates, stateOff)

		var myStates states
		myStates.States = duskStates
		setStates(myDevice.ID, myStates)
	}
}

func nightWakeUp() {
	wakeStart := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 2, 0, 0, 0, time.UTC)
	wakeEnd := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 4, 0, 0, 0, time.UTC)

	if wakeEnd.After(time.Now()) && wakeStart.Before(time.Now().Local().Add(time.Minute*time.Duration(1))) {
		fmt.Println("wake up")
	}
}

func getState() string {
	req, err := http.NewRequest("GET", baseURL+"/all", nil)
	if err != nil {
		errorLog(err, "Bad 'set state' format")
	}

	req.Header.Set("Authorization", "Bearer "+myConfig.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		errorLog(err, "Bad 'set state' request")
	}

	defer resp.Body.Close()

	responseData, err := ioutil.ReadAll(resp.Body)
	errorLog(err, "Error with 'get state'")

	return string(responseData)
}

func setState(myState state) {
	body := strings.NewReader("power=" + myState.Power + ";color=" + fmt.Sprintf("%f", myState.Color) + ";brightness=" + fmt.Sprintf("%f", myState.Brightness) + ";duration=" + fmt.Sprintf("%f", myState.Duration) + ";fast=" + strconv.FormatBool(myState.Fast))

	req, err := http.NewRequest("PUT", baseURL+myState.Selector+"/state", body)
	if err != nil {
		errorLog(err, "Bad 'set state' format")
	}

	req.Header.Set("Authorization", "Bearer "+myConfig.Token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		errorLog(err, "Bad 'set state' request")
	}

	defer resp.Body.Close()
}

//for cycles
func setStates(mySelector string, myStates states) {
	jsonValue, _ := json.Marshal(myStates)
	body := strings.NewReader(string(jsonValue))

	req, err := http.NewRequest("POST", baseURL+mySelector+"/cycle", body)
	if err != nil {
		errorLog(err, "Bad 'set states' format")
	}

	req.Header.Set("Authorization", "Bearer "+myConfig.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		errorLog(err, "Bad 'set states' request")
	}

	defer resp.Body.Close()
}

func errorLog(err error, message string) {
	//todo
	if err != nil {
		fmt.Println(message)
		panic(err)
	}
}

//BulbHandler takes care of controlling a bulb or group of bulbs
func BulbHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	w.WriteHeader(http.StatusOK)

	selectorPtr := vars["selector"]
	actionPtr := vars["action"]

	if _, err := os.Stat("./config.json"); os.IsNotExist(err) {
		fmt.Fprintln(w, "No configuration found")
	} else {
		myConfig.getConfig()

		if selectorPtr == "" {
			fmt.Fprintln(w, "Missing selector")
		} else {
			var selector device
			for i := range myConfig.Devices {
				if myConfig.Devices[i].Name == selectorPtr {
					selector = myConfig.Devices[i]
				}
			}

			if selector.ID != "" {
				takeAction(actionPtr, selector)
				fmt.Fprintln(w, "Should be done by now...")
			} else {
				fmt.Fprintln(w, "Selector not found")
			}
		}
	}
}

//TextHandler just shows that Mux works
func TextHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, vars["text"])
}
