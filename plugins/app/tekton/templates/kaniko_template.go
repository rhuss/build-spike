package templates

const (
	KANIKO_TEMPLATE = `
apiVersion: tekton.dev/v1alpha1
kind: Task
metadata:
  name: source-to-image
spec:
  inputs:
    params:
    - default: .
      description: The path to the build context, used by Kaniko - within the workspace
      name: pathToContext
    - default: Dockerfile
      description: The path to the dockerfile to build (relative to the context)
      name: pathToDockerFile
    - description: Url of image repository
      name: imageUrl
    - default: latest
      description: Tag to apply to the built image
      name: imageTag
    resources:
    - name: git-source
      outputImageDir: ""
      targetPath: ""
      type: git
  steps:
  - args:
    - --dockerfile=${inputs.params.pathToDockerFile}
    - --destination=${inputs.params.imageUrl}:${inputs.params.imageTag}
    - --context=/workspace/git-source/${inputs.params.pathToContext}
    command:
    - /kaniko/executor
    image: gcr.io/kaniko-project/executor
    name: build-and-push
    resources: {}
`
)
