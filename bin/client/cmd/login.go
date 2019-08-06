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
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/agence-webup/backr/manager/proto"
	"github.com/chzyer/readline"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
)

// loginCmd represents the login command
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login using username and password, and save token into a file in $HOME directory (.backr_auth)",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {

		rl, err := readline.New("username: ")
		if err != nil {
			panic(err)
		}
		defer rl.Close()

		username, err := rl.Readline()
		if err != nil && err != io.EOF {
			fmt.Printf("unable to get username: %v", err)
			os.Exit(1)
		}

		password, err := rl.ReadPassword("password: ")
		if err != nil {
			fmt.Printf("unable to get password: %v", err)
			os.Exit(1)
		}

		conn, err := grpcConnect()
		if err != nil {
			fmt.Printf("unable to dial to addr: %v\n", err)
			os.Exit(1)
		}
		defer conn.Close()

		client := proto.NewBackrApiClient(conn)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		req := &proto.AuthenticateAccountRequest{Username: username, Password: string(password)}
		resp, err := client.AuthenticateAccount(ctx, req)
		if err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}

		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		cacheFilepath := filepath.Join(home, ".backr_auth")
		cacheFile, err := os.OpenFile(cacheFilepath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		defer cacheFile.Close()
		if err != nil {
			fmt.Println("unable to save token into file")
			fmt.Println("")
			fmt.Println("token:", resp.Token)
		}

		fmt.Fprint(cacheFile, resp.Token)
		fmt.Println("token saved in", cacheFilepath)
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// loginCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// loginCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
