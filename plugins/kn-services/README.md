# Kn service build & deploy plugin

## Create build task first
Please create the build task first from https://github.com/tektoncd/catalog
Such as buildpacks:
- Download yaml from https://github.com/tektoncd/catalog/blob/master/buildpacks/buildpacks-v3.yaml as buildpacks-v3.yaml
- `kubectl apply -f buildpacks-v3.yaml`

## Build image
```
  # Build from Git repository into an image by using kaniko builder
  kn-service build example-image --giturl https://github.com/bluebosh/knap-example --gitrevision master --builder kaniko --image us.icr.io/test/example-image --serviceaccount default
```

## Deploy Knative service by building image
```
  # Deploy knative application by building image by using buildpacks builder
  kn-service deploy example-image --giturl https://github.com/zhangtbj/cf-sample-app-nodejs --gitrevision master --builder buildpacks-v3 --image us.icr.io/test/example-image --serviceaccount default
```

## Work flow
![Work Flow](doc/flow.png)
