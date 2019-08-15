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
	"strconv"
	"strings"
	"time"

	"github.com/agence-webup/backr/manager/proto"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new project",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("create called")

		name, err := cmd.Flags().GetString("name")
		if err != nil {
			fmt.Printf("unable to get 'name' param: %v\n", err)
			os.Exit(1)
		}

		rawRules, err := cmd.Flags().GetStringSlice("rule")
		if err != nil {
			fmt.Printf("unable to get 'rule' params: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(rawRules)

		rules := []*proto.Rule{}
		for _, r := range rawRules {
			comps := strings.Split(r, ".")
			if len(comps) == 2 {
				count, err := strconv.ParseInt(comps[0], 10, 32)
				if err != nil || count == 0 {
					continue
				}
				minAge, err := strconv.ParseInt(comps[1], 10, 32)
				if err != nil || minAge == 0 {
					continue
				}
				rules = append(rules, &proto.Rule{Count: int32(count), MinAge: int32(minAge)})
			}
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

		req := &proto.CreateProjectRequest{
			Name:  name,
			Rules: rules,
		}
		_, err = client.CreateProject(ctx, req)
		if err != nil {
			fmt.Printf("error: %v", err)
		}
	},
}

func init() {
	projectsCmd.AddCommand(createCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// createCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// createCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	createCmd.Flags().StringP("name", "n", "", "Name of the project. Should be unique")
	createCmd.Flags().StringSliceP("rule", "r", []string{}, "Define a rule with this pattern: COUNT.MIN_AGE  (i.e -r 3.1)")

	createCmd.MarkFlagRequired("name")
	createCmd.MarkFlagRequired("rule")
}
