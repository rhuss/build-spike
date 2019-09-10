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
	servingclientset_v1alpha1 "github.com/knative/client/pkg/serving/v1alpha1"
	serving_v1alpha1_api "github.com/knative/serving/pkg/apis/serving/v1alpha1"
	serviceclientset "github.com/knative/serving/pkg/client/clientset/versioned"
	"github.com/spf13/cobra"
	tektoncdclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc" // from https://github.com/kubernetes/client-go/issues/345
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"time"
)

const (
	// How often to retry in case of an optimistic lock error when replacing a service (--force)
	MaxUpdateRetries = 3

	// Timeout to wait service creation
	MAX_TIMEOUT = 60
)

// deployCmd represents the deploy command
var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy knative application by building image",
	Example: `
  # Deploy from Git repository into an image
  # ( related: https://github.com/knative-community/build-spike/blob/master/plugins/app/doc/deploy-git-resource.md )
  app go deploy test -u https://github.com/bluebosh/knap-example -r master`,
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
		//path := cmd.Flag("path").Value.String()

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


		// Deploy knative service
		knclient, err := serviceclientset.NewForConfig(cfg)
		if err != nil {
			fmt.Errorf("\nError serving kubeconfig: %v", err)
		}

		client := servingclientset_v1alpha1.NewKnServingClient(knclient.ServingV1alpha1(), namespace)

		service, err := constructService(name, namespace)
		if err != nil {
			fmt.Errorf("\nError constructing service: %v", err)
		}

		serviceExists, err := serviceExists(client, name)
		if err != nil {
			fmt.Errorf("\nError checking service exit: %v", err)
		}

		action := "created"
		if serviceExists {
			err = replaceService(client, service)
			action = "replaced"
		} else {
			err = createService(client, service)
		}
		if err != nil {
			fmt.Errorf("\nError create service: %v", err)
		} else {
			fmt.Println("[INFO] Service", color.BlueString(service.Name) , "successfully", action ,"in namespace", color.CyanString(namespace))
		}

		time.Sleep(5 * time.Second)
		for i := 0; i <= MAX_TIMEOUT; i++ {
			service, err = client.GetService(name)
			if service.Status.LatestReadyRevisionName != "" {
				fmt.Println("[INFO] service", color.BlueString(name),"is ready")
				url := service.Status.URL.String()
				if url == "" {
					url = service.Status.DeprecatedDomain
				}
				fmt.Println("[INFO] Service", color.CyanString(name),"url is", color.CyanString(url))
				return
			} else {
				fmt.Println("[INFO] service", color.BlueString(name),"is still creating, waiting")
				time.Sleep(5 * time.Second)
			}
			if i == MAX_TIMEOUT {
				fmt.Println("[ERROR] Fail to create service", color.CyanString(name), "after timeout")
				os.Exit(1)
			}
			time.Sleep(5 * time.Second)
		}
	},
}

func init() {
	rootCmd.AddCommand(deployCmd)

	deployCmd.Flags().StringP( "namespace", "n","default", "namespace of app")
	deployCmd.Flags().StringP( "url", "u","", "[Git] url of git repo")
	deployCmd.Flags().StringP( "revision", "r","master", "[Git] revision of git repo")
	deployCmd.Flags().StringP("template", "t", "kaniko", "[Git] template of build-to-image task")
}

// Create a new Knative service
func createService(client servingclientset_v1alpha1.KnClient, service *serving_v1alpha1_api.Service) error {
	err := client.CreateService(service)
	if err != nil {
		return err
	}
	return nil
}

// Replace the existing Knative service
func replaceService(client servingclientset_v1alpha1.KnClient, service *serving_v1alpha1_api.Service) error {
	var retries = 0
	for {
		service.ResourceVersion = ""
		err := client.UpdateService(service)
		if err != nil {
			// Retry to update when a resource version conflict exists
			if api_errors.IsConflict(err) && retries < MaxUpdateRetries {
				retries++
				continue
			}
			return err
		}
		return nil
	}
}

// Check if the service exists
func serviceExists(client servingclientset_v1alpha1.KnClient, name string) (bool, error) {
	_, err := client.GetService(name)
	if api_errors.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// Create service struct from provided options
func constructService(name string, namespace string) (*serving_v1alpha1_api.Service,
	error) {

	service := serving_v1alpha1_api.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	// TODO: Should it always be `runLatest` ?
	service.Spec.DeprecatedRunLatest = &serving_v1alpha1_api.RunLatestType{
		Configuration: serving_v1alpha1_api.ConfigurationSpec{
			DeprecatedRevisionTemplate: &serving_v1alpha1_api.RevisionTemplateSpec{
				Spec: serving_v1alpha1_api.RevisionSpec{
					DeprecatedContainer: &corev1.Container{},
				},
			},
		},
	}

	return &service, nil
}
