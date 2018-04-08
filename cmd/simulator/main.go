package main

import (
	"bytes"
	"crypto/md5"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mgutz/logxi"
)

var (
	listen       = flag.String("listen", ":8080", "Address to bind to")
	scenarioPath = flag.String("path", "./", "Path served as document root.")
	remote       = flag.Bool("remote", false, "Enable remote management of the scenario being run")
	scale        = flag.Int("scale", 1, "factor by which to accelerate the relative rate of the clock")
	compressTime = flag.Duration("compress-time", time.Duration(25*time.Hour), "Compress time scale to remove specified periods of inactivity")
)

type testSlot struct {
	secondSlot int    // The second at which the served directory activates
	dir        string // The directory that activates
}

type testWindow struct {
	startTime time.Time
	slots     []*testSlot
	sync.Mutex
}

var (
	// create Logger interface
	logW = logxi.NewLogger(logxi.NewConcurrentWriter(os.Stdout), "mawt-simulator")

	testSchedule = testWindow{
		startTime: time.Now().Round(time.Second),
		slots:     []*testSlot{},
	}

	// This channel forces an immediate reload of the scenario
	forcedLoad = make(chan bool, 1)
)

func main() {

	flag.Parse()

	_, err := filepath.Abs(*scenarioPath)
	if err != nil {
		logxi.Fatal(err.Error())
		os.Exit(-1)
	}

	// Load the inital per second slots from the
	// scenario directory
	loadTest(*scenarioPath)

	// Start a service function that tracks over time the slots
	// and scenarios being used
	//
	go auditWindow()

	http.HandleFunc("/", serveHandler)

	if err = http.ListenAndServe(*listen, nil); err != nil {
		logW.Warn(err.Error())
	}
}

// hashFile will get the MD5 sum of a file and return it
//
func hashFile(fn string) []byte {
	f, err := os.Open(fn)
	if err != nil {
		return []byte{}
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return []byte{}
	}

	return h.Sum(nil)
}

// loadTest examines the scenario directory for the serve directories
// that will be used and loads them into the testSchedule
//
func loadTest(scenario string) (err error) {
	testSchedule.Lock()
	defer testSchedule.Unlock()

	testSchedule.startTime = time.Now().Round(time.Second)
	testSchedule.slots = []*testSlot{}

	oldHash := []byte{}
	newHash := []byte{}

	err = filepath.Walk(scenario,
		func(path string, f os.FileInfo, err error) error {
			if path == scenario {
				return nil
			}

			// Drop empty test files they are not of any use
			if f.Size() == 0 {
				return nil
			}
			slot, err := strconv.Atoi(f.Name())

			// The compress-time option is used to collapse uninteresting time periods in the life
			// of the data that is being transmitted.  The collapse will look for dead air, or for files
			// the are duplicated and remove them.  In this case we will remove duplicates and later on
			// will deal with dead air
			if *compressTime < time.Duration(24*time.Hour) {
				newHash = hashFile(filepath.Join(path, f.Name()))
				if len(newHash) != 0 && 0 == bytes.Compare(newHash, oldHash) {
					logxi.Info("duplicate file", filepath.Join(path, f.Name()))
					return nil
				}
			}
			oldHash = newHash

			if err == nil {
				testSchedule.slots = append(testSchedule.slots, &testSlot{
					dir:        path,
					secondSlot: slot})
			}
			if f.IsDir() {
				return filepath.SkipDir
			}
			return nil
		})
	if err != nil {
		logW.Warn(fmt.Sprintf("could not load test scenario from %s due to %s", scenario, err.Error()), "error", err)
	}

	// Sort our slots  ascending order and we are done
	sort.Slice(testSchedule.slots, func(i, j int) bool {
		return testSchedule.slots[i].secondSlot < testSchedule.slots[j].secondSlot
	})

	// Having got the files in order deal with the case of  having large air gaps
	// in the rolling data
	//
	if *compressTime < time.Duration(24*time.Hour) {

		lastAbsSlot := 0
		lastAdjSlot := 0
		accumulated := 0

		adjust := int(compressTime.Seconds())
		for i, _ := range testSchedule.slots {

			// Adjust the slot time down by the total accumulated seconds that
			// have been removed
			newAbsSlot := testSchedule.slots[i].secondSlot
			testSchedule.slots[i].secondSlot -= accumulated
			newAdjSlot := testSchedule.slots[i].secondSlot

			if testSchedule.slots[i].secondSlot > lastAdjSlot+adjust {
				dropSecs := testSchedule.slots[i].secondSlot - lastAdjSlot - adjust
				logW.Debug(fmt.Sprintf("removing %d seconds between %d (%d) and %d (%d)",
					dropSecs, lastAbsSlot, lastAdjSlot, newAbsSlot, newAdjSlot))

				accumulated += dropSecs
				testSchedule.slots[i].secondSlot = lastAdjSlot + adjust
			}
			lastAdjSlot = newAdjSlot
			lastAbsSlot = newAbsSlot
		}
	}

	logW.Debug(fmt.Sprintf("loaded scenario %s", scenario))

	return err
}

func getSlotDir() (dir string) {
	testSchedule.Lock()
	defer testSchedule.Unlock()

	second := int(time.Since(testSchedule.startTime).Seconds()*float64(*scale)) - 1
	slot := sort.Search(len(testSchedule.slots), func(i int) bool { return testSchedule.slots[i].secondSlot >= second })

	if slot < len(testSchedule.slots) && testSchedule.slots[slot].secondSlot == second {
		return testSchedule.slots[slot].dir
	}

	if slot >= len(testSchedule.slots) {
		slot = len(testSchedule.slots) - 1
	} else {
		if testSchedule.slots[slot].secondSlot > second {
			slot = slot - 1
		}
		if slot < 0 {
			slot = 0
		}
	}

	return testSchedule.slots[slot].dir
}

func auditWindow() {
	tick := time.NewTicker(500 * time.Millisecond)
	defer tick.Stop()

	for {
		select {
		case <-forcedLoad:
			logW.Debug(fmt.Sprintf("forced load of %s occurring", *scenarioPath))
			loadTest(*scenarioPath)

		case <-tick.C:
			logW.Debug(fmt.Sprintf("using %s", getSlotDir()))

			files, _ := ioutil.ReadDir(getSlotDir())
			for _, aFile := range files {
				if aFile.Name() == "finish" {
					loadTest(*scenarioPath)
					break
				}
			}
		}
	}
}

func serveConfigure(w http.ResponseWriter, r *http.Request) {

	if !path.IsAbs(r.URL.Path) {
		http.Error(w, "configure paths must be absolute", 404)
		return
	}

	*scenarioPath = strings.TrimPrefix(r.URL.Path, "/configure")

	select {
	case forcedLoad <- true:
	case <-time.After(3 * time.Second):
		http.Error(w, "configure path although saved, could not be applied immediately", 500)
	}
}

func serveHandler(w http.ResponseWriter, r *http.Request) {

	if *remote && strings.HasPrefix(r.URL.Path, "/configure/") {
		serveConfigure(w, r)
		return
	}

	// Locate from the current test scenario which
	// directory is the appropriate one to serve up
	//
	file := getSlotDir() + r.URL.Path
	logW.Debug(fmt.Sprintf("serving %s", file))

	http.ServeFile(w, r, file)
}
