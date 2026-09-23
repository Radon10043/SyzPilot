# setup image/debdroid

(container.syzpilot): build a custom **deb**ian image for running an**droid** kernel:
```bash
mkdir -p $EXPERIMENT_ROOT/image/debdroid/bullseye && cd $EXPERIMENT_ROOT/image/debdroid/bullseye
cp $EXPERIMENT_ROOT/fuzzer/cloud/scripts/linux/create-image.sh . && chmod +x ./create-image.sh
./create-image.sh
```
