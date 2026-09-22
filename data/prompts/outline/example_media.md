# Example: outlining todo tasks for media driver

`(Tool call: <tool> <target>)` marks a tool call and the code block after it is the tool result. `/* ... */` marks lines omitted in this example only; real tool results are complete.

## Input

```c
static const struct media_file_operations media_device_fops = {
    .owner = THIS_MODULE,
    .open = media_device_open,
    .ioctl = media_device_ioctl,
#ifdef CONFIG_COMPAT
    .compat_ioctl = media_device_compat_ioctl,
#endif /* CONFIG_COMPAT */
    .release = media_device_close,
}
```

## Output

### Thought

`.ioctl = media_device_ioctl` is the command entry point.

(Tool call: get_func_code_by_name `media_device_ioctl`)

```c
static long media_device_ioctl(struct file *filp, unsigned int cmd,
			       unsigned long __arg)
{
	/* ... */
	if (_IOC_NR(cmd) >= ARRAY_SIZE(ioctl_info)
	    || ioctl_info[_IOC_NR(cmd)].cmd != cmd)
		return -ENOIOCTLCMD;

	info = &ioctl_info[_IOC_NR(cmd)];
	/* ... */
	ret = info->fn(dev, karg);
	/* ... */
}
```

Commands are dispatched through the table `ioctl_info`.

(Tool call: get_global_var_code_by_name `ioctl_info`)

```c
static const struct media_ioctl_info ioctl_info[] = {
	MEDIA_IOC(DEVICE_INFO, media_device_get_info, MEDIA_IOC_FL_GRAPH_MUTEX),
	MEDIA_IOC(ENUM_ENTITIES, media_device_enum_entities, MEDIA_IOC_FL_GRAPH_MUTEX),
	MEDIA_IOC(ENUM_LINKS, media_device_enum_links, MEDIA_IOC_FL_GRAPH_MUTEX),
	MEDIA_IOC(SETUP_LINK, media_device_setup_link, MEDIA_IOC_FL_GRAPH_MUTEX),
	MEDIA_IOC(G_TOPOLOGY, media_device_get_topology, MEDIA_IOC_FL_GRAPH_MUTEX),
	MEDIA_IOC(REQUEST_ALLOC, media_device_request_alloc, 0),
};
```

Entries use macro `MEDIA_IOC`; resolve how it builds the command.

(Tool call: get_macro_def_code_by_name `MEDIA_IOC`)

```c
#define MEDIA_IOC(__cmd, func, fl)					\
	MEDIA_IOC_ARG(__cmd, func, fl, copy_arg_from_user, copy_arg_to_user)
```

(Tool call: get_macro_def_code_by_name `MEDIA_IOC_ARG`)

```c
#define MEDIA_IOC_ARG(__cmd, func, fl, from_user, to_user)		\
	[_IOC_NR(MEDIA_IOC_##__cmd)] = {				\
		.cmd = MEDIA_IOC_##__cmd,				\
		.fn = func,						\
		.flags = fl,						\
		.arg_from_user = from_user,				\
		.arg_to_user = to_user,					\
	}
```

`MEDIA_IOC_##__cmd` concatenates, e.g. `DEVICE_INFO` → `MEDIA_IOC_DEVICE_INFO`; same for all 6 entries.

`init_syscall`: media nodes are multi-instance (`/dev/media0`, `/dev/media1`, ...) → `syz_open_dev$media`, not a static `openat`.

### Final output

```json
[
	{"type": "init_syscall", "name": "syz_open_dev$media"},
	{"type": "syscall", "name": "ioctl$MEDIA_IOC_DEVICE_INFO"},
	{"type": "syscall", "name": "ioctl$MEDIA_IOC_ENUM_ENTITIES"},
	{"type": "syscall", "name": "ioctl$MEDIA_IOC_ENUM_LINKS"},
	{"type": "syscall", "name": "ioctl$MEDIA_IOC_SETUP_LINK"},
	{"type": "syscall", "name": "ioctl$MEDIA_IOC_G_TOPOLOGY"},
	{"type": "syscall", "name": "ioctl$MEDIA_IOC_REQUEST_ALLOC"}
]
```
