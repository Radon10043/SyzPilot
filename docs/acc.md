How do we calculate the accuracy of synthesized specs for a subsystem?

1. Extract all synthesized syscalls.
2. Read the kernel source and extract syscall related dependencies.
3. Read the synthesized specifications, extract the dependencies corresponding to the kernel elements from step 2, and check whether they align with the kernel.
4. Calculate accuracy for each syscall, then calculate accuracy for the subsystem.
