# machine-controller-manager-provider-aws
This project contains the external/out-of-tree plugin (driver) implementation of machine-controller-manager for AWS. This implementation adheres to the machine-interface defined at following project: 
- https://github.com/gardener/machine-spec

## Fundamental Design Principles:
Following are the basic principles kept in mind while developing the external plugin.
* Communication between external plugin and cmi-client (machine-controller) is achieved using gRPC mechanism.
* External plugin behaves as gRPC-server and cmi-client behaves as gRPC client.
* Cloud-provider specific contract should be scoped under `ProviderSpec` field. `ProviderSpec` field is expected to be raw-bytes at machine-controller-side. External plugin should have pre-defined typed-apis to parse the `ProviderSpec` to make necessary CP specific calls.
* External plugins do not need to communicate with kubernetes api-server.
    * Kubeconfig may not be available to external-plugins.

## Sequence Diagram:

![Sequence Diagram](images/seqdiagram.png)