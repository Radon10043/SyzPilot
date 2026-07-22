# setup image/debian

(container.cloud):
```bash
mkdir -p $EXPERIMENT_ROOT/image/debian/bullseye && cd $EXPERIMENT_ROOT/image/debian/bullseye
cp $EXPERIMENT_ROOT/fuzzer/cloud/scripts/linux/create-image.sh . && chmod +x ./create-image.sh
./create-image.sh
```
