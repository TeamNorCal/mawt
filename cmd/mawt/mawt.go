package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/mgutz/logxi" // Using a forked copy of this package results in build issues

	"github.com/TeamNorCal/mawt/version"

	"github.com/go-stack/stack"
	"github.com/karlmutch/errors"

	"github.com/karlmutch/envflag" // Forked copy of https://github.com/GoBike/envflag
)

var (
	logger = logxi.New("mawt")

	verbose = flag.Bool("v", false, "When enabled will print internal logging for this tool")
)

func usage() {
	fmt.Fprintln(os.Stderr, path.Base(os.Args[0]))
	fmt.Fprintln(os.Stderr, "usage: ", os.Args[0], "[options]       techthulu ← TCP → OPC (mawt)      ", version.GitHash, "    ", version.BuildTime)
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "mawt is a gateway between Niantic Ingress Techthulu and OPC based USB fadecandy boards")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Options:")
	fmt.Fprintln(os.Stderr, "")
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Environment Variables:")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "options can also be extracted from environment variables by changing dashes '-' to underscores and using upper case.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "log levels are handled by the LOGXI env variables, these are documented at https://github.com/mgutz/logxi")
}

func init() {
	flag.Usage = usage
}

// Go runtime entry point.
//
func main() {

	quitC := make(chan struct{})
	defer close(quitC)

	// Skip this step when the server is not running in production mode, that is when the
	// server is being used in an automatted test
	//
	if err := exclusive("mawt", quitC); err != nil {
		logger.Error(fmt.Sprintf("An instance of this process is already running %s", err.Error()))
		os.Exit(-1)
	}

	Main()
}

func Main() {

	if !flag.Parsed() {
		envflag.Parse()
	}

	if *verbose {
		logger.SetLevel(logxi.LevelDebug)
	}

	logger.Debug(fmt.Sprintf("%s built at %s, against commit id %s\n", os.Args[0], version.BuildTime, version.GitHash))

	doneC := make(chan struct{})
	quitC := make(chan struct{})

	if errs := EntryPoint(quitC, doneC); len(errs) != 0 {
		for _, err := range errs {
			logger.Error(err.Error())
		}
		os.Exit(-1)
	}

	// After starting the application message handling loops
	// wait until the system has shutdown
	//
	select {
	case <-quitC:
	}

	// Allow the quitC to be sent before exiting, giving other modules a chance to stop
	time.Sleep(time.Second)

}

func initOPC(quitC <-chan struct{}) (err errors.Error) {

	go func(quitC <-chan struct{}) {
	}(quitC)

	return nil
}

func initSound(quitC <-chan struct{}) (err errors.Error) {

	go func(quitC <-chan struct{}) {
	}(quitC)

	return nil
}

func initTechthulu(quitC <-chan struct{}) (err errors.Error) {

	go func(quitC <-chan struct{}) {
	}(quitC)

	return nil
}

func EntryPoint(quitC chan struct{}, doneC chan struct{}) (errs []errors.Error) {

	errs = []errors.Error{}

	defer close(doneC)

	// Supplying the context allows the client to pubsub to cancel the
	// blocking receive inside the run
	ctx, cancel := context.WithCancel(context.Background())

	// Setup a channel to allow a CTRL-C to terminate all processing.  When the CTRL-C
	// occurs we cancel the background msg pump processing pubsub mesages from
	// google, and this will also cause the main thread to unblock and return
	//
	stopC := make(chan os.Signal)
	go func() {
		defer cancel()

		select {
		case <-quitC:
			return
		case <-stopC:
			logger.Warn("CTRL-C Seen")
			close(quitC)
			return
		}
	}()

	signal.Notify(stopC, os.Interrupt, syscall.SIGTERM)

	// Now start initializing the servers processing components

	if err := initSound(ctx.Done()); err != nil {
		errs = append(errs, err)
	}

	if err := initOPC(ctx.Done()); err != nil {
		errs = append(errs, err)
	}

	if err := initTechthulu(ctx.Done()); err != nil {
		errs = append(errs, err)
	}

	return errs
}

func exclusive(name string, quitC chan struct{}) (err errors.Error) {

	excl := struct {
		name     string
		releaseC chan struct{}
		listen   net.Listener
	}{
		name:     name,
		releaseC: quitC,
		listen:   nil}

	// Construct an abstract name socket that allows the name to be recycled between process
	// restarts without needing to unlink etc. For more information please see
	// https://gavv.github.io/blog/unix-socket-reuse/, and
	// http://man7.org/linux/man-pages/man7/unix.7.html
	sockName := "@/tmp/"
	sockName += name

	errGo := fmt.Errorf("")
	excl.listen, errGo = net.Listen("unix", sockName)
	if errGo != nil {
		return errors.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
	}
	go func() {
		go excl.listen.Accept()
		<-excl.releaseC
	}()
	return nil
}
