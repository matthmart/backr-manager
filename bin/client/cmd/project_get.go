/*
Copyright © 2019 Matthieu MARTIN <matthieu@agence-webup.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/agence-webup/backr/manager/proto"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get [PROJECT_NAME]",
	Short: "Get infos on a project",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println("You must provide a project name.")
			os.Exit(1)
		}
		if len(args) > 1 {
			fmt.Println("You must provide only one project name.")
			os.Exit(1)
		}

		addr := viper.GetString("endpoint")
		conn, err := grpcConnect(addr)
		if err != nil {
			fmt.Println("unable to dial to addr")
			os.Exit(1)
		}
		defer conn.Close()

		client := proto.NewBackrApiClient(conn)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		req := &proto.GetProjectRequest{Name: args[0]}

		resp, err := client.GetProject(ctx, req)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		p := resp.Project

		showAll, err := cmd.Flags().GetBool("all")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if showAll {
			w := tabwriter.NewWriter(os.Stdout, 1, 1, 3, ' ', 0)
			fmt.Fprintf(w, "%v\t%v\t\n", "PROJECT NAME", "CREATED AT")
			fmt.Fprintf(w, "%v\t%v\t\n", p.Name, time.Unix(p.CreatedAt, 0))
			w.Flush()
			fmt.Println("")
		}

		showFiles, err := cmd.Flags().GetBool("files")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		if showFiles || showAll {
			w := tabwriter.NewWriter(os.Stdout, 1, 1, 3, ' ', 0)
			for _, r := range p.Rules {
				fmt.Printf("\033[1;36m%s\033[0m\n", fmt.Sprintf("%d.%d (next: %v)", r.Count, r.MinAge, time.Unix(r.NextDate, 0)))
				if r.Error > 0 {
					fmt.Printf("%v %v\n", fmt.Sprintf(ErrorColor, "error:"), r.Error.String())
				}
				fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t\n", "PATH", "DATE", "EXPIRE AT", "SIZE", "ERROR")
				for _, f := range r.Files {
					errTxt := "-"
					if f.Error > 0 {
						errTxt = fmt.Sprintf(ErrorColor, f.Error.String())
					}
					fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t\n", f.Path, time.Unix(f.Date, 0), time.Unix(f.Expiration, 0), f.Size, errTxt)
				}
			}
			w.Flush()
			fmt.Println("")
		}

	},
}

func init() {
	projectsCmd.AddCommand(getCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// getCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// getCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	getCmd.Flags().BoolP("files", "f", true, "Display files associated to project")
	getCmd.Flags().BoolP("all", "a", false, "Display all informations associated to project")
}
