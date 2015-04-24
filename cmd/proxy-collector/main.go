package main

import (
	"flag"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	log "github.com/Sirupsen/logrus"
	"github.com/soh335/proxy-collector/proxy"
)

var (
	host     = flag.String("host", "0.0.0.0", "host")
	port     = flag.String("port", "7243", "port")
	loglevel = flag.String("loglevel", "info", "loglevel")
	config   = flag.String("config", "proxy-collector.json", "config json")
)

func main() {
	flag.Parse()

	lvl, err := log.ParseLevel(*loglevel)
	if err != nil {
		log.Fatal(err)
	}
	log.SetLevel(lvl)

	if err := _main(); err != nil {
		log.Fatal(err)
	}
}

func _main() error {
	c, err := LoadConfig(*config)
	if err != nil {
		return err
	}
	targetList, err := c.TargetListAsURL()
	if err != nil {
		return err
	}

	h := proxy.NewProxy(targetList)
	h.BodyFallback = c.BodyFallback

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGHUP)
	go func() {
		for {
			sig := <-sigChan
			switch sig {
			case syscall.SIGHUP:
				log.Info("receive hup signal. reload config...")
				c, err := LoadConfig(*config)
				if err != nil {
					log.Errorf("reload config failed:%v", err)
					break
				}

				targetList, err := c.TargetListAsURL()
				if err != nil {
					log.Errorf("reload config failed:%v", err)
					break
				}

				h.M.Lock()
				h.TargetList = targetList
				h.BodyFallback = h.BodyFallback
				h.M.Unlock()
				log.Infof("reload config done")
			}
		}
	}()

	addr := net.JoinHostPort(*host, *port)
	log.Infof("start:%v", addr)
	return http.ListenAndServe(addr, h)
}
