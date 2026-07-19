# ablation

> [!NOTE]
> generator-nodb and generator-noiter only support linux kernel, but they are easy to extend to other kernels I think.

build:
```bash
make ablation
```

generator-nodb: disable kernel database, only rely on LLM's inherent knowledge.
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

generator-noiter: disable iteration, directly let LLM to generate syscall specs.
```bash
./bin/generator-noiter \
    -db=./data/database/linux.db \
    -model=gemini-3-flash-preview \
    -outdir=./.workdir/ablation/noiter \
    -ref=./.workdir/ablation/ref.txt \
    -sysdir=./data/trimsys \
    -jobs=4
```

generator-trimtool: disable available tool(s) in spec generation.
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

generator-openllm: reuse generator but write open weight llm settings in environment file:
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

available tools:
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
