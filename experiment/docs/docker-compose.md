# docker-compose

## preparation

plase setup the environment variables in an env file before using this. you should set following variables in the env file:
- `EXPERIMENT_ROOT_HOST`: root directory of the experiment in the host machine
- `EXPERIMENT_ROOT_CONTAINER`: root directory of the experiment in the container, the docker compose file will mount `EXPERIMENT_ROOT_HOST` as `EXPERIMENT_ROOT_CONTAINER`
- `EXPERIMENT_CPUS`: number of CPUs allocated for each container
- `EXOERIMENT_VMCOUNT`: number of vm used for fuzzing, expect is EXPERIMENT_CPUS / 2

## usage

```bash
docker compose --env-file $ENV_FILE -f $DOCKER_COMPOSE_FILE up $SERVICE --scale $SERVICE=$NUM_INSTANCE -d
```

e.g.
```bash
docker compose --env-file ./compose.env -f ./docker-compose/compose.syzkaller.yaml up linux-v6.18-kernel --scale linux-v6.18-kernel=5 -d
```

schedule tasks:
```bash
echo "docker compose --env-file ./compose.env -f ./docker-compose/compose.syzkaller.yaml up linux-v6.18-kernel --scale linux-v6.18-kernel=5 -d" | at 18:00 7/20/2026
```
