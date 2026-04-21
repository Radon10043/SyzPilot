How to calculate accuracy of synthesized specification for a subsystem?

1. extract all synthesized syscall
2. read kernel's source, extract syscall related dependencies
3. read synthesized specs, extract dependencies corresponding to kernel elements in step 2, check whether aligned with kernel
4. calculate accuracy for a syscall, calculate accuracy for a subsystem
