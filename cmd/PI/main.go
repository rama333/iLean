package main

import (
	"github.com/sirupsen/logrus"
	"iLean/agent"
	"iLean/config"
	"time"
)

func main() {

	if err := run(); err != nil {
		logrus.Fatal(err)
	}

}

func run() error {

	logrus.Info("run application PI")

	var st time.Time
	defer func() {
		logrus.WithField("shutdown_time", time.Now().Sub(st)).Info("stopped")
	}()

	config, err := config.LoadConfig("/opt/config/pi.conf")

	if err != nil {
		logrus.Fatal("failed load config", err)
	}

	agent, err := agent.NewAgent(&config)

	if err != nil {
		return err
	}

	agent.ProcessingStream()

	defer agent.Close()

	return nil
}
