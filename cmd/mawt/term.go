package main

import (
	"fmt"
	"os"

	"github.com/karlmutch/errors"
)

var (
	errorC   chan<- errors.Error
	messageC <-chan string
	errorsC  <-chan errors.Error

	msgV = os.Stdout
	errV = os.Stderr
)

func runTUI(msgC chan string, errC chan errors.Error, quitC <-chan struct{}) {

	errorC = errC
	messageC = msgC
	errorsC = errC
	/**
		g, errGo := gocui.NewGui(gocui.Output256)
		if errGo != nil {
			errorC <- errors.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
		}
		defer g.Close()

		if errGo = g.SetManagerFunc(layout); errGo != nil {
			errorC <- errors.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
		}
	**/
	go msgWatch(messageC, errorsC, quitC)
	/**
	if errGo = g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); errGo != nil {
		errorC <- errors.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
	}

	if errGo = g.MainLoop(); errGo != nil && errGo != gocui.ErrQuit {
		errorC <- errors.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
	}
	**/
}

func msgWatch(msgsC <-chan string, errorC <-chan errors.Error, quitC <-chan struct{}) {
	for {
		select {
		case msg := <-msgsC:
			if msgV != nil {
				fmt.Fprint(msgV, msg)
			}
		case err := <-errorC:
			if msgV != nil {
				fmt.Fprint(msgV, err.Error())
			}
		case <-quitC:
			return
		}
	}
}

/**
func layout(g *gocui.Gui) (err error) {
	maxX, maxY := g.Size()
	if msgV, errGo := g.SetView("colors", -1, -1, maxX, maxY); errGo != nil {
		if errGo != gocui.ErrUnknownView {
			errorC <- errors.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
			return err
		}

		// 256-colors escape codes
		for i := 0; i < 256; i++ {
			str := fmt.Sprintf("\x1b[48;5;%dm\x1b[30m%3d\x1b[0m ", i, i)
			str += fmt.Sprintf("\x1b[38;5;%dm%3d\x1b[0m ", i, i)

			if (i+1)%10 == 0 {
				str += "\n"
			}

			fmt.Fprint(msgV, str)
		}

		fmt.Fprint(msgV, "\n\n")

		// 8-colors escape codes
		ctr := 0
		for i := 0; i <= 7; i++ {
			for _, j := range []int{1, 4, 7} {
				str := fmt.Sprintf("\x1b[3%d;%dm%d:%d\x1b[0m ", i, j, i, j)
				if (ctr+1)%20 == 0 {
					str += "\n"
				}

				fmt.Fprint(msgV, str)

				ctr++
			}
		}
	}
	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}
**/
