package main

import (
	"fmt"
	"net/http"
	"os"

	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv2"
	"code.cloudfoundry.org/cli/integration/helpers"
	"code.cloudfoundry.org/lager"
	"github.com/georgi-lozev/cf-broker-proxy/broker"
	"github.com/pivotal-cf/brokerapi"
)

func main() {
	brokerLogger := lager.NewLogger("cf-broker-proxy")
	brokerLogger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))
	brokerLogger.RegisterSink(lager.NewWriterSink(os.Stderr, lager.ERROR))
	brokerLogger.Info("starting up broker...")

	ccClient := authenticate()
	serviceBroker := &broker.CFBrokerProxy{APIClient: ccClient, Logger: brokerLogger}

	brokerCredentials := brokerapi.BrokerCredentials{
		Username: "admin",
		Password: "password",
	}

	brokerAPI := brokerapi.New(serviceBroker, brokerLogger, brokerCredentials)

	http.Handle("/", brokerAPI)

	var port string
	if port = os.Getenv("PORT"); len(port) == 0 {
		port = "8080"
	}

	brokerLogger.Fatal("http-listen", http.ListenAndServe(":"+port, nil))
}

func authenticate() *ccv2.Client {
	ccClient, err := helpers.CreateCCV2Client()

	if err != nil {
		panic(fmt.Sprintf("Unable to create CC client due to %s", err))
	}

	return ccClient
}
