package tekton

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	"knative.dev/pkg/apis"
	"github.com/knative-community/build-spike/plugins/app/tekton/templates"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tekton_v1alpha1_api "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tekton_v1alpha1_client "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/typed/pipeline/v1alpha1"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strconv"
	"time"
)

const BUILD_TASK_NAME  = "build-to-image"

type TektonClient interface {
	// Create PipelineResource struct from provided options
	ConstructPipelineResource(name string, url string, revision string, namespace string) *tekton_v1alpha1_api.PipelineResource

	// Get a PipelineResource by its unique name
	GetPipelineResource(name string) (*v1alpha1.PipelineResource, error)

	// Check if the PipelineResource exists
	PipelineResourceExists(name string) (bool, error)

	// Create a new pipelineresource
	CreatePipelineResource(pipelineresource *v1alpha1.PipelineResource) error

	// Update the given pipelineresource
	UpdatePipelineResource(pipelineresource *v1alpha1.PipelineResource) error

	// Create task struct from provided options
	ConstructTask(name string, template string, namespace string) (*tekton_v1alpha1_api.Task, error)

	// Get a task by its unique name
	GetTask(name string) (*v1alpha1.Task, error)

	// Check if the task exists
	TaskExists(name string) (bool, error)

	// Create a new task
	CreateTask(task *v1alpha1.Task) error

	// Update the given task
	UpdateTask(task *v1alpha1.Task) error

	// Create taskrun struct from provided options
	ConstructTaskRun(name string, run_template string, taskref string, namespace string) (*tekton_v1alpha1_api.TaskRun, error)

	// Get a taskrun by its unique name
	GetTaskRun(name string) (*v1alpha1.TaskRun, error)

	// Create a new taskrun
	StartTaskRun(task *v1alpha1.TaskRun) (string, error)

	// Build application from git
	BuildFromGit(name string, namespace string, url string, revision string) (string, error)
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

// Create PipelineResource struct from provided options
func (cl *tektonClient) ConstructPipelineResource(name string, url string, revision string, namespace string) *tekton_v1alpha1_api.PipelineResource {

	pipelineresource := tekton_v1alpha1_api.PipelineResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: tekton_v1alpha1_api.PipelineResourceSpec{
			Type: tekton_v1alpha1_api.PipelineResourceTypeGit,
			Params: []tekton_v1alpha1_api.ResourceParam{{
				Name:  "url",
				Value: url,
			}, {
				Name:  "revision",
				Value: revision,
			}},
		},
	}

	return &pipelineresource
}

func (cl *tektonClient) PipelineResourceExists(name string) (bool, error) {
	_, err := cl.client.PipelineResources(cl.namespace).Get(name, metav1.GetOptions{})
	if api_errors.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// Get a pipelineresource by its unique name
func (cl *tektonClient) GetPipelineResource(name string) (*v1alpha1.PipelineResource, error) {
	pipelineresource, err := cl.client.PipelineResources(cl.namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return pipelineresource, nil
}

// Create a new pipelineresource
func (cl *tektonClient) CreatePipelineResource(pipelineresource *v1alpha1.PipelineResource) error {
	_, err := cl.client.PipelineResources(cl.namespace).Create(pipelineresource)
	if err != nil {
		return err
	}
	return nil
}

// Update the given pipelineresource
func (cl *tektonClient) UpdatePipelineResource(pipelineresource *v1alpha1.PipelineResource) error {
	var retries = 0
	var MaxUpdateRetries = 3
	for {
		existingPipelineResource, err := cl.client.PipelineResources(cl.namespace).Get(pipelineresource.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		updateResource := pipelineresource.DeepCopy()
		updateResource.ResourceVersion = existingPipelineResource.ResourceVersion

		_, err = cl.client.PipelineResources(cl.namespace).Update(updateResource)
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

// Create Task struct from provided options
func (cl *tektonClient) ConstructTask(name string, template string, namespace string) (*tekton_v1alpha1_api.Task,
	error) {
	bytes := []byte("")
	if template == "kaniko" {
		bytes = []byte(templates.KANIKO_TEMPLATE)
	}

	var spec tekton_v1alpha1_api.Task
	err := yaml.Unmarshal(bytes, &spec)
	if err != nil {
		panic(err.Error())
	}

	task := tekton_v1alpha1_api.Task{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: spec.Spec,
	}

	return &task, nil
}

func (cl *tektonClient) TaskExists(name string) (bool, error) {
	_, err := cl.client.Tasks(cl.namespace).Get(name, metav1.GetOptions{})
	if api_errors.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// Get a task by its unique name
func (cl *tektonClient) GetTask(name string) (*v1alpha1.Task, error) {
	task, err := cl.client.Tasks(cl.namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return task, nil
}

// Create a new task
func (cl *tektonClient) CreateTask(task *v1alpha1.Task) error {
	_, err := cl.client.Tasks(cl.namespace).Create(task)
	if err != nil {
		return err
	}
	return nil
}

// Update the given task
func (cl *tektonClient) UpdateTask(task *v1alpha1.Task) error {
	var retries = 0
	var MaxUpdateRetries = 3
	for {
		existingTask, err := cl.client.Tasks(cl.namespace).Get(task.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		updateTask := task.DeepCopy()
		updateTask.ResourceVersion = existingTask.ResourceVersion

		_, err = cl.client.Tasks(cl.namespace).Update(updateTask)
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

// Create Task struct from provided options
func (cl *tektonClient) ConstructTaskRun(name string, run_template string, taskref string, namespace string) (*tekton_v1alpha1_api.TaskRun,
	error) {
	bytes := []byte("")
	if run_template == "kaniko" {
		bytes = []byte(templates.KANIKO_RUN_TEMPLATE)
	}

	var spec tekton_v1alpha1_api.TaskRun
	err := yaml.Unmarshal(bytes, &spec)
	if err != nil {
		panic(err.Error())
	}

	taskrun := tekton_v1alpha1_api.TaskRun{}
	resource, err := cl.GetPipelineResource(name + "-git")
	if err != nil {
		panic(err.Error())
	}

	time := time.Now().Format("20060102150405")
	taskrun.Spec = spec.Spec
	taskrun.Name = name + "-tr-" + strconv.FormatInt(resource.Generation, 10) + "-" + taskref + "-" + time
	taskrun.Namespace = namespace
	taskrun.Spec.Inputs.Params[1].Value.StringVal = "us.icr.io/knative_jordan/" + name
	taskrun.Spec.Inputs.Params[2].Value.StringVal = strconv.FormatInt(resource.Generation, 10) + ".0"
	taskrun.Spec.Inputs.Resources[0].ResourceRef.Name = resource.Name
	taskrun.Spec.TaskRef.Name = taskref

	return &taskrun, nil
}

// Get a taskrun by its unique name
func (cl *tektonClient) GetTaskRun(name string) (*v1alpha1.TaskRun, error) {
	taskrun, err := cl.client.TaskRuns(cl.namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return taskrun, nil
}

// Start a new taskrun
func (cl *tektonClient) StartTaskRun(taskrun *v1alpha1.TaskRun) (string, error) {
	MAX_TIMEOUT := 60
	_, err := cl.client.TaskRuns(cl.namespace).Create(taskrun)
	if err != nil {
		return taskrun.Name, err
	}

	time.Sleep(5 * time.Second)
	for i := 0; i < MAX_TIMEOUT; i++ {
		taskrun, err = cl.client.TaskRuns(cl.namespace).Get(taskrun.Name, metav1.GetOptions{})
		if taskrun.Status.Conditions[0].Type == apis.ConditionSucceeded && taskrun.Status.Conditions[0].Status == "True" {
			fmt.Println("[INFO] Build taskrun", color.CyanString(taskrun.Name),"is ready from", taskrun.Status.StartTime, "to", taskrun.Status.CompletionTime)
			return taskrun.Name, nil
		} else {
			fmt.Println("[INFO] Build taskrun", color.CyanString(taskrun.Name),"is still", taskrun.Status.Conditions[0].Reason,", waiting")
			time.Sleep(5 * time.Second)
		}
		time.Sleep(5 * time.Second)
	}

	return taskrun.Name, fmt.Errorf("[ERROR] The build taskrun is not ready after timeout")
}


// Build application from git
func (cl *tektonClient) BuildFromGit(name string, namespace string, url string, revision string) (string, error) {

	fmt.Println("[INFO] Building application", color.BlueString(name), "image in namespace", color.CyanString(namespace))
	fmt.Println("[INFO] From git repo", color.CyanString(url), "revision", color.CyanString(revision), "\n")
	// Create pipelineresource
	pipelineresource := cl.ConstructPipelineResource(name + "-git", url, revision, namespace)
	pipelineresourceExists, err := cl.PipelineResourceExists(name+ "-git")
	if err != nil {
		glog.Fatalf("\nCheck resource exist error: %s", err)
	}
	if pipelineresourceExists {
		fmt.Println("[INFO] Git resource", color.CyanString(pipelineresource.Name) ,"exists, updating")
		err = cl.UpdatePipelineResource(pipelineresource)
	} else {
		fmt.Println("[INFO] Git resource", color.CyanString(pipelineresource.Name) ,"doesn't exist, creating")
		err = cl.CreatePipelineResource(pipelineresource)
	}
	if err != nil {
		glog.Fatalf("\nCreate resource error: %s", err)
	}
	fmt.Println("[INFO] Git resource", color.CyanString(pipelineresource.Name),"created successfully\n")

	// Create build-to-image task
	buildtask, err := cl.ConstructTask(BUILD_TASK_NAME, "kaniko", namespace)
	if err != nil {
		glog.Fatalf("\nGenerate new task error: %s", err)
	}

	taskExists, err := cl.TaskExists(BUILD_TASK_NAME)
	if err != nil {
		glog.Fatalf("\nCheck task exist error: %s", err)
	}

	if taskExists {
		fmt.Println("[INFO] Build task", color.CyanString(buildtask.Name) ,"exists, updating")
		err = cl.UpdateTask(buildtask)
	} else {
		fmt.Println("[INFO] Build task", color.CyanString(buildtask.Name) ,"doesn't exist, creating")
		err = cl.CreateTask(buildtask)
	}
	if err != nil {
		glog.Fatalf("\nCreate task error: %s", err)
	}
	fmt.Println("[INFO] Build task", color.CyanString(pipelineresource.Name),"created successfully\n")

	// Start build-to-image taskrun
	buildtaskrun, err := cl.ConstructTaskRun(name, "kaniko", BUILD_TASK_NAME, namespace)
	fmt.Println("[INFO] Start taskrun", color.CyanString(buildtaskrun.Name), "for application", color.BlueString(name))
	taskrunname, err := cl.StartTaskRun(buildtaskrun)
	if err != nil {
		glog.Fatalf("\nStart taskrun error: %s", err)
	}
	fmt.Println("[INFO] Complete taskrun", color.CyanString(taskrunname),"successfully\n")
	fmt.Println("[INFO] Complete building application", color.BlueString(name), "image in namespace", color.CyanString(namespace))

	resource, err := cl.GetPipelineResource(name + "-git")
	if err != nil {
		glog.Fatalf("\nGet resource error: %s", err)
	}

	imagename := "us.icr.io/knative_jordan/" + name + ":" + strconv.FormatInt(resource.Generation, 10) + ".0"
	return imagename, nil
}
