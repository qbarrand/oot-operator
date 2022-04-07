# Kernel Module Management Operator

The Kernel Module Management Operator manages the deployment of out-of-tree kernel modules and
associated device plug-ins in Kubernetes.
Along with deployment it also manages the lifecycle of the kernel modules for new incoming kernel
versions attached to upgrades.

[![codecov](https://codecov.io/gh/rh-ecosystem-edge/kernel-module-management/branch/main/graph/badge.svg?token=OMIRXMN03W)](https://codecov.io/gh/rh-ecosystem-edge/kernel-module-management)
[![Go Reference](https://pkg.go.dev/badge/github.com/rh-ecosystem-edge/kernel-module-management.svg)](https://pkg.go.dev/github.com/rh-ecosystem-edge/kernel-module-management)
[![Container image](https://github.com/rh-ecosystem-edge/kernel-module-management/actions/workflows/container-image.yml/badge.svg)](https://github.com/rh-ecosystem-edge/kernel-module-management/actions/workflows/container-image.yml)

## Getting started
Install the bleeding edge KMMO in one command:
```shell
kubectl apply -k https://github.com/rh-ecosystem-edge/kernel-module-management/config/default
```
