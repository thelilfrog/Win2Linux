package main

import (
	_ "embed"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/getlantern/systray"
)

//go:generate go-winres make

type (
	Entry struct {
		Key        string
		Attributes map[string]string
	}
)

var (
	r *regexp.Regexp = regexp.MustCompile(`-+`)
)

//go:embed tray.ico
var icon []byte

type ()

func main() {
	systray.Run(onReady, onExit)
}

func onReady() {
	entries := list()
	systray.SetTitle("Win2Linux")
	systray.SetTooltip("Switch from Windows to Linux")
	systray.SetIcon(icon)

	for _, entry := range entries {
		key := entry.Key
		if desc, ok := entry.Attributes["description"]; ok {
			key = desc
		}
		mCustom := systray.AddMenuItem(key, "Switch to"+key)
		go func() {
			<-mCustom.ClickedCh
			reboot(entry.Attributes["identifier"])
		}()

	}
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit")

	go func() {
		<-mQuit.ClickedCh
		os.Exit(0)
	}()
}

func onExit() {
	// clean up here
}

func list() []Entry {
	cmd := exec.Command("bcdedit", "/enum", "firmware")
	out, err := cmd.Output()
	if err != nil {
		panic(err)
	}
	return parse(string(out))
}

func parse(out string) []Entry {
	lines := strings.Split(out, "\r\n")
	section := false
	lastLine := ""
	secName := ""
	a := make(map[string][]string)
	for _, l := range lines {
		if !section {
			if r.Match([]byte(l)) {
				secName = lastLine
				section = true
			}
		} else {
			if len(l) != 0 {
				a[secName] = append(a[secName], l)
			} else {
				section = false
			}
		}

		lastLine = l
	}

	var entries []Entry
	for k, sec := range a {
		entry := Entry{
			Key:        k,
			Attributes: make(map[string]string),
		}
		for _, l := range sec {
			l = strings.Join(strings.Fields(l), " ")
			val := strings.SplitN(l, " ", 1)
			if len(val) != 2 {
				continue
			}
			entry.Attributes[val[0]] = val[1]
		}
		entries = append(entries, entry)
	}
	return entries
}

func reboot(uuid string) {
	if err := exec.Command("bcdedit", "/Set", "{fwbootmgr}", "BootSequence", uuid, "/addFirst").Run(); err != nil {
		panic(err)
	}

	if err := exec.Command("shutdown", "/r", "/t", "2").Run(); err != nil {
		panic(err)
	}

	os.Exit(0)
}
