// Copyright © 2018 Matthieu MARTIN <matthieu@agence-webup.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/agence-webup/backr/manager/config"

	"github.com/agence-webup/backr/manager/api"
	"github.com/agence-webup/backr/manager/proto"
	"google.golang.org/grpc"

	"github.com/agence-webup/backr/manager"

	"github.com/agence-webup/backr/manager/notifier/stateful"
	"github.com/agence-webup/backr/manager/process"
	"github.com/agence-webup/backr/manager/repositories/bolt"
	"github.com/agence-webup/backr/manager/repositories/s3"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.etcd.io/bbolt"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the daemon managing files lifecycle",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {

		// configuration
		log.Debug().Msg("fetching config")
		config := config.Get()

		// open a Bolt DB
		db, err := bbolt.Open(config.Bolt.Filepath, 0666, &bbolt.Options{Timeout: 1 * time.Second})
		if err != nil {
			fmt.Printf("unable to open BoltDB file: %v\n", err.Error())
			os.Exit(1)
		}
		defer db.Close()

		// prepare tools & repositories
		notifier := stateful.NewNotifier(db, config.SlackNotifier)
		projectRepo := bolt.NewProjectRepository(db)
		accountRepo := bolt.NewAccountRepository(db)
		fileRepo, err := s3.NewFileRepository(config.S3)
		if err != nil {
			log.Error().Str("err", err.Error()).Msg("unable to setup S3 file repository")
			os.Exit(1)
		}

		// prepare a context to allow cancelling of the 2 goroutines
		ctx, cancel := context.WithCancel(context.Background())
		wg := sync.WaitGroup{}

		// each goroutine must increment WaitGroup counter
		startProcess(ctx, &wg, projectRepo, fileRepo, notifier)
		startAPI(ctx, &wg, config, projectRepo, fileRepo, accountRepo)

		// prepare chan for listening to SIGINT signal
		sigint := make(chan os.Signal)
		signal.Notify(sigint, syscall.SIGINT, syscall.SIGTERM)
		// wait for SIGINT
		<-sigint
		log.Debug().Msg("received SIGINT. cleaning...")

		// cancelling context
		cancel()
		// wait for goroutines to be finished
		wg.Wait()

		log.Debug().Msg("exiting")
	},
}

func init() {
	daemonCmd.AddCommand(startCmd)
}

func startProcess(ctx context.Context, wg *sync.WaitGroup, projectRepo manager.ProjectRepository, fileRepo manager.FileRepository, notifier manager.Notifier) {

	wg.Add(1)

	log.Debug().Msg("process started")

	go func() {
		defer wg.Done()

		// prepare ticker
		tick := time.NewTicker(1 * time.Minute)

		for {
			select {
			case <-tick.C:
				referenceDate := time.Now()

				log.Debug().Time("ref_date", referenceDate).Msg("tick: executing process...")
				err := process.Execute(referenceDate, projectRepo, fileRepo)
				if err != nil {
					log.Error().Err(err).Msg("error executing process")
				}
				log.Debug().Msg("tick: process done")

				log.Debug().Msg("tick: starting notify...")
				err = process.Notify(projectRepo, notifier)
				if err != nil {
					log.Error().Err(err).Msg("unable to execute process")
				}
				log.Debug().Msg("tick: notify done")

				log.Debug().Msg("---------------")

			case <-ctx.Done():
				tick.Stop()
				log.Debug().Msg("process cleaning done.")
				return
			}
		}
	}()
}

func startAPI(ctx context.Context, wg *sync.WaitGroup, config manager.Config, projectRepo manager.ProjectRepository, fileRepo manager.FileRepository, accountRepo manager.AccountRepository) {

	wg.Add(1)

	addr := fmt.Sprintf("%s:%s", config.API.ListenIP, config.API.ListenPort)

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal().Str("addr", addr).Err(err).Msg("grpc: failed to listen on addr")
	}

	backrSrv := api.NewServer(projectRepo, fileRepo, accountRepo, config.API)
	srv := grpc.NewServer()
	proto.RegisterBackrApiServer(srv, backrSrv)

	log.Debug().Str("addr", addr).Msg("API started")

	go func() {
		srv.Serve(lis)
	}()

	go func() {
		defer wg.Done()

		<-ctx.Done()
		srv.GracefulStop()
		log.Debug().Msg("API stopped")
	}()
}
