package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/neovim/go-client/nvim"
	"github.com/neovim/go-client/nvim/plugin"
)

const (
	UNINITIALIZED string = ""
	DARK                 = "dark"
	LIGHT                = "light"
)

var currentMode string

type setupArgs struct {
	v                *nvim.Nvim `msgpack:"-"`
	ChangeBackground bool       `msgpack:",array"`
	SendUserEvent    bool
	Colorscheme      *struct {
		Dark  string `msgpack:"dark"`
		Light string `msgpack:"light"`
	}
}

func getMode(args []string) (string, error) {
	if currentMode == UNINITIALIZED {
		return "", errors.New("Mode not yet initialized, call `Setup`")
	}
	return currentMode, nil
}

func (args *setupArgs) handleNewMode() error {
	var err error
	var background, colorscheme, event string
	switch currentMode {
	case DARK:
		background, event = "dark", "DarkMode"
		if args.Colorscheme != nil {
			colorscheme = args.Colorscheme.Dark
		}
	case LIGHT:
		background, event = "light", "LightMode"
		if args.Colorscheme != nil {
			colorscheme = args.Colorscheme.Light
		}
	default:
		return fmt.Errorf("Unexpected mode: %s", currentMode)
	}
	if c := args.Colorscheme; c != nil {
		err = args.v.Command("colorscheme " + colorscheme)
		if err != nil {
			return err
		}
	}
	if args.ChangeBackground {
		err = args.v.SetOption("background", background)
		if err != nil {
			return err
		}
	}
	if args.SendUserEvent {
		err = args.v.Command("doautocmd User " + event)
		if err != nil {
			return err
		}
	}
	return err
}

func setup(v *nvim.Nvim, args setupArgs) {
	var err error
	var p Portal
	var ch <-chan string
	if currentMode != UNINITIALIZED {
		err = errors.New("setup() already called")
		goto error
	}
	args.v = v
	if p, err = setupPortal(); err != nil {
		goto error
	}
	if currentMode, err = p.getMode(); err != nil {
		goto error
	}
	if err = args.handleNewMode(); err != nil {
		goto error
	}

	if ch, err = p.setupSignal(); err != nil {
		goto error
	}
	go func() {
		for {
			if newMode := <-ch; newMode != currentMode {
				currentMode = newMode
				args.handleNewMode()
			}
		}
	}()
	return

error:
	v.WriteErr(fmt.Sprintf("darkman: %v\n", err))
	return
}

func main() {
	if len(os.Args) == 2 && os.Args[1] == "debug" {
		var err error
		var p Portal
		var ch <-chan string
		if p, err = setupPortal(); err != nil {
			log.Fatal(err)
		}
		if currentMode, err = p.getMode(); err != nil {
			log.Fatal(err)
		}
		log.Println("Current mode is", currentMode)
		if ch, err = p.setupSignal(); err != nil {
			log.Fatal(err)
		}
		for currentMode = range ch {
			log.Println("New mode", currentMode)
		}
	} else {
		plugin.Main(func(p *plugin.Plugin) error {
			p.HandleFunction(&plugin.FunctionOptions{Name: "DarkmanGetMode"}, getMode)
			p.HandleFunction(&plugin.FunctionOptions{Name: "DarkmanSetup"}, setup)
			return nil
		})
	}
}
