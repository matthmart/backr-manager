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
	"time"

	"github.com/agence-webup/backr/manager/proto"
	"github.com/spf13/cobra"
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

		conn, err := grpcConnect()
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

		fmt.Printf("%v (created at %v)\n", p.Name, time.Unix(p.CreatedAt, 0))
		for _, r := range p.Rules {
			fmt.Printf(" - min_age=%v count=%v\n", r.MinAge, r.Count)
		}
		fmt.Println("")

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
}
