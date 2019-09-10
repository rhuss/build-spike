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
package cmd

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/knative-community/build-spike/plugins/app/tekton"
	"github.com/spf13/cobra"
	tektoncdclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc" // from https://github.com/kubernetes/client-go/issues/345
	"k8s.io/client-go/tools/clientcmd"
	"os"
)

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build knative application image",
	Example: `
  # Build from Git repository into an image
  # ( related: https://github.com/knative-community/build-spike/blob/master/plugins/app/doc/deploy-git-resource.md )
  app build test -u https://github.com/bluebosh/knap-example -r master`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("")
		if len(args) < 1 {
			fmt.Println("[Error] Require at least", color.CyanString("two"), "parameters\n")
			os.Exit(1)
		}
		name := args[0]

		url := cmd.Flag("url").Value.String()
		revision := cmd.Flag("revision").Value.String()
		namespace := cmd.Flag("namespace").Value.String()

		// Config kubeconfig
		kubeconfig = rootCmd.Flag("kubeconfig").Value.String()
		if kubeconfig == "" {
			kubeconfig = os.Getenv("KUBECONFIG")
		}
		cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			fmt.Errorf("\nError parsing kubeconfig: %v", err)
		}

		imagename := ""
		if len(url) > 0 {
			client, err := tektoncdclientset.NewForConfig(cfg)
			if err != nil {
				fmt.Errorf("\nError building kubeconfig: %v", err)
			}
			tektonClient := tekton.NewTektonClient(client.TektonV1alpha1(), namespace)

			imagename, err = tektonClient.BuildFromGit(name, namespace, url, revision)
			if err != nil {
				fmt.Errorf("\nError building application: %v", err)
			}
			fmt.Println("[INFO] Generate image", color.CyanString(imagename), "from git repo for application", color.BlueString(name))
		}
	},
}

func init() {
	rootCmd.AddCommand(buildCmd)

	buildCmd.Flags().StringP( "namespace", "n","default", "namespace of app")
	buildCmd.Flags().StringP( "url", "u","", "[Git] url of git repo")
	buildCmd.Flags().StringP( "revision", "r","master", "[Git] revision of git repo")
	buildCmd.Flags().StringP("template", "t", "kaniko", "[Git] template of build-to-image task")
}
