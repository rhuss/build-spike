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
	servingclientset_v1alpha1 "github.com/knative/client/pkg/serving/v1alpha1"
	serving_v1alpha1_api "github.com/knative/serving/pkg/apis/serving/v1alpha1"
	serving_v1beta1_api "github.com/knative/serving/pkg/apis/serving/v1beta1"
	serviceclientset "github.com/knative/serving/pkg/client/clientset/versioned"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
	MaxTimeout = 300
)

var cfgFile string
var kubeconfig string

// redeployCmd represents the redeploy command
var redeployCmd = &cobra.Command{
	Use:   "redeploy",
	Short: "Redeploy Knative service by special settings",
	Example: `
  # Rebuild and redeploy from source code to Knative service
  # ( related: https://github.com/knative-community/build-spike/blob/master/plugins/deploy/README.md )
  > kn redeploy example-image
    --namespace default

  > kn redeploy test-image 
      --saved-image us.icr.io/test/test-image2 
      --namespace default

  > kn redeploy example-function-image 
      --file "function main() {return {payload: 'Hello Jordan!'};}"
      --saved-image us.icr.io/test/example-function-image2 
      --namespace default`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("")
		if len(args) < 1 {
			fmt.Println("[ERROR] Name cannot be empty, please set as the first parameter")
			cmd.Help()
			os.Exit(0)
		}
		name := args[0]
		gitResourceName := name + "-git"
		imageResourceName := name + "-image"

		fmt.Println("[INFO] Redeploy Knative service by special settings")
		namespace := cmd.Flag("namespace").Value.String()
		if namespace == "" {
			fmt.Println("[ERROR] Namespace cannot be empty, please use --namespace to set")
			os.Exit(1)
		}

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
			imageResource, err := tektonClient.GetPipelineResource(imageResourceName)
			if err != nil {
				fmt.Println("[ERROR] Get image output resource error:", err)
				os.Exit(1)
			}
			if len(imageResource.Labels) > 0 {
				builder = imageResource.Labels["builder"]
			}
		}
		if builder == "" {
			fmt.Println("[ERROR] Cannot get builder for redeploy, please use --builder to set")
			os.Exit(1)
		}

		serviceAccount := cmd.Flag("serviceaccount").Value.String()
		if serviceAccount == "" {
			imageResource, err := tektonClient.GetPipelineResource(imageResourceName)
			if err != nil {
				fmt.Println("[ERROR] Get image output resource error:", err)
				os.Exit(1)
			}
			if len(imageResource.Labels) > 0 {
				serviceAccount = imageResource.Labels["serviceaccount"]
			}
		}
		if serviceAccount == "" {
			fmt.Println("[ERROR] Cannot get serviceaccount for redeploy, please use --serviceaccount to set")
			os.Exit(1)
		}

		file := cmd.Flag("file").Value.String()
		if file == "" {
			imageResource, err := tektonClient.GetPipelineResource(imageResourceName)
			if err != nil {
				fmt.Println("[ERROR] Get image output resource error:", err)
				os.Exit(1)
			}
			if len(imageResource.Annotations) > 0 {
				file = imageResource.Annotations["file"]
			}
		}

		gitUrl := ""
		gitRevision := ""
		if len(file) > 0 {
			fmt.Println("[INFO] Get function code from file")
			fmt.Println("[INFO] Get function code:", file)
		} else {
			gitUrl = cmd.Flag("git-url").Value.String()
			if gitUrl == "" {
				gitResource, err := tektonClient.GetPipelineResource(gitResourceName)
				if err != nil {
					fmt.Println("[ERROR] Get Git output resource error:", err)
					os.Exit(1)
				}
				if len(gitResource.Spec.Params) > 0 {
					gitUrl = gitResource.Spec.Params[0].Value
				}
			}

			gitRevision = cmd.Flag("git-revision").Value.String()
			if gitRevision == "" {
				gitResource, err := tektonClient.GetPipelineResource(gitResourceName)
				if err != nil {
					fmt.Println("[ERROR] Get Git output resource error:", err)
					os.Exit(1)
				}
				if len(gitResource.Spec.Params) > 0 {
					gitRevision = gitResource.Spec.Params[1].Value
				}
			}
			fmt.Println("[INFO] Get source code from git resources")
			fmt.Println("[INFO] Git url:", gitUrl, "git revision:", gitRevision)
		}

		image := cmd.Flag("saved-image").Value.String()
		if image == "" {
			imageResource, err := tektonClient.GetPipelineResource(imageResourceName)
			if err != nil {
				fmt.Println("[ERROR] Get image output resource error:", err)
				os.Exit(1)
			}
			if len(imageResource.Spec.Params) > 0 {
				image = imageResource.Spec.Params[0].Value
			}
		}
		if image == "" {
			fmt.Println("[ERROR] Cannot get saved-image for redeploy, please use --saved-image to set")
			os.Exit(1)
		}

		if len(gitUrl) > 0 && len(file) == 0 {
			fmt.Println("[INFO] Re-build from git repository into an image")
			err = tektonClient.BuildFromGit(name, builder, gitUrl, gitRevision, image, serviceAccount, namespace)
			if err != nil {
				fmt.Println("[ERROR] Re-building image from git error:", err)
				os.Exit(1)
			}
			fmt.Println("[INFO] Re-create image", image, "from git repo")
		} else if len(file) > 0 && len(gitUrl) == 0 {
			fmt.Println("[INFO] Re-build from function file into an image")
			err = tektonClient.BuildFromFunctionFile(name, builder, file, image, serviceAccount, namespace)
			if err != nil {
				fmt.Println("[ERROR] Re-building image from function file error:", err)
				os.Exit(1)
			}
			fmt.Println("[INFO] Re-create image", image, "from function file")
		} else {
			fmt.Println("[ERROR] Do not support set --git-url and --file parameters together")
			os.Exit(1)
		}

		// Deploy knative service
		knclient, err := serviceclientset.NewForConfig(cfg)
		if err != nil {
			fmt.Println("[ERROR] Serving kubeconfig:", err)
			os.Exit(1)
		}

		fmt.Println("\n[INFO] Re-deploy the Knative service by using the new generated image")
		servingClient := servingclientset_v1alpha1.NewKnServingClient(knclient.ServingV1alpha1(), namespace)
		serviceExists, service, err := serviceExists(servingClient, name)
		if err != nil {
			fmt.Println("[ERROR] Checking service exist:", err)
			os.Exit(1)
		}
		action := "created"
		if serviceExists {
			err = replaceService(servingClient, service, image, serviceAccount)
			action = "replaced"
		} else {
			if service == nil {
				service, err = constructService(name, image, serviceAccount, namespace)
			}
			if err != nil {
				fmt.Println("[ERROR] Constructing service:", err)
				os.Exit(1)
			}
			err = createService(servingClient, service)
		}
		if err != nil {
			fmt.Println("[ERROR] Create service:", err)
			os.Exit(1)
		} else {
			fmt.Println("[INFO] Service", service.Name , "successfully", action ,"in namespace", namespace)
		}

		time.Sleep(5 * time.Second)
		i := 0
		for  i < MaxTimeout {
			service, err = servingClient.GetService(name)
			if service.Status.LatestReadyRevisionName != "" {
				fmt.Println("[INFO] service", name,"is ready")
				url := service.Status.URL.String()
				if url == "" {
					url = service.Status.DeprecatedDomain
				}
				fmt.Println("[INFO] Service", name,"url is", url)
				return
			} else {
				fmt.Println("[INFO] Service", name,"is still creating, waiting")
				time.Sleep(5 * time.Second)
			}
			if i == MaxTimeout {
				fmt.Println("[ERROR] Fail to create service", name, "after timeout")
				os.Exit(1)
			}
			i += 5
			time.Sleep(5 * time.Second)
		}
	},
}

func Execute() {
	if err := redeployCmd.Execute(); err != nil {
	  fmt.Println(err)
	  os.Exit(1)
	}
  }

func init() {
	cobra.OnInitialize(initConfig)
	redeployCmd.PersistentFlags().StringP("kubeconfig", "", "", "kube config file (default is KUBECONFIG from ENV property)")
	redeployCmd.Flags().StringP("builder", "b", "", "builder of source-to-image task")
	redeployCmd.Flags().StringP( "git-url", "u","", "[Git] url of git repo")
	redeployCmd.Flags().StringP( "file", "f","", "[Function] File of function which snippet of code without http server")
	redeployCmd.Flags().StringP( "git-revision", "r","master", "[Git] revision of git repo")
	redeployCmd.Flags().StringP("saved-image", "i", "", "generated saved image path")
	redeployCmd.Flags().StringP("serviceaccount", "s", "", "service account to push image")
	redeployCmd.Flags().StringP( "namespace", "n","", "namespace of build")
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

// Create a new Knative service
func createService(client servingclientset_v1alpha1.KnClient, service *serving_v1alpha1_api.Service) error {
	err := client.CreateService(service)
	if err != nil {
		return err
	}
	return nil
}

// Replace the existing Knative service
func replaceService(client servingclientset_v1alpha1.KnClient, service *serving_v1alpha1_api.Service, image, serviceAccount string) error {
	var retries = 0
	for {
		service.Spec = serving_v1alpha1_api.ServiceSpec{
			ConfigurationSpec:    serving_v1alpha1_api.ConfigurationSpec{
				Template: &serving_v1alpha1_api.RevisionTemplateSpec{
					Spec: serving_v1alpha1_api.RevisionSpec{
						RevisionSpec: serving_v1beta1_api.RevisionSpec{
							PodSpec: serving_v1beta1_api.PodSpec{
								ServiceAccountName: serviceAccount,
								Containers: []corev1.Container{
									{
										Image: image,
									},
								},
							},
						},
					},
				},
			},
		}
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
func serviceExists(client servingclientset_v1alpha1.KnClient, name string) (bool, *serving_v1alpha1_api.Service, error) {
	service, err := client.GetService(name)
	if api_errors.IsNotFound(err) {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, err
	}
	return true, service, nil
}

// Create service struct from provided options
func constructService(name, image, serviceAccount, namespace string) (*serving_v1alpha1_api.Service,
	error) {

	service := serving_v1alpha1_api.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: serving_v1alpha1_api.ServiceSpec{
			ConfigurationSpec:    serving_v1alpha1_api.ConfigurationSpec{
				Template: &serving_v1alpha1_api.RevisionTemplateSpec{
					Spec: serving_v1alpha1_api.RevisionSpec{
						RevisionSpec: serving_v1beta1_api.RevisionSpec{
							PodSpec: serving_v1beta1_api.PodSpec{
								ServiceAccountName: serviceAccount,
								Containers: []corev1.Container{
									{
										Image: image,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	return &service, nil
}
