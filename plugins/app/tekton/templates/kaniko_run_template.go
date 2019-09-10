package templates

const (
	KANIKO_RUN_TEMPLATE = `
apiVersion: tekton.dev/v1alpha1
kind: TaskRun
spec:
  inputs:
    params:
    - name: pathToContext
      value: src
    - name: imageUrl
      value: us.icr.io/knative_jordan/picalc
    - name: imageTag
      value: "1.0"
    resources:
    - name: git-source
      resourceRef:
        apiVersion: tekton.dev/v1alpha1
        name: helloworld-git
  outputs: {}
  serviceAccount: pipeline-account
  taskRef:
    kind: Task
    name: source-to-image
`
)
