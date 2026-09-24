# Ablation

> [!NOTE]
> `generator-nodb` and `generator-noiter` currently support only the Linux kernel, but they should be easy to extend to other kernels.

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
    -outdir=./.workdir/ablation/nodb \
    -ref=./.workdir/ablation/ref.txt \
    -sysdir=./data/trimsys \
    -jobs=4
```

`generator-noiter`: disables iteration and asks the LLM to generate syscall specs directly.
```bash
./bin/generator-noiter \
    -db=./data/database/linux.db \
    -model=gemini-3-flash-preview \
    -outdir=./.workdir/ablation/noiter \
    -ref=./.workdir/ablation/ref.txt \
    -sysdir=./data/trimsys \
    -jobs=4
```

`generator-trimtool`: disables the specified tool(s) during spec generation. Use commas to separate multiple tools passed to `-disable`.
```bash
./bin/generator-trimtool \
    -db=./data/database/linux.db \
    -kernel=$KERNSRC \
    -os=linux \
    -model=gemini-3-flash-preview \
    -outdir=./.workdir/ablation/trimtool \
    -ref=./data/refs/linux/sg.txt \
    -sysdir=./data/trimsys \
    -disable=get_func_code_by_name \
    -jobs=4
```

`generator-openllm`: reuses `generator` and writes the open-weight LLM settings in a separate environment file:
```bash
./bin/generator \
    -env=./qwen.env \
    -db=./data/database/linux.db \
    -outdir=./.workdir/ablation/qwen3 \
    -kernel=$KERNSRC \
    -model=qwen3-235b-a22b-instruct-2507 \
    -ref=./data/refs/linux/autofs.txt \
    -sysdir=./data/trimsys \
    -os=linux \
    -jobs=2 > logs/autofs.log 2>&1 &
```

Available tools:
```
get_enum_code_by_enumerator
get_enum_code_by_specifier
get_func_code_by_name
get_global_var_code_by_name
get_macro_def_code_by_name
get_macro_def_loc_by_name
get_macro_def_codes_by_pattern
get_struct_code_by_name
get_union_code_by_name
get_typedef_code_by_define
get_typedef_type_by_define
```
