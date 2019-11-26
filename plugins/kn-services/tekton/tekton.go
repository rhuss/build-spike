package tekton

import (
	"fmt"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	pipelinev1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tekton_v1alpha1_client "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/typed/pipeline/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	"time"
)

const (
	BuildpacksBuilderName  = "buildpacks-v3"
	BuildpacksBuilderImage = "cloudfoundry/cnb:bionic"
	KanikoBuilderName      = "kaniko"
	KanikoBuilderImage     = "gcr.io/kaniko-project/executor:v0.13.0"
	BuildTimeout           = 300
)

type TektonClient interface {
	// Check if the builder task exists
	TaskExists(name string) error

	// Create task run struct from provided options
	ConstructTaskRun(name, builder, gitUrl, gitRevision, image, serviceAccount, namespace string) *pipelinev1alpha1.TaskRun

	// Get a task run by its unique name
	GetTaskRun(name string) (*v1alpha1.TaskRun, error)

	// Create a new task run
	StartTaskRun(task *v1alpha1.TaskRun) (string, error)

	// Build image from git
	BuildFromGit(name, builder, gitUrl, gitRevision, image, serviceAccount, namespace string) error
}

type tektonClient struct {
	client    tekton_v1alpha1_client.TektonV1alpha1Interface
	namespace string
}

// Create a new client facade for the provided namespace
func NewTektonClient(client tekton_v1alpha1_client.TektonV1alpha1Interface, namespace string) TektonClient {
	return &tektonClient{
		client:    client,
		namespace: namespace,
	}
}

func (cl *tektonClient) TaskExists(name string) error {
	_, err := cl.client.Tasks(cl.namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	return nil
}

// ArrayOrString creates an ArrayOrString of type ParamTypeString or ParamTypeArray, based on
// how many inputs are given (>1 input will create an array, not string).
func ArrayOrString(value string, additionalValues ...string) *pipelinev1alpha1.ArrayOrString {
	if len(additionalValues) > 0 {
		additionalValues = append([]string{value}, additionalValues...)
		return &pipelinev1alpha1.ArrayOrString{
			Type:     pipelinev1alpha1.ParamTypeArray,
			ArrayVal: additionalValues,
		}
	}
	return &pipelinev1alpha1.ArrayOrString{
		Type:      pipelinev1alpha1.ParamTypeString,
		StringVal: value,
	}
}

// Create Task struct from provided options
func (cl *tektonClient) ConstructTaskRun(name, builder, gitUrl, gitRevision, image, serviceAccount, namespace string) *pipelinev1alpha1.TaskRun {

	builderImage := ""
	if builder == BuildpacksBuilderName {
		builderImage = BuildpacksBuilderImage
	} else if builder == KanikoBuilderName {
		builderImage = KanikoBuilderImage
	}

	return &pipelinev1alpha1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    namespace,
			GenerateName: name + "-",
		},
		Spec: pipelinev1alpha1.TaskRunSpec{
			TaskRef: &pipelinev1alpha1.TaskRef{
				Name: builder,
			},
			Inputs: pipelinev1alpha1.TaskRunInputs{
				Resources: []pipelinev1alpha1.TaskResourceBinding{{
					ResourceSpec: &pipelinev1alpha1.PipelineResourceSpec{
						// TODO also need prepare for other resource
						Type: pipelinev1alpha1.PipelineResourceTypeGit,
						Params: []pipelinev1alpha1.ResourceParam{
							{
								Name:  "url",
								Value: gitUrl,
							},
							{
								Name:  "revision",
								Value: gitRevision,
							},
						},
					},
					Name: "source",
				}},
				Params: []pipelinev1alpha1.Param{
					{
						Name:  "BUILDER_IMAGE",
						Value: *ArrayOrString(builderImage),
					},
				},
			},
			Outputs: pipelinev1alpha1.TaskRunOutputs{
				Resources: []pipelinev1alpha1.TaskResourceBinding{{
					ResourceSpec: &pipelinev1alpha1.PipelineResourceSpec{
						Type: pipelinev1alpha1.PipelineResourceTypeImage,
						Params: []pipelinev1alpha1.ResourceParam{{
							Name:  "url",
							Value: image,
						}},
					},
					Name: "image",
				}},
			},
			ServiceAccount: serviceAccount,
		},
	}
	return nil
}

// Get a tas krun by its unique name
func (cl *tektonClient) GetTaskRun(name string) (*v1alpha1.TaskRun, error) {
	taskRun, err := cl.client.TaskRuns(cl.namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return taskRun, nil
}

// Start a new task run
func (cl *tektonClient) StartTaskRun(taskrun *v1alpha1.TaskRun) (string, error) {
	newTaskRun, err := cl.client.TaskRuns(cl.namespace).Create(taskrun)
	if err != nil {
		return "", err
	}

	time.Sleep(5 * time.Second)
	i := 0
	for  i < BuildTimeout {
		taskrun, err = cl.client.TaskRuns(cl.namespace).Get(newTaskRun.Name, metav1.GetOptions{})
		if taskrun.Status.Conditions[0].Type == apis.ConditionSucceeded && taskrun.Status.Conditions[0].Status == "True" {
			fmt.Println("[INFO] Build task run", taskrun.Name, "is ready from", taskrun.Status.StartTime, "to", taskrun.Status.CompletionTime)
			return taskrun.Name, nil
		} else {
			fmt.Println("[INFO] Build task run", taskrun.Name, "is still", taskrun.Status.Conditions[0].Reason, ", waiting")
			time.Sleep(5 * time.Second)
		}
		i += 5
		time.Sleep(5 * time.Second)
	}

	return taskrun.Name, fmt.Errorf("the build taskrun is not ready after timeout")
}


// Build application from git
func (cl *tektonClient) BuildFromGit(name, builder, gitUrl, gitRevision, image, serviceAccount, namespace string) error {

	fmt.Println("[INFO] Building image", name, "in namespace", namespace)
	fmt.Println("[INFO] From git repo", gitUrl, "revision", gitRevision)

	err := cl.TaskExists(builder)
	if err != nil {
		return err
	}
	fmt.Println("[INFO] Get task", builder, "successfully")

	// Start source-to-image task run
	buildTaskRun := cl.ConstructTaskRun(name, builder, gitUrl, gitRevision, image, serviceAccount, namespace)
	fmt.Println("[INFO] Start task run for image", name)
	taskRunName, err := cl.StartTaskRun(buildTaskRun)
	if err != nil {
		return err
	}
	fmt.Println("[INFO] Complete task run", taskRunName,"successfully")
	fmt.Println("[INFO] Complete building application", name, "image in namespace", namespace)

	return nil
}
