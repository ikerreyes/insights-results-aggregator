// Copyright 2020 Red Hat, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"strconv"

	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-results-aggregator/conf"
	"github.com/RedHatInsights/insights-results-aggregator/migration"
	"github.com/RedHatInsights/insights-results-aggregator/server"
)

var (
	serverInstance                                               *server.HTTPServer
	serverInstanceIsStarting, finishServerInstanceInitialization = context.WithCancel(context.Background())
)

// startServer starts the server or returns an error
func startServer() error {
	defer finishServerInstanceInitialization()

	dbStorage, err := createStorage()
	if err != nil {
		return err
	}
	defer closeStorage(dbStorage)

	serverCfg := conf.GetServerConfiguration()

	serverInstance = server.New(serverCfg, dbStorage)

	// fill-in additional info used by /info endpoint handler
	fillInInfoParams(serverInstance.InfoParams)

	// try to retrieve the actual DB migration version
	// and add it into the `params` map
	currentVersion, err := migration.GetDBVersion(dbStorage.GetConnection())
	if err != nil {
		const msg = "Unable to retrieve DB migration version"
		log.Error().Err(err).Msg(msg)
		serverInstance.InfoParams["DB_version"] = msg
	} else {
		serverInstance.InfoParams["DB_version"] = strconv.Itoa(int(currentVersion))
	}

	err = serverInstance.Start(finishServerInstanceInitialization)
	if err != nil {
		log.Error().Err(err).Msg("HTTP(s) start error")
		return err
	}

	return nil
}

func stopServer() error {
	waitForServerToStartOrFail()

	if serverInstance == nil {
		return nil
	}

	err := serverInstance.Stop(context.Background())
	if err != nil {
		log.Error().Err(err).Msg("HTTP(s) server stop error")
		return err
	}

	return nil
}

func waitForServerToStartOrFail() {
	log.Info().Msg("waiting for server to start")
	<-serverInstanceIsStarting.Done()
}
