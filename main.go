package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/gobs/args"
	"github.com/raff/godet"
)

func runCommand(commandString string) error {
	parts := args.GetArgs(commandString)
	cmd := exec.Command(parts[0], parts[1:]...)
	return cmd.Start()
}

func limit(s string, l int) string {
	if len(s) > l {
		return s[:l] + "..."
	}
	return s
}

func main() {
	var chromeapp string

	switch runtime.GOOS {
	case "darwin":
		for _, c := range []string{
			"/Applications/Google Chrome Canary.app",
			"/Applications/Google Chrome.app",
		} {
			// MacOS apps are actually folders
			if info, err := os.Stat(c); err == nil && info.IsDir() {
				chromeapp = fmt.Sprintf("open %q --args", c)
				break
			}
		}

	case "linux":
		for _, c := range []string{
			"headless_shell",
			"chromium",
			"google-chrome-beta",
			"google-chrome-unstable",
			"google-chrome-stable"} {
			if _, err := exec.LookPath(c); err == nil {
				chromeapp = c
				break
			}
		}

	case "windows":
	}

	if chromeapp != "" {
		chromeapp += " --remote-debugging-port=9222 --disable-extensions --disable-gpu --window-size=1440,900 --headless about:blank"
	}

	cmd := flag.String("cmd", chromeapp, "command to execute to start the browser")
	port := flag.String("port", "localhost:9222", "Chrome remote debugger port")
	verbose := flag.Bool("v", false, "verbose logging")
	flag.Parse()

	if *cmd != "" {
		log.Println(*cmd)
		if err := runCommand(*cmd); err != nil {
			log.Println("cannot start browser", err)
		}
	}

	var remote *godet.RemoteDebugger
	var err error

	for i := 0; i < 10; i++ {
		if i > 0 {
			time.Sleep(500 * time.Millisecond)
		}

		remote, err = godet.Connect(*port, *verbose)
		if err == nil {
			break
		}

		log.Println("connect", err)
	}

	if err != nil {
		log.Fatal("cannot connect to browser")
	}

	defer remote.Close()

	done := make(chan bool)

	v, err := remote.Version()
	if err != nil {
		log.Fatal("cannot get version: ", err)
	}

	log.Println("connected to", v.Browser, "protocol version", v.ProtocolVersion)

	remote.CallbackEvent(godet.EventClosed, func(params godet.Params) {
		log.Println("RemoteDebugger connection terminated.")
		done <- true
	})

	remote.CallbackEvent("DOM.documentUpdated", func(params godet.Params) {
		log.Println("document updated. taking screenshot...")
		t := time.Now()
		remote.SaveScreenshot(fmt.Sprintf("screenshot-%s.png", t.Format("20060102150405")), 0644, 0, true)
		done <- true
	})

	// install some callbacks
	remote.CallbackEvent(godet.EventClosed, func(params godet.Params) {
		fmt.Println("RemoteDebugger connection terminated.")
	})

	remote.CallbackEvent("Network.requestWillBeSent", func(params godet.Params) {
		fmt.Println("requestWillBeSent",
			params["type"],
			params["documentURL"],
			params["request"].(map[string]interface{})["url"])
	})

	remote.CallbackEvent("Network.responseReceived", func(params godet.Params) {
		fmt.Println("responseReceived",
			params["type"],
			params["response"].(map[string]interface{})["url"])
	})

	remote.CallbackEvent("Log.entryAdded", func(params godet.Params) {
		entry := params["entry"].(map[string]interface{})
		fmt.Println("LOG", entry["type"], entry["level"], entry["text"])
	})

	// navigate in existing tab
	tabs, err := remote.TabList("")
	if err != nil {
		log.Println("error getting tabs: ", err)
	}
	err = remote.ActivateTab(tabs[0])
	if err != nil {
		log.Println("error switching tab: ", err)
	}

	// re-enable events when changing active tab
	remote.AllEvents(true) // enable all events

	_, err = remote.Navigate("https://natwelch.com")
	if err != nil {
		log.Println("error navigating: ", err)
	}

	<-done

	log.Println("Closing")
}
