# setup environment file

> [!WARNING]
> Please ensure that the mount point of the experiment directory in the container is consistent with the previous setup, otherwise it will cause kernel coverage filtering and issue report symbolization failure.

(host) setup an environment file:
```bash
echo "EXPERIMENT_CPUS=[NUMBER_OF_CPU_FOR_EACH_CONTAINER]" > compose.env
echo "EXPERIMENT_VMCOUNT=[NUMBER_OF_VM_FOR_FUZZING]" >> compose.env
echo "EXPERIMENT_ROOT_HOST=[PATH_TO_ENVIRONMENT_ON_HOST]" >> compose.env
echo "EXPERIMENT_ROOT_CONTAINER=[MOUNT_POINT_IN_CONTAINER]" >> compose.env
```
