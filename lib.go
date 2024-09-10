package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pwiecz/go-fltk"
)

// this is a value from the fltk lib that is used for scrolling to the bottom of
// the helpview
const MAX_TOP_LINE = 1000000

// Wraps around log.Println() as well as adding activity to the
// activity text buffer. Always adds a newline to the activity buffer.
func Log(v ...any) {
	activityText = fmt.Sprintf("%v<p>%v</p>", activityText, fmt.Sprintf("%v", v...))
	activity.SetValue(activityText)
	log.Println(v...)
	activity.SetTopLine(MAX_TOP_LINE)
	activity.SetTopLine(activity.TopLine() - activity.H()) // scroll to the bottom
}

// Wraps around log.Printf() as well as adding activity to the
// activity text buffer. Always adds a newline to the activity buffer.
func Logf(format string, v ...any) {
	format = fmt.Sprintf("%v\n", format)
	activityText = fmt.Sprintf("%v<p>%v</p>", activityText, fmt.Sprintf(format, v...))
	activity.SetValue(activityText)
	log.Printf(format, v...)
	activity.SetTopLine(MAX_TOP_LINE)
	activity.SetTopLine(activity.TopLine() - activity.H()) // scroll to the bottom
}

// isPortrait returns true if the screen is taller than it is wide. It returns
// false otherwise, including for square screens.
func isPortrait() (bool, error) {
	_, _, width, height := fltk.ScreenWorkArea(int(fltk.SCREEN))

	if width == 0 || height == 0 {
		return false, fmt.Errorf("received 0 for one of screen height or width")
	}

	if width > height {
		return false, nil
	}

	return true, nil
}

// Each ScannerDevice corresponds to a discovered device via the scanimage
// command.
type ScannerDevice struct {
	Device string
	Vendor string
	Model  string
	Type   string
	Index  string
}

// Parses the output of scanimage v1.3.1 to retrieve options. Example output:
//
//	mode
//	resolution
//	source
//	brightness
//	contrast
// func getDeviceOptions(dev string) ([]string, error) {
// 	args := []string{
// 		"-c",
// 		fmt.Sprintf(`scanimage --device='%v' -A | grep -v '\[inactive\]' | grep -v '\[advanced\]' | grep '\-\-' | awk '{ print $1 }' | tr -d '\-\-'`, dev),
// 	}

// 	cmd := exec.Command("/bin/bash", args...)
// 	o, err := cmd.Output()
// 	if err != nil {
// 		return []string{}, fmt.Errorf("failed to get device options via scanimage cli: %w", err)
// 	}

// 	results := []string{}

// 	out := strings.Split(string(o), "\n")

// 	for _, opt := range out {
// 		opt = strings.TrimSpace(opt)
// 		if strings.HasSuffix(opt, "]") {
// 			opt = strings.Split(opt, "[")[0]
// 		}
// 		results = append(results, opt)
// 	}

// 	return results, err
// }

// Parses the output of scanimage v1.3.1 to retrieve options. Example CLI
// output:
//
//	`
//	mode 24bit Color[Fast]|Black & White|True Gray|Gray[Error Diffusion] [24bit Color[Fast]]
//	resolution 100|150|200|300|400|600|1200dpi [100]
//	source Automatic Document Feeder(left aligned) [Automatic Document Feeder(left aligned)]
//	`
func parseDeviceOptionConstraints(out []string) (map[string][]string, map[string]string) {
	r, err := regexp.Compile(`^(.*) \[(.*)\]$`)
	if err != nil {
		log.Fatalf("failed to compile regexp: %v", err.Error())
	}

	results := make(map[string][]string)
	defaults := make(map[string]string)
	for _, opt := range out {
		opt = strings.TrimSpace(opt)
		expanded := strings.Split(opt, " ")
		parsedOpt := expanded[0]

		if len(expanded) <= 1 {
			continue
		}

		withoutOpt := strings.Split(opt, parsedOpt)[1]

		constraints := strings.Split(withoutOpt, "|")
		if len(constraints) == 0 {
			continue
		}

		results[parsedOpt] = []string{}

		// try to identify the supported constraints for each option, including
		// the currently selected one
		for i, constraint := range constraints {
			// the last constraint will always contain the currently set value of
			// the option, such as "1200dpi [100]"
			if i == len(constraints)-1 {
				matches := r.FindStringSubmatch(constraint)
				log.Printf("%v matches: %v", len(matches), matches)
				if len(matches) != 3 {
					continue
				}

				constraint = matches[1]
				defaults[parsedOpt] = matches[2]
			}

			results[parsedOpt] = append(results[parsedOpt], strings.TrimSpace(constraint))
		}
	}

	return results, defaults
}

func getDeviceOptionsConstraints(dev string) (map[string][]string, map[string]string, error) {
	args := []string{
		"-c",
		fmt.Sprintf(`scanimage --device='%v' -A | grep -v '\[inactive\]' | grep -v '\[advanced\]' | grep '\-\-' | tr -d '\-\-'`, dev),
	}

	cmd := exec.Command("/bin/bash", args...)
	o, err := cmd.Output()
	if err != nil {
		return map[string][]string{}, map[string]string{}, fmt.Errorf("failed to get device option constraints via scanimage cli: %w", err)
	}

	out := strings.Split(string(o), "\n")
	results, defaults := parseDeviceOptionConstraints(out)

	return results, defaults, err
}

// getDevices retrieves a list of scanner devices as a string slice.
func getDevices() ([]ScannerDevice, error) {
	// in the event that the sane library doesn't work, here's how to do it
	// via command line:

	var ob bytes.Buffer
	var eb bytes.Buffer

	command := "scanimage"
	args := []string{`--formatted-device-list=%d||%v||%m||%t||%i;;;`, "--list-devices"}

	log.Printf("running command %v with args %v", command, args)

	_, err := RunCommand(command, args, []string{}, nil, &ob, &eb)
	if err != nil {
		log.Printf("stdout for convert: %v", ob.String())
		log.Printf("stderr for convert: %v", eb.String())
		return []ScannerDevice{}, err
	}

	scanners := []ScannerDevice{}

	outLines := strings.Split(ob.String(), ";;;")
	for _, outLine := range outLines {
		// split based on the || pattern from above
		values := strings.Split(outLine, "||")

		newScanner := ScannerDevice{}
		for i := range values {
			switch i {
			case 0:
				newScanner.Device = values[i]
			case 1:
				newScanner.Vendor = values[i]
			case 2:
				newScanner.Model = values[i]
			case 3:
				newScanner.Type = values[i]
			case 4:
				newScanner.Index = values[i]
			}
		}

		if newScanner.Device == "" {
			continue
		}

		scanners = append(scanners, newScanner)
	}

	return scanners, nil

	// devs, err := sane.Devices()
	// if err != nil {
	// 	return []ScannerDevice{}, fmt.Errorf("failed to get devices: %v", err.Error())
	// }

	// scanners := []ScannerDevice{}
	// for i, dev := range devs {
	// 	scanners = append(scanners, ScannerDevice{
	// 		Device: dev.Name,
	// 		Vendor: dev.Vendor,
	// 		Type:   dev.Type,
	// 		Model:  dev.Model,
	// 		Index:  fmt.Sprint(i),
	// 	})
	// }

	// return scanners, nil
}

// ScanImage runs a /bin/bash command to scan an image.
func ScanImage(filename string, deviceSettings map[string]string, format string, dev string) (string, error) {
	// scanimage --device='brother5:bus2;dev1' --resolution 300 --progress --format=pdf > scanned_doc_$(date +%s).pdf

	sb := new(strings.Builder)
	for k, v := range deviceSettings {
		if v == "" {
			continue
		}

		sb.WriteString(fmt.Sprintf("--%v='%v' ", k, v))
	}

	args := []string{
		"-c",
		fmt.Sprintf(`scanimage --device='%v' %v--format=%v > %v`, dev, sb.String(), format, filename),
	}
	cmd := exec.Command("/bin/bash", args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// Runs a command with the provided command (such as `/bin/sh`) and args (such
// as ["-c","'echo hello'"]) and environment variables (such as 'DISPLAY=:0').
//
// `stdin`, `stdout`, and `stderr` can all be `nil`.
//
// Returns the exit code of the command when it finishes.
func RunCommand(command string, args []string, env []string, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	cmd := exec.Command(command, args...)
	cmd.Env = append(cmd.Env, env...)

	if stdin != nil {
		cmd.Stdin = stdin
	}

	if stdout != nil {
		cmd.Stdout = stdout
	}

	if stderr != nil {
		cmd.Stderr = stdout
	}

	err := cmd.Run()
	if err != nil {
		return cmd.ProcessState.ExitCode(), err
	}

	return cmd.ProcessState.ExitCode(), nil
}

// getShortcut returns the keyboard shortcut for the corresponding index.
func getShortcut(i int) int {
	switch i {
	case 0:
		return '1'
	case 1:
		return '2'
	case 2:
		return '3'
	case 3:
		return '4'
	case 4:
		return '5'
	case 5:
		return '6'
	case 6:
		return '7'
	case 7:
		return '8'
	case 8:
		return '9'
	default:
		return 0
	}
}

// getFileType returns "png", "pdf", "jpeg", or other similar formats that are
// in scope for this application. If the filetype isn't supported, it returns an
// empty string.
func getFileType(s string) string {
	ext := filepath.Ext(s)
	switch ext {
	case ".png":
		return ext
	case ".jpg":
		return ext
	case ".jpeg":
		return ext
	case ".pdf":
		return ext
	}

	return ""
}
