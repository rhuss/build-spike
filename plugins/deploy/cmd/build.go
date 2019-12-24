/*
Copyright © 2019 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package main

import (
	"fmt"
	"github.com/knative-community/build-spike/plugins/deploy/tekton"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	tektoncdclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc" // from https://github.com/kubernetes/client-go/issues/345
	"k8s.io/client-go/tools/clientcmd"
	"os"
)

const (
	// How often to retry in case of an optimistic lock error when replacing a service (--force)
	MaxUpdateRetries = 3
	// Timeout to wait service creation
	MaxTimeout = 300
)

var cfgFile string
var kubeconfig string

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build Knative service image",
	Example: `
  # Build from source code into an image
  # ( related: https://github.com/knative-community/build-spike/blob/master/plugins/kn-services/doc/deploy-git-resource.md )
  kn build example-image --git-url https://github.com/bluebosh/knap-example -git-revision master --builder kaniko --saved-image us.icr.io/test/example-image --serviceaccount default`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("")
		if len(args) < 1 {
			fmt.Println("[ERROR] Name cannot be empty, please set as the first parameter")
			cmd.Help()
			os.Exit(0)
		}
		name := args[0]

		serviceAccount := cmd.Flag("serviceaccount").Value.String()
		namespace := cmd.Flag("namespace").Value.String()

		// Config kubeconfig
		kubeconfig = cmd.Flag("kubeconfig").Value.String()
		if kubeconfig == "" {
			kubeconfig = os.Getenv("KUBECONFIG")
		}
		cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			fmt.Println("[ERROR] Parsing kubeconfig error:", err)
			os.Exit(1)
		}
		client, err := tektoncdclientset.NewForConfig(cfg)
		if err != nil {
			fmt.Println("[ERROR] Building kubeconfig error:", err)
			os.Exit(1)
		}
		tektonClient := tekton.NewTektonClient(client.TektonV1alpha1(), namespace)

		builder := cmd.Flag("builder").Value.String()
		if builder == "" {
			fmt.Println("[ERROR] Builder cannot be empty, please use --builder to set")
			os.Exit(1)
		}

		gitUrl := cmd.Flag("git-url").Value.String()
		gitRevision := cmd.Flag("git-revision").Value.String()
		file := cmd.Flag("file").Value.String()

		image := cmd.Flag("saved-image").Value.String()
		if image == "" {
			fmt.Println("[ERROR] Saved-image cannot be empty, please use --saved-image to set")
			os.Exit(1)
		}

		if len(gitUrl) > 0 && len(file) == 0 {
			fmt.Println("[INFO] Build from git repository into an image")
			err = tektonClient.BuildFromGit(name, builder, gitUrl, gitRevision, image, serviceAccount, namespace)
			if err != nil {
				fmt.Println("[ERROR] Building image from git error:", err)
				os.Exit(1)
			}
			fmt.Println("[INFO] Create image", image, "from git repo")
		} else if len(file) > 0 && len(gitUrl) == 0 {
			fmt.Println("[INFO] Build from function file into an image")
			err = tektonClient.BuildFromFunctionFile(name, builder, file, image, serviceAccount, namespace)
			if err != nil {
				fmt.Println("[ERROR] Building image from function file error:", err)
				os.Exit(1)
			}
			fmt.Println("[INFO] Create image", image, "from function file")
		} else {
			fmt.Println("[ERROR] Do not support set --git-url and --file parameters together")
			os.Exit(1)
		}
	},
}

func Execute() {
	if err := buildCmd.Execute(); err != nil {
	  fmt.Println(err)
	  os.Exit(1)
	}
  }

func init() {
	cobra.OnInitialize(initConfig)
	buildCmd.PersistentFlags().StringP("kubeconfig", "", "", "kube config file (default is KUBECONFIG from ENV property)")
	buildCmd.Flags().StringP("builder", "b", "", "builder of source-to-image task")
	buildCmd.Flags().StringP( "git-url", "u","", "[Git] url of git repo")
	buildCmd.Flags().StringP( "git-revision", "r","master", "[Git] revision of git repo")
	buildCmd.Flags().StringP( "file", "f","", "[Function] File of function which snippet of code without http server")
	buildCmd.Flags().StringP("saved-image", "i", "", "generated saved image path")
	buildCmd.Flags().StringP("serviceaccount", "s", "default", "service account to push image")
	buildCmd.Flags().StringP( "namespace", "n","default", "namespace of build")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
	  // Use config file from the flag.
	  viper.SetConfigFile(cfgFile)
	} else {
	  // Find home directory.
	  home, err := homedir.Dir()
	  if err != nil {
		fmt.Println(err)
		os.Exit(1)
	  }
  
	  // Search config in home directory with name ".app" (without extension).
	  viper.AddConfigPath(home)
	  viper.SetConfigName(".app")
	}
  
	viper.AutomaticEnv() // read in environment variables that match
  
	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
	  fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
  }

func main() {
    Execute()
    os.Exit(0)
}