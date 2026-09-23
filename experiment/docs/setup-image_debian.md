# setup image/debian

(container.syzpilot):
```bash
mkdir -p $EXPERIMENT_ROOT/image/debian/bullseye && cd $EXPERIMENT_ROOT/image/debian/bullseye
cp $EXPERIMENT_ROOT/fuzzer/SyzPilot/scripts/linux/create-image.sh . && chmod +x ./create-image.sh
./create-image.sh
```
