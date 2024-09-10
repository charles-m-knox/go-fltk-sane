package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/adrg/xdg"
	"github.com/pwiecz/go-fltk"
	"gopkg.in/yaml.v3"
)

var (
	forcePortrait  bool
	forceLandscape bool
	portrait       bool
	// The currently selected DPI resolution
	// selectedRes int
	// the globally shared sane connection, once a device has been connected
	// if true, sane has been initialized already
	// saneInitialized bool

	// Data is stored between runs of this application in this yml config file.
	configFilePath string
	appConf        AppConfig
)

type AppConfig struct {
	// The currently selected output directory for scanned images
	SelectedDir      string
	FilenameTemplate string
	Scanners         []ScannerDevice
	// The device that will perform the scanning operation
	Device string
	// Contains all of the options available for the device, as well as the
	// constraints for each option (for example, the "resolution" option may
	// have constraints of 100,200,300 dpi)
	DeviceMap map[string][]string
	// Contains the current settings for the device. Each key is the same
	// as a value from deviceMap, although empty values may get removed before
	// scanning occurs.
	DeviceSettings map[string]string
	// When the app is closed, the data from the activity feed is saved to this
	// variable.
	Log string
}

// Buttons, inputs, widgets, etc that need to be repositioned in a
// responsive manner
var (
	getDevicesBtn *fltk.Button
	directoryBtn  *fltk.Button
	scanBtn       *fltk.Button
	devicesChoice *fltk.Choice
	// The options available for the device, such as resolution - as presented
	// by sane. This is an FLTK choice picker for them.
	optChoice *fltk.Choice
	// The constraints for each device option, as presented by sane. This is an
	// FLTK choice picker for them.
	constChoice   *fltk.Choice
	fileTmplInput *fltk.Input
	activity      *fltk.HelpView
	activityText  string
)

func parseFlags() {
	flag.BoolVar(&forcePortrait, "portrait", false, "force portrait orientation for the interface")
	flag.BoolVar(&forceLandscape, "landscape", false, "force landscape orientation for the interface")
	flag.StringVar(&configFilePath, "f", "", "the config file to write to, instead of the default provided by XDG config directories")
	flag.Parse()
}

func main() {
	// for profiling only:
	// runtime.SetBlockProfileRate(1)
	// go func() {
	// 	log.Println(http.ListenAndServe("127.0.0.1:6060", nil))
	// }()

	parseFlags()

	var err error

	if configFilePath == "" {
		configFilePath, err = xdg.SearchConfigFile("go-fltk-sane/config.yml")
		if err != nil {
			log.Printf("failed to get xdg config dir: %v", err.Error())
		}
	}

	if configFilePath != "" {
		bac, err := os.ReadFile(configFilePath)
		if err != nil {
			log.Printf("config file not readable at %v", configFilePath)
		}

		err = yaml.Unmarshal(bac, &appConf)
		if err != nil {
			log.Printf("config file %v failed to parse: %v", configFilePath, err.Error())
		}

		log.Printf("loaded config: %v", appConf)
	} else {
		if xdg.ConfigHome != "" {
			configFilePath = path.Join(xdg.ConfigHome, "go-fltk-sane", "config.yml")
			log.Printf("using %v for config file path", configFilePath)
		} else {
			log.Println("unable to automatically identify any suitable config dirs; configuration will not be saved")
		}
	}

	portrait, err = isPortrait()
	if err != nil {
		log.Fatalf("failed to determine screen size: %v", err.Error())
	}

	// probably could write this more intelligently later
	windowWidth := WIDTH_LANDSCAPE
	windowHeight := HEIGHT_LANDSCAPE
	if portrait || forcePortrait {
		windowWidth = WIDTH_PORTRAIT
		windowHeight = HEIGHT_PORTRAIT
		portrait = true
	}
	if forceLandscape {
		windowWidth = WIDTH_LANDSCAPE
		windowHeight = HEIGHT_LANDSCAPE
		portrait = false
	}

	win := fltk.NewWindow(windowWidth, windowHeight)
	fltk.SetScheme("gtk+")
	win.SetLabel("Main Window")
	win.Resizable(win)

	getDevicesBtn = fltk.NewButton(0, 0, 0, 0, "Get Devices")
	directoryBtn = fltk.NewButton(0, 0, 0, 0, "Choose directory...")
	scanBtn = fltk.NewButton(0, 0, 0, 0, "Scan")
	devicesChoice = fltk.NewChoice(0, 0, 0, 0)
	optChoice = fltk.NewChoice(0, 0, 0, 0)
	constChoice = fltk.NewChoice(0, 0, 0, 0)
	fileTmplInput = fltk.NewInput(0, 0, 0, 0)
	activity = fltk.NewHelpView(0, 0, 0, 0)

	fileTmplInput.SetCallback(func() {
		f := fileTmplInput.Value()

		ext := getFileType(f)
		if ext == "" {
			fltk.MessageBox("Warning", "This application only supports png, jpg, and pdf formats.")
		}
	})

	scanBtn.SetCallback(func() {
		if appConf.SelectedDir == "" {
			fltk.MessageBox("Error", "A directory has not been chosen. Please presss the Choose Directory button.")
			return
		}
		if appConf.Device == "" {
			fltk.MessageBox("Error", "A device has not been selected. Please refresh the list of devices and choose one.")
			return
		}

		ext := strings.ToLower(getFileType(fileTmplInput.Value()))
		if ext == "" {
			fltk.MessageBox("Error", "This application only supports png, jpg, and pdf formats.")
			return
		}

		// if ext == ".pdf" {
		// selectedResStr, ok := deviceMap["resolution"]
		// if !ok {
		// 	fltk.MessageBox("Error", "A resolution was not specified. Please update your device settings.")
		// 	return
		// }

		// if selectedRes == 0 {
		// 	fltk.MessageBox("Error", "When rendering to a PDF, the DPI must be specifically set (currently it is unset). To resolve this, find the 'resolution' option and set it to a supported value for your device.")
		// 	return
		// } else if selectedRes > 300 {
		// 	i := fltk.ChoiceDialog("Warning: Setting a very high DPI (above 300) may result in high memory usage and it may take a while to save the result. Are you sure you want to proceed?", "Yes", "Cancel")
		// 	if i != 0 {
		// 		Log("canceling operation due to user accepting high DPI warning")
		// 		return
		// 	}
		// }
		// }

		go func() {
			scanBtn.Deactivate()
			scanBtn.SetLabel("Scanning...")
			defer scanBtn.SetLabel("Scan")
			defer scanBtn.Activate()

			file := strings.ReplaceAll(fileTmplInput.Value(), "%t", fmt.Sprint(time.Now().Unix()))
			pathToWrite := path.Join(appConf.SelectedDir, file)

			// if useScanImage {
			// conn.Close()
			Logf("reading image using scanimage binary...")
			out, err := ScanImage(pathToWrite, appConf.DeviceSettings, strings.TrimPrefix(ext, "."), appConf.Device)
			if out != "" {
				Log(out)
			}
			if err != nil {
				fltk.MessageBox("Error", fmt.Sprintf("Failed to scan to file %v: %v", pathToWrite, err.Error()))
				return
			}

			Logf("successfully wrote scanned image/document to %v", pathToWrite)

			// 	return
			// }

			// Logf("reading image using sane lib...")

			// img, err := conn.ReadImage()
			// if err != nil {
			// 	fltk.MessageBox("Error", fmt.Sprintf("Failed to read image: %v", err.Error()))
			// 	return
			// }

			// switch ext {
			// case ".pdf":
			// 	f, err := os.Create(pathToWrite)
			// 	if err != nil {
			// 		fltk.MessageBox("Error", fmt.Sprintf("Failed to create pdf file %v: %v", pathToWrite, err.Error()))
			// 		return
			// 	}
			// 	defer f.Close()

			// 	Logf("preparing pdf...")
			// 	pdf := gofpdf.New("P", "mm", "A4", "")
			// 	defer pdf.Close()
			// 	pdf.AddPage()

			// 	Logf("encoding image for pdf...")
			// 	buf := &bytes.Buffer{}
			// 	err = png.Encode(buf, img)
			// 	if err != nil {
			// 		fltk.MessageBox("Error", fmt.Sprintf("Failed to convert to png: %v", err.Error()))
			// 		return
			// 	}

			// 	imageInfo := pdf.RegisterImageOptionsReader("image", gofpdf.ImageOptions{
			// 		ImageType: "PNG",
			// 		ReadDpi:   false,
			// 	}, buf)

			// 	imageInfo.SetDpi(float64(selectedRes))

			// 	Logf("imageInfo: h=%v w=%v", imageInfo.Height(), imageInfo.Width())

			// 	pdf.ImageOptions("image", 0, 0, -1, -1, false, gofpdf.ImageOptions{
			// 		ImageType: "PNG",
			// 		ReadDpi:   false,
			// 	}, 0, "")

			// 	// Save the PDF
			// 	err = pdf.Output(f)
			// 	if err != nil {
			// 		fltk.MessageBox("Error", fmt.Sprintf("Failed to write pdf to disk: %v", err.Error()))
			// 		return
			// 	}
			// 	Logf("successfully wrote pdf file to %v", pathToWrite)
			// case ".png":
			// 	f, err := os.Create(pathToWrite)
			// 	if err != nil {
			// 		fltk.MessageBox("Error", fmt.Sprintf("Failed to create png file %v: %v", pathToWrite, err.Error()))
			// 		return
			// 	}
			// 	defer f.Close()

			// 	Logf("preparing png...")
			// 	err = png.Encode(f, img)
			// 	if err != nil {
			// 		fltk.MessageBox("Error", fmt.Sprintf("Failed to write png to disk: %v", err.Error()))
			// 		return
			// 	}
			// 	Logf("successfully scanned image to %v", pathToWrite)
			// }

			// defer runtime.GC()
		}()
	})

	directoryBtn.SetCallback(func() {
		fc := fltk.NewNativeFileChooser()
		defer fc.Destroy()
		fc.SetOptions(fltk.NativeFileChooser_NEW_FOLDER)
		fc.SetType(fltk.NativeFileChooser_BROWSE_DIRECTORY)
		fc.SetTitle("Choose the directory for saving scanned files")
		fc.Show()

		for i, dir := range fc.Filenames() {
			if i > 0 {
				break // only choose the first result from the slice
			}
			appConf.SelectedDir = dir
			Logf("will save scanned files to directory: %v", appConf.SelectedDir)
		}
	})

	deviceOptConstrChoiceCallback := func(option string, constraint string) func() {
		return func() {
			Logf("setting option %v to %v", option, constraint)
			appConf.DeviceSettings[option] = constraint

			// Logf("setting option %v to %v", options[j].Name, options[j].ConstrSet[k])
			// _, err := conn.SetOption(options[j].Name, options[j].ConstrSet[k])
			// if err != nil {
			// 	fltk.MessageBox("Error", fmt.Sprintf("Failed to set %v to %v: %v", options[j].Name, options[j].ConstrSet[k], err.Error()))
			// 	return
			// }

			// update the stored resolution value
			// if strings.ToLower(options[j].Name) == "resolution" {
			// 	r, ok := options[j].ConstrSet[k].(int)
			// 	if !ok {
			// 		Logf("failed to cast resolution dpi %v as int", options[j].ConstrSet[k])
			// 	} else {
			// 		selectedRes = r
			// 		Logf("updated selected resolution to %v", selectedRes)
			// 	}
			// }
		}
	}

	deviceOptChoiceCallback := func(option string, constraints []string) func() {
		return func() {
			// clear out the constChoice options
			// iterate through all the constraints for this option
			// and add each of them to the constChoice items

			if constChoice == nil {
				fltk.MessageBox("Error", "Unable to populate the device constraints choice widget.")
				return
			}

			constChoice.Clear()
			constChoice.Redraw()

			// Logf("constraints for option %v: %v", options[j].Name, options[j].ConstrSet)
			Logf("constraints for option %v: %v", option, constraints)

			// get the current value
			// current, err := conn.GetOption(options[j].Name)
			// if err != nil {
			// 	Logf("failed to get the currently selected option: %v", err.Error())
			// 	return
			// }

			selectedIndex := 0

			// for k := range options[j].ConstrSet {
			for k, constraint := range constraints {
				// k := k
				// Logf("adding option %v constraint %v choice...", options[j].Name, options[j].ConstrSet[k])
				// l := fmt.Sprint(options[j].ConstrSet[k])
				// constChoice.Add(l, func() {
				constChoice.Add(constraint, deviceOptConstrChoiceCallback(option, constraint))

				// set a default value for the resolution (seems like sane usually just picks
				// the first value)
				// if selectedRes == 0 && strings.ToLower(options[j].Name) == "resolution" {
				// 	r, ok := options[j].ConstrSet[k].(int)
				// 	if !ok {
				// 		Logf("failed to cast resolution dpi %v as int", options[j].ConstrSet[k])
				// 	} else {
				// 		selectedRes = r
				// 		Logf("updated selected resolution to %v", selectedRes)
				// 	}
				// }

				// if l == fmt.Sprint(current) && l != "" {
				// 	selectedIndex = k
				// }

				c, ok := appConf.DeviceSettings[option]
				if !ok || c == "" {
					return
				}
				if c == constraint {
					selectedIndex = k
				}
			}

			// if len(options[j].ConstrSet) == 0 {
			if len(constraints) == 0 {
				constChoice.Deactivate()
			} else {
				constChoice.Activate()
				constChoice.SetValue(selectedIndex)
				constChoice.Redraw()
			}
		}
	}

	getDeviceOptsCallback := func(i int, scanner ScannerDevice) func() {
		return func() {
			// if conn != nil {
			// 	if conn.Device == device {
			// 		return
			// 	}

			// 	choice := fltk.ChoiceDialog("You're already connected to another device. Do you want to disconnect?", "Yes", "Cancel")
			// 	if choice == 1 {
			// 		Log("canceling operation")
			// 		return
			// 	}
			// 	Log("connecting to new device")

			// conn.Close()
			// sane.Exit()
			// err := sane.Init()
			// if err != nil {
			// 	fltk.MessageBox("Error", fmt.Sprintf("Failed to initialize scanner framework (sane): %v", err.Error()))
			// 	return
			// }
			// }

			appConf.Device = scanner.Device
			Log(appConf.Device)
			// conn, err = sane.Open(device)
			// if err != nil {
			// 	fltk.MessageBox("Error", fmt.Sprintf("Failed to connect to device %v: %v", device, err.Error()))
			// 	return
			// }

			// if conn == nil {
			// 	fltk.MessageBox("Error", "No device connection is present. Please try again.")
			// 	return
			// }

			// options := conn.Options()

			appConf.DeviceMap, appConf.DeviceSettings, err = getDeviceOptionsConstraints(appConf.Device)
			if err != nil {
				fltk.MessageBox("Error", fmt.Sprintf("Unable to get device options: %v", err.Error()))
				return
			}

			optChoice.Clear()
			constChoice.Clear()
			optChoice.Redraw()
			constChoice.Redraw()
			// for j := range options {
			for option, constraints := range appConf.DeviceMap {
				// j := j
				// Logf("device option: %v, values: %v", options[j].Name, options[j].ConstrSet)
				// optLabel := options[j].Name
				// if len(options[j].ConstrSet) == 0 {
				// optLabel = fmt.Sprintf("%v [no options]", optLabel)
				// }
				optLabel := option
				if len(constraints) == 0 {
					optLabel = fmt.Sprintf("%v [no options]", optLabel)
				}

				optChoice.Add(optLabel, deviceOptChoiceCallback(option, constraints))

				// switch option.Name {
				// case "resolution":
				// 	for _, resolution := range option.ConstrSet {
				// 		r, ok := resolution.(int)
				// 		if !ok {
				// 			Logf("failed to cast %v as string", resolution)
				// 			continue
				// 		}
				// 		optChoice.Add(fmt.Sprintf("%v dpi", r), func() {
				// 			Logf("setting device resolution to %v", r)
				// 			// selectedRes = r
				// 			info, err := conn.SetOption(option.Name, resolution)
				// 			if err != nil {
				// 				fltk.MessageBox("Error", fmt.Sprintf("Failed to set %v to %v: %v", option.Name, resolution, err.Error()))
				// 				return
				// 			}

				// 			Logf("info: %v", info)
				// 		})
				// 	}
				// }
			}

			// res, err := conn.GetOption("resolution")
			// if err != nil {
			// 	fltk.MessageBox("Error", fmt.Sprintf("Failed to get resolution options for device %v: %v", device, err.Error()))
			// 	return
			// }

			// r, ok := res.(int)
			// if ok {
			// 	optChoice.SetValue(r)
			// }

			// Logf("current resolution for this device: %v", res)

			scanBtn.Activate()
		}
	}

	getDevicesCallback := func() {
		getDevicesBtn.SetLabel("Getting devices...")
		getDevicesBtn.SetValue(true)
		getDevicesBtn.Deactivate()

		go func() {
			// if !saneInitialized {
			// 	err = sane.Init()
			// 	if err != nil {
			// 		fltk.MessageBox("Error", fmt.Sprintf("Failed to initialize scanning framework (sane): %v", err.Error()))
			// 		return
			// 	}
			// 	saneInitialized = true
			// }

			Log("please wait, scanning devices...")
			var err error
			appConf.Scanners, err = getDevices()
			if err != nil {
				fltk.MessageBox("Error", fmt.Sprintf("Failed to get scanner devices: %v", err.Error()))
				return
			}

			if devicesChoice != nil {
				devicesChoice.Clear()
			}

			for i, scanner := range appConf.Scanners {
				Logf("scanner device: %v, model: %v", scanner.Device, scanner.Model)
				devicesChoice.AddEx(
					strings.ReplaceAll(fmt.Sprintf("%v (%v) [%v]", scanner.Model, scanner.Device, scanner.Type), "/", "\\/"),
					getShortcut(i),
					getDeviceOptsCallback(i, scanner),
					fltk.MENU_VALUE,
				)
			}

			getDevicesBtn.Activate()
			getDevicesBtn.SetValue(false)
			getDevicesBtn.SetLabel("Get Devices")
		}()
	}

	getDevicesBtn.SetCallback(getDevicesCallback)

	devicesChoice.SetTooltip("Discovered devices will show up here. Press the Get Devices button below first.")
	optChoice.SetTooltip("Device options will appear here, once a device is chosen")
	constChoice.SetTooltip("Choose a configurable parameter from the left dropdown, and the available options will be shown here")
	fileTmplInput.SetTooltip("Set the templated filename. %t=unix epoch seconds")

	if len(appConf.Scanners) != 0 {
		for i, scanner := range appConf.Scanners {
			devicesChoice.AddEx(
				strings.ReplaceAll(fmt.Sprintf("%v (%v) [%v]", scanner.Model, scanner.Device, scanner.Type), "/", "\\/"),
				getShortcut(i),
				getDeviceOptsCallback(i, scanner),
				fltk.MENU_VALUE,
			)

			if scanner.Device == appConf.Device && appConf.Device != "" {
				devicesChoice.SetValue(i)
			}
		}
	} else {
		devicesChoice.Add("Discovered devices will show up here. Click here or press the Get Devices button below first.", getDevicesCallback)
	}

	if appConf.DeviceMap != nil && len(appConf.DeviceMap) != 0 {
		validSettings := appConf.DeviceSettings != nil && len(appConf.DeviceSettings) != 0
		for option, constraints := range appConf.DeviceMap {
			optLabel := option
			if len(constraints) == 0 {
				optLabel = fmt.Sprintf("%v [no options]", optLabel)
			}

			optChoice.Add(optLabel, deviceOptChoiceCallback(option, constraints))
			optChoice.SetValue(0)
			for i, constraint := range constraints {
				constChoice.Add(constraint, deviceOptConstrChoiceCallback(option, constraint))
				if validSettings {
					presetConstraint, ok := appConf.DeviceSettings[option]
					if ok && presetConstraint == constraint {
						constChoice.SetValue(i)
					}
				}
			}
		}
	} else {
		optChoice.Add("Device options will appear here, once a device is chosen", nil)
	}

	if appConf.DeviceSettings == nil || len(appConf.DeviceSettings) == 0 {
		constChoice.Add("Choose a configurable parameter from the left dropdown, and the available options will be shown here", nil)
	}

	if appConf.Log != "" {
		activityText = appConf.Log
	} else {
		activityText = "<i>Information will appear here.</i><br/>"
	}
	activity.SetValue(activityText)

	if appConf.FilenameTemplate != "" {
		fileTmplInput.SetValue(appConf.FilenameTemplate)
	} else {
		fileTmplInput.SetValue("scanned-doc-%t.png")
	}

	if appConf.SelectedDir == "" || appConf.Device == "" {
		scanBtn.Deactivate()
	}
	activity.SetCallback(nil)

	gracefulExit := func() {
		Log("closing app and saving config, please wait a moment...")
		// if conn != nil {
		// 	conn.Close()
		// }
		// sane.Exit()

		if configFilePath != "" {
			// push the activity log to the config
			if activity != nil {
				appConf.Log = activity.Value()
			}

			// write the config
			b, err := yaml.Marshal(appConf)
			if err != nil {
				log.Printf("failed to marshal app config to yaml: %v", err.Error())
			} else {
				dir, _ := filepath.Split(configFilePath)
				err := os.MkdirAll(dir, 0o755)
				if err != nil {
					log.Printf("failed to create app config parent dir %v: %v", dir, err.Error())
				}
				err = os.WriteFile(configFilePath, b, 0o644)
				if err != nil {
					log.Printf("failed to save app config to %v: %v", configFilePath, err.Error())
				}
			}

		}

		Log("done, exiting now.")
		os.Exit(0)
	}

	win.SetCallback(gracefulExit)

	fltk.EnableTooltips()
	fltk.SetTooltipDelay(0.1)

	if portrait {
		win.Resize(0, 0, WIDTH_PORTRAIT*3, HEIGHT_PORTRAIT*3)
	} else {
		win.Resize(0, 0, WIDTH_LANDSCAPE*3, HEIGHT_LANDSCAPE*3)
	}

	win.SetResizeHandler(func() {
		responsive(win)
	})

	responsive(win)

	win.End()
	win.Show()

	go fltk.Run()

	// Create a channel to receive OS signals
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// Block until a signal is received
	<-signalChan

	gracefulExit()
}
