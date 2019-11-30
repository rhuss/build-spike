# Kn services deploy plugin

## Create build task first
Please create the build task first from https://github.com/tektoncd/catalog
### Such as buildpacks build task:
- Download yaml from https://github.com/tektoncd/catalog/blob/master/buildpacks/buildpacks-v3.yaml as buildpacks-v3.yaml
- `kubectl apply -f buildpacks-v3.yaml`
### Such as kaniko build task:
- Download yaml from https://github.com/tektoncd/catalog/blob/master/kaniko/kaniko.yaml
- `kubectl apply -f kaniko.yaml`

## Build and Deploy Knative service by buildpacks build task
```
  # Deploy knative service by building image by using buildpacks builder
  kn services deploy cnbtest \
    --builder buildpacks-v3 \
    --giturl https://github.com/zhangtbj/cf-sample-app-nodejs \
    --gitrevision master \
    --saved-image us.icr.io/test/cnbtest:v1.0 \
    --serviceaccount default \
    --namespace default --force
```
Command Result:
![Deploy Command](doc/deploy.png)

## Build and Deploy Knative service by kaniko build task
```
  # Deploy knative service by building image by using buildpacks builder
  kn services deploy kanikotest \
    --builder kaniko \
    --giturl https://github.com/bluebosh/knap-example \
    --saved-image us.icr.io/knative_jordan/kanikotest:latest \
    --serviceaccount default \
    --force
```

## Redeploy Knative service by special settings
```
  # Redeploy knative service by changing image tag
  kn services redeploy cnbtest \
    --saved-image us.icr.io/test/cnbtest:v2.0 \
    --namespace default
```
Command Result:
![Redeploy Command](doc/redeploy.png)

## Build image
```
  # Build from git repository into an image by using kaniko builder
  kn-service build kanikotest \
     --builder kaniko \
    --giturl https://github.com/bluebosh/knap-example \
    --gitrevision master \
    --saved-image us.icr.io/test/kaniko-image
```

## Work flow
![Work Flow](doc/flow.png)
