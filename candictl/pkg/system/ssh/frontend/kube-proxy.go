package frontend

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/ssh/terminal"

	"flant/candictl/pkg/app"
	"flant/candictl/pkg/log"
	"flant/candictl/pkg/system/ssh/session"
)

var LocalApiPort = 22322

type KubeProxy struct {
	Session *session.Session

	KubeProxyPort string
	LocalPort     string

	proxy  *Command
	tunnel *Tunnel

	stop bool
}

func NewKubeProxy(sess *session.Session) *KubeProxy {
	return &KubeProxy{Session: sess}
}

func (k *KubeProxy) Start() (port string, err error) {
	success := false
	defer func() {
		if !success {
			k.Stop()
		}
	}()

	k.proxy = NewCommand(k.Session, `kubectl proxy --port=0`).Sudo().WithSSHArgs("-o", "ServerAliveCountMax=3")

	port = ""
	portReady := make(chan struct{}, 1)
	portRe := regexp.MustCompile(`Starting to serve on .*?:(\d+)`)

	k.proxy.WithStdoutHandler(func(line string) {
		m := portRe.FindStringSubmatch(line)
		if len(m) == 2 && m[1] != "" {
			port = m[1]
			log.InfoF("Got proxy port = %s\n", port)
			portReady <- struct{}{}
		}
	})
	onStart := make(chan struct{}, 1)
	k.proxy.OnCommandStart(func() {
		onStart <- struct{}{}
	})
	waitCh := make(chan error, 1)
	k.proxy.WithWaitHandler(func(err error) {
		waitCh <- err
	})

	app.Debugf("Start proxy process\n")
	err = k.proxy.Start()
	if err != nil {
		return "", fmt.Errorf("start kubectl proxy: %v", err)
	}

	<-onStart

	// Wait for proxy startup
	t := time.NewTicker(20 * time.Second)
	defer t.Stop()
	select {
	case e := <-waitCh:
		return "", fmt.Errorf("proxy exited suddenly: %v", e)
	case <-t.C:
		return "", fmt.Errorf("timeout waiting for api proxy port")
	case <-portReady:
		if port == "" {
			return "", fmt.Errorf("got empty port from kubectl proxy")
		}
	}

	localPort := LocalApiPort
	maxRetries := 12
	retry := 0
	var lastError error
	var tun *Tunnel

	for {
		// try to start tunnel from localPort to proxy port
		tunnelAddress := fmt.Sprintf("%d:localhost:%s", localPort, port)
		tun = NewTunnel(k.Session, "L", tunnelAddress)
		// TODO if local port is busy, increase port and start again
		err := tun.Up()
		if err != nil {
			tun.Stop()
			lastError = fmt.Errorf("tunnel '%s': %v", tunnelAddress, err)
			localPort++
			retry++
			if retry >= maxRetries {
				tun = nil
				break
			}
			//return "",
		} else {
			break
		}
	}

	if tun == nil {
		return "", fmt.Errorf("tunnel up error: max retries reached, last error: %v", lastError)
	}

	k.tunnel = tun
	success = true

	go func() {
		err := k.tunnel.HealthMonitor()
		if err != nil {
			log.ErrorLn(err)
		}
	}()
	return fmt.Sprintf("%d", localPort), nil
}

func (k *KubeProxy) Stop() {
	if k == nil {
		return
	}
	if k.stop {
		return
	}
	if k.proxy != nil {
		k.proxy.Stop()
	}
	if k.tunnel != nil {
		k.tunnel.Stop()
	}
	k.stop = true
}

func (k *KubeProxy) Restart() error {
	k.Stop()
	_, err := k.Start()
	return err
}

// ScanPasswordOrLines is a split function for a Scanner that returns each line of
// text, stripped of any trailing end-of-line marker or if colon is occurred.
func ScanPasswordOrLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	//fmt.Printf("scan got %d bytes\n", len(data))
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexByte(data, ':'); i >= 0 {
		if strings.Contains(string(data), "assword") {
			// We have a password prompt.
			return i + 1, append(data[0:i], ':'), nil
		}
	}
	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		return bufio.ScanLines(data, atEOF)
	}
	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), data, nil
	}
	// Request more data.
	return 0, nil, nil
}

// ReadPassword prints prompt and read password from terminal without echoing symbols.
func ReadPassword(prompt string) (result string, err error) {
	fmt.Print(prompt)
	var data []byte
	if terminal.IsTerminal(int(os.Stdin.Fd())) {
		data, err = terminal.ReadPassword(int(os.Stdin.Fd()))
		result = string(data)
		// need to print a newline?
		//fmt.Println()
	} else {
		return "", fmt.Errorf("stdin is not a terminal, error reading password")
	}
	return result, err
}