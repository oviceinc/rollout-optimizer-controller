# rollout-optimizer-controller
This is a controller that helps blue green deployment of [Argo Rollout](https://argoproj.github.io/argo-rollouts/features/bluegreen/).


When you specify `scaleDownDelaySeconds` to 0 in [Argo Rollout](https://argoproj.github.io/argo-rollouts/features/bluegreen/#scaledowndelayseconds) resource, Argo Rollout will not scale down the old Replica Set. The aim of this controller is to manage it. This controller scales down the old Replica Set after argo-rollouts finished.


The problem of argo-rollouts is too early to terminate the old Replica Set pods. For example, if you execute WebSocket server on your Rollout, you want to scale dow slowly during deployment. But argo-rollouts terminate the old pods after switching traffics. Even though Service is switched by argo-rollouts, old WebSocket connections still connect to the old pods. So you want to terminate the old pods 1 by 1 to mitigate the load when reconnecting. This controller can terminate the olds pods slowly.


# Install
## Install argo-rollouts
This controller depends on argo-rollouts, so please install [argo-rollouts](https://github.com/argoproj/argo-rollouts).

## Install rollout-optimizer-controller
```
$ kubectl apply -k https://github.com/oviceinc/rollout-optimizer-controller//config/default/?ref=v0.1.0
```

# Usage
At first, please apply your Rollout, like

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: rollout-sample
spec:
  replicas: 2
  revisionHistoryLimit: 2
  selector:
    matchLabels:
      app: rollout-sample
  template:
    metadata:
      labels:
        app: rollout-sample
    spec:
      containers:
      - name: nginx
        image: nginx:1.25.0
        imagePullPolicy: Always
        ports:
        - containerPort: 80
  strategy:
    blueGreen:
      activeService: rollout-bluegreen-active
      previewService: rollout-bluegreen-preview
      autoPromotionEnabled: true
      # Do not scaleDown from ArgoRollout
      scaleDownDelaySeconds: 0

```

Then, apply RolloutScaleDown resource, like

```yaml
apiVersion: rollout.ovice.com/v1alpha1
kind: RolloutScaleDown
metadata:
  name: rolloutscaledown-sample
spec:
  targetRollout: "rollout-sample"
  terminatePerOnce: 1
  coolTimeSeconds: 300
```

Finally, when you update the Rollout resource, the old pods will be terminated 1 by 1 per 300 seconds.

# License

Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

