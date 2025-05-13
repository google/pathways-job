# pathways-job
PathwaysJob API is an OSS Kubernetes-native API, to deploy ML training and batch inference workloads, using Pathways on GKE. 

## Introduction to Pathways on Cloud
Pathways is a system designed to enable the creation of large-scale, multi-task, and sparsely activated machine learning systems. It allows for the use of thousands or tens of thousands of accelerators, with the ability to dynamically allocate varying amounts of compute for different tasks based on their processing requirements. Pathways simplifies large-scale machine learning computations by enabling a single JAX client to orchestrate workloads across multiple large TPU slices, potentially spanning hundreds or thousands of TPU chips. Pathways is used internally at Google to train large models like Gemini. Pathways on Cloud brings the same benefits to Google Cloud customers.

## Description of PathwaysJob
PathwaysJob is an API that provides an easy way to run JAX workloads using Pathways. A PathwaysJob instance bundles the Pathways resource manager(RM) AKA Pathways server, the Pathways proxy server and the user workload containers into a single pod named `pathways-head`. When a user pod is not provided (headless workloads), the `pathways-head` pod consists of the Pathways RM and the proxy server containers.
It supports the following two deployment modes: ColocateHeadWithWorkers and Default.
### ColocateHeadWithWorkers mode
The `colocate_head_with_workers` mode deploys the `pathways-head` pod besides a `worker` pod on one of the TPU workers. This is preferred for Pathways batch inference workloads, where latency is crucial.
### Default mode
The `pathways-head` pod is schedule on a CPU nodepool and the `workers` are scheduled on TPUs. The default mode is preferred for Pathways training workloads where the worker utilizes the TPUs completely.
### With a dockerized workload
The user workload is scheduled as a container within the `pathways-head` pod.
### Headless mode for interactive supercomputing
The user workload is typically on a Vertex AI notebook, so users can connect to the PathwaysJob via port-forwarding.

## Getting Started

### Prerequisites
- go version v1.23.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.
- JobSet version v0.8.0+ installed on the cluster following https://jobset.sigs.k8s.io/docs/installation/#install-a-released-version


### Install a released version
To install the latest released version of PathwaysJob version on your cluster, run the following command:
```sh
VERSION=v0.1.1
kubectl apply --server-side -f https://github.com/google/pathways-job/releases/download/$VERSION/install.yaml
```

To uninstall the latest released version of PathwaysJob version on your cluster, run the following command:
```sh
VERSION=v0.1.1
kubectl delete -f https://github.com/google/pathways-job/releases/download/$VERSION/install.yaml
```

### Build and install from source
To build PathwaysJob from source and install it on your cluster, run the following commands:
**Build and push your image to the location specified by `IMAGE`:**

```sh
git clone https://github.com/google/pathways-job.git
cd pathways-job
IMAGE=<$(IMAGE_REGISTRY)/pathwaysjob-controller:$(IMAGE_TAG)>
make docker-build docker-push IMG=$IMAGE
```

**NOTE:** Please update IMAGE_REGISTRY and IMAGE_TAG in the commands mentioned above.
This image ought to be published in the personal registry you specified.
And it is required to have access to pull the image from the working environment.
Make sure you have the proper permission to the registry if the above commands don’t work.

**Install the CRDs into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMAGE`:**

```sh
make deploy IMG=<some-registry>/pathways-job:tag
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin
privileges or be logged in as admin.

**Create instances of your solution**
You can apply the examples from the config/samples:

```sh
kubectl apply -k config/samples/<example name>.yaml
```

>**NOTE**: Refer to the examples showcasing PathwaysJob features.
>Ensure that the examples has default values to test it out.

**Delete the instances (CRs) from the cluster:**
You can delete the examples from the config/samples, applied above:
```sh
kubectl delete -k config/samples/<example name>.yaml
```

**Undeploy the controller from the cluster:**

```sh
make undeploy
```


## Contributing
We welcome contributions! Please look at [contributing.md](/usr/local/google/home/roshanin/pathways-job/docs/contributing.md).
More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
