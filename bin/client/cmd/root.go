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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	homedir "github.com/mitchellh/go-homedir"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "backrctl",
	Short: "CLI tool to interact with a running Backr manager instance",
	Long:  ``,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	//	Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().String("endpoint", "127.0.0.1:3000", "Endpoint of the Backr instance")
	viper.BindPFlag("endpoint", rootCmd.PersistentFlags().Lookup("endpoint"))

	viper.SetEnvPrefix("backrctl")
	viper.AutomaticEnv() // read in environment variables that match
}

func grpcConnect(addr string) (*grpc.ClientConn, error) {
	// try to find an auth token
	cleanToken := ""
	// Find home directory.
	home, err := homedir.Dir()
	if err == nil {
		// get token file
		token, err := ioutil.ReadFile(filepath.Join(home, ".backr_auth"))
		if err == nil {
			// fmt.Println("unable to get auth token from file ~/.backr_auth")
			// fmt.Println("You must authenticate using `backrctl login`")
			// return nil, err
			cleanToken = strings.ReplaceAll(string(token), "\n", "")
		}
	}

	return grpc.Dial(addr, grpc.WithInsecure(), grpc.WithPerRPCCredentials(tokenAuth{token: cleanToken}))
}
