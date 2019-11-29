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
	"github.com/knative-community/build-spike/plugins/kn-services/tekton"
	"github.com/spf13/cobra"
	tektoncdclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc" // from https://github.com/kubernetes/client-go/issues/345
	"k8s.io/client-go/tools/clientcmd"
	"os"
)

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build Knative service image",
	Example: `
  # Build from Git repository into an image
  # ( related: https://github.com/knative-community/build-spike/blob/master/plugins/app/doc/deploy-git-resource.md )
  kn-service build example-image --giturl https://github.com/bluebosh/knap-example -gitrevision master --builder kaniko --saved-image us.icr.io/test/example-image --serviceaccount default`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("")
		if len(args) < 1 {
			cmd.Help()
			os.Exit(0)
		}
		name := args[0]

		builder := cmd.Flag("builder").Value.String()
		if builder == "" {
			fmt.Println("[ERROR] Builder cannot be empty")
			os.Exit(1)
		}
		gitUrl := cmd.Flag("giturl").Value.String()
		if gitUrl == "" {
			fmt.Println("[ERROR] Git url cannot be empty")
			os.Exit(1)
		}
		gitRevision := cmd.Flag("gitrevision").Value.String()
		if gitRevision == "" {
			fmt.Println("[ERROR] Git revision cannot be empty")
			os.Exit(1)
		}
		image := cmd.Flag("saved-image").Value.String()
		if image == "" {
			fmt.Println("[ERROR] Image cannot be empty")
			os.Exit(1)
		}
		serviceAccount := cmd.Flag("serviceaccount").Value.String()
		namespace := cmd.Flag("namespace").Value.String()

		// Config kubeconfig
		kubeconfig = rootCmd.Flag("kubeconfig").Value.String()
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

		if len(gitUrl) > 0 {
			err = tektonClient.BuildFromGit(name, builder, gitUrl, gitRevision, image, serviceAccount, namespace)
			if err != nil {
				fmt.Println("[ERROR] Building image error:", err)
				os.Exit(1)
			}
			fmt.Println("[INFO] Generate image", image, "from git repo")
		}
	},
}

func init() {
	rootCmd.AddCommand(buildCmd)
	buildCmd.Flags().StringP("builder", "b", "", "builder of source-to-image task")
	buildCmd.Flags().StringP( "giturl", "u","", "[Git] url of git repo")
	buildCmd.Flags().StringP( "gitrevision", "r","master", "[Git] revision of git repo")
	buildCmd.Flags().StringP("saved-image", "i", "", "generated saved image path")
	buildCmd.Flags().StringP("serviceaccount", "s", "default", "service account to push image")
	buildCmd.Flags().StringP( "namespace", "n","default", "namespace of build")
}
