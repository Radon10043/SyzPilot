# Ablation

**NOTE:** `generator-nodb` and `generator-noiter` currently support only the Linux kernel, but they should be easy to extend to other kernels.

Build:
```bash
make ablation
```

`generator-nodb`: disables the kernel database and relies only on the LLM's inherent knowledge.
```bash
./bin/generator-nodb \
    -db=./data/database/linux.db \
    -kernel=$KERNSRC \
    -model=gemini-3-flash-preview \
    -outdir=./workdir/ablation/nodb \
    -ref=./workdir/ablation/ref.txt \
    -sysdir=./data/trimsys \
    -jobs=4
```

`generator-noiter`: disables iteration and asks the LLM to generate syscall specs directly.
```bash
./bin/generator-noiter \
    -db=./data/database/linux.db \
    -model=gemini-3-flash-preview \
    -outdir=./workdir/ablation/noiter \
    -ref=./workdir/ablation/ref.txt \
    -sysdir=./data/trimsys \
    -jobs=4
```
