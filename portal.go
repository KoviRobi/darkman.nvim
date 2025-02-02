package main

import (
	"github.com/godbus/dbus/v5"
)

const PORTAL_BUS_NAME = "nl.whynothugo.darkman"
const PORTAL_OBJ_PATH = "/nl/whynothugo/darkman"
const PORTAL_INTERFACE = "org.freedesktop.DBus.Properties"
const PORTAL_NAMESPACE = "nl.whynothugo.darkman"
const PORTAL_KEY = "Mode"

type Portal struct {
	*dbus.Conn
}

func setupPortal() (Portal, error) {
	conn, err := dbus.ConnectSessionBus()
	return Portal{conn}, err
}

func (p *Portal) getMode() (string, error) {
	dest := p.Object(PORTAL_BUS_NAME, PORTAL_OBJ_PATH)
	var mode string
	err := dest.Call(PORTAL_INTERFACE+".Get", 0, PORTAL_NAMESPACE, PORTAL_KEY).Store(&mode)
	if err != nil {
		return "", err
	}
	return mode, nil
}

func (p *Portal) setupSignal() (<-chan string, error) {
	signals := make(chan *dbus.Signal)
	modeChan := make(chan string)
	p.Signal(signals)
	err := p.AddMatchSignal(
		dbus.WithMatchSender(PORTAL_BUS_NAME),
		dbus.WithMatchObjectPath(PORTAL_OBJ_PATH),
		dbus.WithMatchInterface(PORTAL_INTERFACE),
		dbus.WithMatchMember("PropertiesChanged"),
		dbus.WithMatchArg0Namespace(PORTAL_NAMESPACE),
	)
	if err != nil {
		return nil, err
	}
	go func() {
		for {
			sig := <-signals
			if len(sig.Body) != 3 {
				continue
			}
			if len(sig.Body) >= 2 {
				if dict, ok := sig.Body[1].(map[string]dbus.Variant); ok {
					if val, ok := dict[PORTAL_KEY].Value().(string); ok {
						modeChan <- val
					}
				}
			}
		}
	}()

	return modeChan, nil
}
