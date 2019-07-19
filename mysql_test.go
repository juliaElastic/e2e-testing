package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/DATA-DOG/godog"
	"github.com/DATA-DOG/godog/colors"
)

var metricbeatService Service
var mysqlService Service

var opt = godog.Options{Output: colors.Colored(os.Stdout)}

func init() {
	godog.BindFlags("godog.", flag.CommandLine, &opt)
}

func TestMain(m *testing.M) {
	flag.Parse()
	opt.Paths = flag.Args()

	status := godog.RunWithOptions("MySQL", func(s *godog.Suite) {
		FeatureContext(s)
	}, opt)

	if st := m.Run(); st > status {
		status = st
	}
	os.Exit(status)
}

func metricbeatIsInstalledAndConfiguredForMySQLModule(metricbeatVersion string) error {
	metricbeatService, err := NewMetricbeatService(metricbeatVersion, mysqlService)
	if err != nil {
		return err
	}
	if metricbeatService == nil {
		return fmt.Errorf("Could not create Metricbeat %s service for MySQL", metricbeatVersion)
	}

	container, err := metricbeatService.Run()
	if err != nil || container == nil {
		return fmt.Errorf("Could not run Metricbeat %s: %v", metricbeatVersion, err)
	}

	ctx := context.Background()

	ip, err := container.Host(ctx)
	if err != nil {
		return fmt.Errorf("Could not run Metricbeat %s: %v", metricbeatVersion, err)
	}
	fmt.Printf("Metricbeat %s is running configured for MySQL on IP %s\n", metricbeatVersion, ip)

	return nil
}

func metricbeatOutputsMetricsToTheFile(fileName string) error {
	time.Sleep(20 * time.Second)

	dir, _ := os.Getwd()

	if _, err := os.Stat(dir + "/outputs/" + fileName); os.IsNotExist(err) {
		return fmt.Errorf("The output file %s does not exist", fileName)
	}

	fmt.Println("Metricbeat outputs to " + fileName)
	return nil
}

func mySQLIsRunningOnPort(mysqlVersion string, port string) error {
	mysqlService = NewMySQLService(mysqlVersion, port)

	container, err := mysqlService.Run()
	if err != nil {
		return fmt.Errorf("Could not run MySQL %s: %v", mysqlVersion, err)
	}

	ctx := context.Background()

	ip, err := container.Host(ctx)
	if err != nil {
		return fmt.Errorf("Could not run MySQL %s: %v", mysqlVersion, err)
	}

	fmt.Printf("MySQL %s is running on %s:%s\n", mysqlVersion, ip, port)

	return nil
}

func FeatureContext(s *godog.Suite) {
	s.Step(`^MySQL "([^"]*)" is running on port "([^"]*)"$`, mySQLIsRunningOnPort)
	s.Step(`^metricbeat "([^"]*)" is installed and configured for MySQL module$`, metricbeatIsInstalledAndConfiguredForMySQLModule)
	s.Step(`^metricbeat outputs metrics to the file "([^"]*)"$`, metricbeatOutputsMetricsToTheFile)

	s.BeforeScenario(func(interface{}) {
		fmt.Println("Before scenario...")
		cleanUpOutputs()
	})

	s.AfterScenario(func(interface{}, error) {
		fmt.Println("After scenario...")
	})
}

func cleanUpOutputs() {
	dir, _ := os.Getwd()

	files, err := filepath.Glob(dir + "/outputs/*.metrics")
	if err != nil {
		fmt.Println("Cannot remove outputs :(")
	}
	for _, f := range files {
		if err := os.Remove(f); err != nil {
			fmt.Printf("Cannot remove output file %s :(\n", f)
		}
	}
}
