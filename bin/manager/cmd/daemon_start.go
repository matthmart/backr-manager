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
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/agence-webup/backr/manager/api"
	"github.com/agence-webup/backr/manager/proto"
	"google.golang.org/grpc"

	"github.com/agence-webup/backr/manager"

	"github.com/agence-webup/backr/manager/notifier/basic"
	"github.com/agence-webup/backr/manager/process"
	"github.com/agence-webup/backr/manager/repositories/inmem"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the daemon managing files lifecycle",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {

		// configuration
		log.Debug().Msg("fetching config")
		//config := config.Get()

		// prepare tools
		notifier := basic.NewNotifier()
		projectRepo := inmem.NewProjectRepository()
		fileRepo := inmem.NewFileRepository()
		// if err != nil {
		// 	log.Fatal().Str("err", err.Error()).Msg("unable to setup S3 client")
		// }

		// simulate files
		// inmem.CreateFakeFile(fileRepo, manager.File{Path: "fera/test1.tar.gz", Size: 450, Date: time.Date(2018, 12, 1, 5, 0, 0, 0, time.Local)})
		// inmem.CreateFakeFile(fileRepo, manager.File{Path: "fera/test2.tar.gz", Size: 455, Date: time.Date(2018, 12, 2, 5, 0, 0, 0, time.Local)})

		// done := make(chan int, 1)

		// // prepare chan for listening to SIGINT signal
		// sigint := make(chan os.Signal)
		// signal.Notify(sigint, syscall.SIGINT, syscall.SIGTERM)

		ctx, cancel := context.WithCancel(context.Background())
		wg := sync.WaitGroup{}

		// each goroutine must increment WaitGroup counter
		startProcess(ctx, &wg, projectRepo, fileRepo, notifier)
		startAPI(ctx, &wg, projectRepo)

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

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// startCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// startCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func startProcess(ctx context.Context, wg *sync.WaitGroup, projectRepo manager.ProjectRepository, fileRepo manager.FileRepository, notifier manager.Notifier) {

	wg.Add(1)

	log.Debug().Msg("process started")

	go func() {
		defer wg.Done()

		// prepare ticker
		tick := time.NewTicker(1 * time.Second)

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

func startAPI(ctx context.Context, wg *sync.WaitGroup, projectRepo manager.ProjectRepository) {

	wg.Add(1)

	addr := "127.0.0.1:3000"

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal().Str("addr", addr).Err(err).Msg("grpc: failed to listen on addr")
	}

	backrSrv := api.NewServer(projectRepo)
	srv := grpc.NewServer()
	proto.RegisterBackrApiServer(srv, backrSrv)

	log.Debug().Str("addr", addr).Msg("API started")

	// http.HandleFunc("/healthz", func(w http.ResponseWriter, req *http.Request) {
	// 	w.Write([]byte("OK\n"))
	// })

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

// func getSimulatedDates(fileRepo manager.FileRepository) []func() time.Time {
// 	return []func() time.Time{
// 		func() time.Time {
// 			inmem.CreateFakeFile(fileRepo, manager.File{Path: "fera/test-3.tar.gz", Size: 450, Date: time.Date(2018, 11, 28, 5, 0, 0, 0, time.Local)})
// 			inmem.CreateFakeFile(fileRepo, manager.File{Path: "fera/test-2.tar.gz", Size: 450, Date: time.Date(2018, 11, 29, 5, 0, 0, 0, time.Local)})
// 			inmem.CreateFakeFile(fileRepo, manager.File{Path: "fera/test-1.tar.gz", Size: 450, Date: time.Date(2018, 11, 30, 5, 0, 0, 0, time.Local)})
// 			inmem.CreateFakeFile(fileRepo, manager.File{Path: "fera/test1.tar.gz", Size: 450, Date: time.Date(2018, 12, 1, 5, 0, 0, 0, time.Local)})
// 			return time.Date(2018, 12, 1, 8, 0, 0, 0, time.Local)
// 		},
// 		// func() time.Time {
// 		// 	return time.Date(2018, 12, 1, 12, 0, 0, 0, time.Local)
// 		// },
// 		func() time.Time {
// 			return time.Date(2018, 12, 1, 20, 0, 0, 0, time.Local)
// 		},
// 		func() time.Time {
// 			inmem.CreateFakeFile(fileRepo, manager.File{Path: "fera/test2.tar.gz", Size: 450, Date: time.Date(2018, 12, 2, 5, 0, 0, 0, time.Local)})
// 			return time.Date(2018, 12, 2, 8, 0, 0, 0, time.Local)
// 		},
// 		// func() time.Time {
// 		// 	return time.Date(2018, 12, 2, 12, 0, 0, 0, time.Local)
// 		// },
// 		func() time.Time {
// 			return time.Date(2018, 12, 2, 20, 0, 0, 0, time.Local)
// 		},
// 		func() time.Time {
// 			inmem.CreateFakeFile(fileRepo, manager.File{Path: "fera/test3.tar.gz", Size: 450, Date: time.Date(2018, 12, 3, 5, 0, 0, 0, time.Local)})
// 			inmem.CreateFakeFile(fileRepo, manager.File{Path: "truite/truite1.tar.gz", Size: 1000, Date: time.Date(2018, 12, 3, 5, 0, 0, 0, time.Local)})
// 			return time.Date(2018, 12, 3, 8, 0, 0, 0, time.Local)
// 		},
// 		// func() time.Time {
// 		// 	return time.Date(2018, 12, 3, 12, 0, 0, 0, time.Local)
// 		// },
// 		func() time.Time {
// 			return time.Date(2018, 12, 3, 20, 0, 0, 0, time.Local)
// 		},
// 		func() time.Time {
// 			// inmem.CreateFakeFile(fileRepo, manager.File{Path: "fera/test4.tar.gz", Size: 450, Date: time.Date(2018, 12, 4, 5, 0, 0, 0, time.Local)})
// 			return time.Date(2018, 12, 4, 8, 0, 0, 0, time.Local)
// 		},
// 		// func() time.Time {
// 		// 	return time.Date(2018, 12, 4, 12, 0, 0, 0, time.Local)
// 		// },
// 		func() time.Time {
// 			return time.Date(2018, 12, 4, 20, 0, 0, 0, time.Local)
// 		},
// 		func() time.Time {
// 			// inmem.CreateFakeFile(fileRepo, manager.File{Path: "fera/test4.tar.gz", Size: 450, Date: time.Date(2018, 12, 4, 5, 0, 0, 0, time.Local)})
// 			return time.Date(2018, 12, 5, 8, 0, 0, 0, time.Local)
// 		},
// 		// func() time.Time {
// 		// 	return time.Date(2018, 12, 5, 12, 0, 0, 0, time.Local)
// 		// },
// 		func() time.Time {
// 			return time.Date(2018, 12, 5, 20, 0, 0, 0, time.Local)
// 		},
// 		func() time.Time {
// 			inmem.CreateFakeFile(fileRepo, manager.File{Path: "fera/test4.tar.gz", Size: 450, Date: time.Date(2018, 12, 6, 5, 0, 0, 0, time.Local)})
// 			return time.Date(2018, 12, 6, 8, 0, 0, 0, time.Local)
// 		},
// 		// func() time.Time {
// 		// 	return time.Date(2018, 12, 6, 12, 0, 0, 0, time.Local)
// 		// },
// 		func() time.Time {
// 			return time.Date(2018, 12, 6, 20, 0, 0, 0, time.Local)
// 		},
// 		func() time.Time {
// 			inmem.CreateFakeFile(fileRepo, manager.File{Path: "fera/test5.tar.gz", Size: 150, Date: time.Date(2018, 12, 7, 5, 0, 0, 0, time.Local)})
// 			return time.Date(2018, 12, 7, 8, 0, 0, 0, time.Local)
// 		},
// 		func() time.Time {
// 			return time.Date(2018, 12, 7, 20, 0, 0, 0, time.Local)
// 		},
// 		func() time.Time {
// 			inmem.CreateFakeFile(fileRepo, manager.File{Path: "fera/test6.tar.gz", Size: 450, Date: time.Date(2018, 12, 8, 5, 0, 0, 0, time.Local)})
// 			return time.Date(2018, 12, 8, 8, 0, 0, 0, time.Local)
// 		},
// 		func() time.Time {
// 			return time.Date(2018, 12, 8, 20, 0, 0, 0, time.Local)
// 		},
// 		func() time.Time {
// 			inmem.CreateFakeFile(fileRepo, manager.File{Path: "fera/test7.tar.gz", Size: 450, Date: time.Date(2018, 12, 9, 5, 0, 0, 0, time.Local)})
// 			return time.Date(2018, 12, 9, 8, 0, 0, 0, time.Local)
// 		},
// 		func() time.Time {
// 			inmem.CreateFakeFile(fileRepo, manager.File{Path: "fera/test8.tar.gz", Size: 460, Date: time.Date(2018, 12, 10, 5, 0, 0, 0, time.Local)})
// 			return time.Date(2018, 12, 10, 8, 0, 0, 0, time.Local)
// 		},
// 		func() time.Time {
// 			inmem.CreateFakeFile(fileRepo, manager.File{Path: "fera/test9.tar.gz", Size: 450, Date: time.Date(2018, 12, 11, 5, 0, 0, 0, time.Local)})
// 			return time.Date(2018, 12, 11, 8, 0, 0, 0, time.Local)
// 		},
// 		func() time.Time {
// 			inmem.CreateFakeFile(fileRepo, manager.File{Path: "fera/test10.tar.gz", Size: 450, Date: time.Date(2018, 12, 12, 5, 0, 0, 0, time.Local)})
// 			return time.Date(2018, 12, 12, 8, 0, 0, 0, time.Local)
// 		},
// 		func() time.Time {
// 			return time.Date(2018, 12, 15, 20, 0, 0, 0, time.Local)
// 		},
// 		func() time.Time {
// 			inmem.CreateFakeFile(fileRepo, manager.File{Path: "fera/test11.tar.gz", Size: 450, Date: time.Date(2018, 12, 16, 5, 0, 0, 0, time.Local)})
// 			return time.Date(2018, 12, 16, 8, 0, 0, 0, time.Local)
// 		},
// 	}
// }

func getSimulatedDates(fileRepo manager.FileRepository) []func() time.Time {
	return []func() time.Time{
		func() time.Time {
			return time.Date(2018, 12, 1, 8, 0, 0, 0, time.Local)
		},
		// func() time.Time {
		// 	return time.Date(2018, 12, 1, 12, 0, 0, 0, time.Local)
		// },
		func() time.Time {
			return time.Date(2018, 12, 1, 20, 0, 0, 0, time.Local)
		},
		func() time.Time {
			return time.Date(2018, 12, 2, 8, 0, 0, 0, time.Local)
		},
		// func() time.Time {
		// 	return time.Date(2018, 12, 2, 12, 0, 0, 0, time.Local)
		// },
		func() time.Time {
			return time.Date(2018, 12, 2, 20, 0, 0, 0, time.Local)
		},
		func() time.Time {
			return time.Date(2018, 12, 3, 8, 0, 0, 0, time.Local)
		},
		// func() time.Time {
		// 	return time.Date(2018, 12, 3, 12, 0, 0, 0, time.Local)
		// },
		func() time.Time {
			return time.Date(2018, 12, 3, 20, 0, 0, 0, time.Local)
		},
		func() time.Time {
			// inmem.CreateFakeFile(fileRepo, manager.File{Path: "fera/test4.tar.gz", Size: 450, Date: time.Date(2018, 12, 4, 5, 0, 0, 0, time.Local)})
			return time.Date(2018, 12, 4, 8, 0, 0, 0, time.Local)
		},
		// func() time.Time {
		// 	return time.Date(2018, 12, 4, 12, 0, 0, 0, time.Local)
		// },
		func() time.Time {
			return time.Date(2018, 12, 4, 20, 0, 0, 0, time.Local)
		},
		func() time.Time {
			// inmem.CreateFakeFile(fileRepo, manager.File{Path: "fera/test4.tar.gz", Size: 450, Date: time.Date(2018, 12, 4, 5, 0, 0, 0, time.Local)})
			return time.Date(2018, 12, 5, 8, 0, 0, 0, time.Local)
		},
		// func() time.Time {
		// 	return time.Date(2018, 12, 5, 12, 0, 0, 0, time.Local)
		// },
		func() time.Time {
			return time.Date(2018, 12, 5, 20, 0, 0, 0, time.Local)
		},
		func() time.Time {
			inmem.CreateFakeFile(fileRepo, manager.File{Path: "fera/test4.tar.gz", Size: 450, Date: time.Date(2018, 12, 6, 5, 0, 0, 0, time.Local)})
			return time.Date(2018, 12, 6, 8, 0, 0, 0, time.Local)
		},
		// func() time.Time {
		// 	return time.Date(2018, 12, 6, 12, 0, 0, 0, time.Local)
		// },
		func() time.Time {
			return time.Date(2018, 12, 6, 20, 0, 0, 0, time.Local)
		},
		func() time.Time {
			inmem.CreateFakeFile(fileRepo, manager.File{Path: "fera/test5.tar.gz", Size: 150, Date: time.Date(2018, 12, 7, 5, 0, 0, 0, time.Local)})
			return time.Date(2018, 12, 7, 8, 0, 0, 0, time.Local)
		},
		func() time.Time {
			return time.Date(2018, 12, 7, 20, 0, 0, 0, time.Local)
		},
		func() time.Time {
			inmem.CreateFakeFile(fileRepo, manager.File{Path: "fera/test6.tar.gz", Size: 450, Date: time.Date(2018, 12, 8, 5, 0, 0, 0, time.Local)})
			return time.Date(2018, 12, 8, 8, 0, 0, 0, time.Local)
		},
		func() time.Time {
			return time.Date(2018, 12, 8, 20, 0, 0, 0, time.Local)
		},
		func() time.Time {
			inmem.CreateFakeFile(fileRepo, manager.File{Path: "fera/test7.tar.gz", Size: 450, Date: time.Date(2018, 12, 9, 5, 0, 0, 0, time.Local)})
			return time.Date(2018, 12, 9, 8, 0, 0, 0, time.Local)
		},
		func() time.Time {
			inmem.CreateFakeFile(fileRepo, manager.File{Path: "fera/test8.tar.gz", Size: 460, Date: time.Date(2018, 12, 10, 5, 0, 0, 0, time.Local)})
			return time.Date(2018, 12, 10, 8, 0, 0, 0, time.Local)
		},
		func() time.Time {
			// inmem.CreateFakeFile(fileRepo, manager.File{Path: "fera/test9.tar.gz", Size: 450, Date: time.Date(2018, 12, 11, 5, 0, 0, 0, time.Local)})
			return time.Date(2018, 12, 11, 8, 0, 0, 0, time.Local)
		},
		func() time.Time {
			// inmem.CreateFakeFile(fileRepo, manager.File{Path: "fera/test10.tar.gz", Size: 450, Date: time.Date(2018, 12, 12, 5, 0, 0, 0, time.Local)})
			return time.Date(2018, 12, 12, 8, 0, 0, 0, time.Local)
		},
		func() time.Time {
			return time.Date(2018, 12, 15, 20, 0, 0, 0, time.Local)
		},
		func() time.Time {
			// inmem.CreateFakeFile(fileRepo, manager.File{Path: "fera/test11.tar.gz", Size: 450, Date: time.Date(2018, 12, 16, 5, 0, 0, 0, time.Local)})
			return time.Date(2018, 12, 16, 8, 0, 0, 0, time.Local)
		},
	}
}
