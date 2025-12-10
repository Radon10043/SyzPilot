# Example: outlining todo tasks for media driver

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

The struct `media_device_fops` points to `media_device_ioctl`.

(Calling tool to get source of function `media_device_ioctl`)

```c
static long media_device_ioctl(struct file *filp, unsigned int cmd,
			       unsigned long __arg)
{
	struct media_devnode *devnode = media_devnode_data(filp);
	struct media_device *dev = devnode->media_dev;
	const struct media_ioctl_info *info;
	void __user *arg = (void __user *)__arg;
	char __karg[256], *karg = __karg;
	long ret;

	if (_IOC_NR(cmd) >= ARRAY_SIZE(ioctl_info)
	    || ioctl_info[_IOC_NR(cmd)].cmd != cmd)
		return -ENOIOCTLCMD;

	info = &ioctl_info[_IOC_NR(cmd)];

	if (_IOC_SIZE(info->cmd) > sizeof(__karg)) {
		karg = kmalloc(_IOC_SIZE(info->cmd), GFP_KERNEL);
		if (!karg)
			return -ENOMEM;
	}

	if (info->arg_from_user) {
		ret = info->arg_from_user(karg, arg, cmd);
		if (ret)
			goto out_free;
	}

	if (info->flags & MEDIA_IOC_FL_GRAPH_MUTEX)
		mutex_lock(&dev->graph_mutex);

	ret = info->fn(dev, karg);

	if (info->flags & MEDIA_IOC_FL_GRAPH_MUTEX)
		mutex_unlock(&dev->graph_mutex);

	if (!ret && info->arg_to_user)
		ret = info->arg_to_user(arg, karg, cmd);

out_free:
	if (karg != __karg)
		kfree(karg);

	return ret;
}
```

The function uses `ioctl_info` array indexed by `_IOC_NR(cmd)`.

(Calling tool to get source of `ioctl_info`)

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

Found commands: `DEVICE_INFO`, `ENUM_ENTITIES`, etc. I need to know the definition of `struct media_ioctl_info` and `MEDIA_IOC`.

(Calling tool to get source of `struct media_ioctl_info`)

```c
struct media_ioctl_info {
	unsigned int cmd;
	unsigned short flags;
	long (*fn)(struct media_device *dev, void *arg);
	long (*arg_from_user)(void *karg, void __user *uarg, unsigned int cmd);
	long (*arg_to_user)(void __user *uarg, void *karg, unsigned int cmd);
};
```

(Calling tool to get source of macro `MEDIA_IOC`)

```c
#define MEDIA_IOC(__cmd, func, fl)					\
	MEDIA_IOC_ARG(__cmd, func, fl, copy_arg_from_user, copy_arg_to_user)
```

Need to understand `MEDIA_IOC_ARG`.

(Calling tool to get source of macro `MEDIA_IOC_ARG`)

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

Macro logic implies prefix `MEDIA_IOC_`. Constructed names: `ioctl$MEDIA_IOC_DEVICE_INFO`, etc.

Media devices are complex (multi-instance), so use `syz_open_dev`.

### Final output

```json
{
    "include": [],
    "resource": [],
    "define": [],
    "syscall": [],
    "flags": [],
    "struct": [],
    "union": [],
    "type-alias": [],
    "type-template": [],
    "todo": [
        "syz_open_dev$media",
        "ioctl$MEDIA_IOC_ENUM_LINKS"
        "ioctl$MEDIA_IOC_DEVICE_INFO",
        "ioctl$MEDIA_IOC_ENUM_ENTITIES",
        "ioctl$MEDIA_IOC_SETUP_LINK",
        "ioctl$MEDIA_IOC_G_TOPOLOGY",
        "ioctl$MEDIA_IOC_REQUEST_ALLOC"
    ],
}
```