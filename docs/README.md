# cloud

We use linux v6.12 as an example.

```bash
cd $KERNEL
cp $FUZZER/configs/syzbot.config .config
make CC="ccache clang-19" olddefconfig modules_prepare all -j16
python3 scripts/clang-tools/gen_compile_commands.py
```

Build analyzer:

```bash
make analyzer
```

Analyze kernel code:

```bash
./bin/analyzer -i $KERNEL/compile_commands.json -o data/database/linux.db -j 16
```
