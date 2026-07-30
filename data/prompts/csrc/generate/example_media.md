# Example of writing spec for media driver (1)

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

Please write specification for initialization syscall `syz_open_dev$media`.

## Output

### Thought

**Task**: Define `syz_open_dev$media`.

**Identify Device Path**: I need to determine the device path pattern. Based on the name `media`, it is likely a char device under `/dev`. I will check `media_devnode_register` to confirm the naming convention.

(Calling tool to get source code of function `media_devnode_register`)

```c
int __must_check media_devnode_register(struct media_device *mdev,
					struct media_devnode *devnode,
					struct module *owner)
{
	int minor;
	int ret;

	/* Part 1: Find a free minor number */
	mutex_lock(&media_devnode_lock);
	minor = find_first_zero_bit(media_devnode_nums, MEDIA_NUM_DEVICES);
	if (minor == MEDIA_NUM_DEVICES) {
		mutex_unlock(&media_devnode_lock);
		pr_err("could not get a free minor\n");
		kfree(devnode);
		return -ENFILE;
	}

	set_bit(minor, media_devnode_nums);
	mutex_unlock(&media_devnode_lock);

	devnode->minor = minor;
	devnode->media_dev = mdev;

	/* Part 1: Initialize dev now to use dev.kobj for cdev.kobj.parent */
	devnode->dev.bus = &media_bus_type;
	devnode->dev.devt = MKDEV(MAJOR(media_dev_t), devnode->minor);
	devnode->dev.release = media_devnode_release;
	if (devnode->parent)
		devnode->dev.parent = devnode->parent;
	dev_set_name(&devnode->dev, "media%d", devnode->minor);
	device_initialize(&devnode->dev);

	/* Part 2: Initialize the character device */
	cdev_init(&devnode->cdev, &media_devnode_fops);
	devnode->cdev.owner = owner;
	kobject_set_name(&devnode->cdev.kobj, "media%d", devnode->minor);

	/* Part 3: Add the media and char device */
	set_bit(MEDIA_FLAG_REGISTERED, &devnode->flags);
	ret = cdev_device_add(&devnode->cdev, &devnode->dev);
	if (ret < 0) {
		clear_bit(MEDIA_FLAG_REGISTERED, &devnode->flags);
		pr_err("%s: cdev_device_add failed\n", __func__);
		goto cdev_add_error;
	}

	return 0;

cdev_add_error:
	mutex_lock(&media_devnode_lock);
	clear_bit(devnode->minor, media_devnode_nums);
	devnode->media_dev = NULL;
	mutex_unlock(&media_devnode_lock);

	put_device(&devnode->dev);
	return ret;
}
```

**Observation**: The device name format is "media%d", so the path is `/dev/media#`.
**Resource Definition**: Since this opens a specific device, I should define a resource `fd_media` dependent on `fd`.
**Syscall Signature**: The syscall `syz_open_dev` handles opening devices by name with an index. It takes `dev` (string), `id` (index), and `flags`.

### Final output

```json
{
	"spec": [
		{"type": "resource", "name": "fd_media", "code": "resource fd_media[fd]"},
		{"type": "init_syscall", "name": "syz_open_dev$media", "code": "syz_open_dev$media(dev ptr[in, string[\"/dev/media#\"]], id intptr, flags flags[open_flags]) fd_media"}
	],
	"required": []
}
```

# Example of writing spec for media driver (2)

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

Please write specification for syscall `ioctl$MEDIA_IOC_ENUM_LINKS`.

## Output

### Thought

**Task:** Create a specification for `ioctl$MEDIA_IOC_ENUM_LINKS`.

**Analyze Handler**: I need to trace `media_device_ioctl` to find the specific handler for `MEDIA_IOC_ENUM_LINKS`.

(Calling tool to get source code of function `media_device_ioctl`)

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

**Inspect Table**: Check `ioctl_info` to find the function pointer.

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

**Analyze Specific Handler**: Check `media_device_enum_links` to understand the argument structure and direction.

(Calling tool to get source of function `media_device_enum_links`)

```c
static long media_device_enum_links(struct media_device *mdev, void *arg)
{
	struct media_links_enum *links = arg;
	struct media_entity *entity;

	entity = find_entity(mdev, links->entity);
	if (entity == NULL)
		return -EINVAL;

	if (links->pads) {
		unsigned int p;

		for (p = 0; p < entity->num_pads; p++) {
			struct media_pad_desc pad;

			memset(&pad, 0, sizeof(pad));
			media_device_kpad_to_upad(&entity->pads[p], &pad);
			if (copy_to_user(&links->pads[p], &pad, sizeof(pad)))
				return -EFAULT;
		}
	}

	if (links->links) {
		struct media_link *link;
		struct media_link_desc __user *ulink_desc = links->links;

		list_for_each_entry(link, &entity->links, list) {
			struct media_link_desc klink_desc;

			/* Ignore backlinks. */
			if (link->source->entity != entity)
				continue;
			memset(&klink_desc, 0, sizeof(klink_desc));
			media_device_kpad_to_upad(link->source,
						  &klink_desc.source);
			media_device_kpad_to_upad(link->sink,
						  &klink_desc.sink);
			klink_desc.flags = link->flags;
			if (copy_to_user(ulink_desc, &klink_desc,
					 sizeof(*ulink_desc)))
				return -EFAULT;
			ulink_desc++;
		}
	}
	memset(links->reserved, 0, sizeof(links->reserved));

	return 0;
}
```

**Analysis**:
- The argument type is `struct media_links_enum`.
- The handler reads `links->entity` to find the target entity, making it `in`.
- The handler writes back to `links->pads` and `links->links`, making it `out`.
- Therefore, the pointer direction is `inout`.

### Final output

```json
{
	"spec": [
		{"type": "include", "name": "uapi/linux/media.h", "code": "include <uapi/linux/media.h>"},
		{"type": "syscall", "name": "ioctl$MEDIA_IOC_ENUM_LINKS", "code": "ioctl$MEDIA_IOC_ENUM_LINKS(fd fd_media, cmd const[MEDIA_IOC_ENUM_LINKS], arg ptr[inout, media_links_enum])"}
	],
	"required": [
		{"type": "init_syscall", "name": "syz_open_dev$media"},
		{"type": "struct", "name": "media_links_enum"}
	]
}
```

# Example of writing spec for media driver (3)

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

Please write specification for struct `media_links_enum`.

## Output

### Thought

**Task**: Define `struct media_links_enum`.

**Inspect Struct**: Get the definition of the struct.

(Calling tool to get source of `struct media_links_enum`)

```c
struct media_links_enum {
	__u32 entity;
	/* Should have enough room for pads elements */
	struct media_pad_desc *pads;
	/* Should have enough room for links elements */
	struct media_link_desc *links;
	__u32 reserved[4];
};
```

**Analyze Fields**:
- `entity`: Used to look up an entity. This should be a resource `media_entity_id`. `find_entity` implies there is logic handling ID validation.
- `pads`: A pointer to an array of `struct media_pad_desc`. Since the kernel fills this array, it is `out`.
- `links`: A pointer to an array of `struct media_link_desc`. Also `out`.

**Check Validation Logic**: Verify `entity` is treated as an ID.

(Calling tool to get source of function `find_entity`)

```c
static struct media_entity *find_entity(struct media_device *mdev, u32 id)
{
	struct media_entity *entity;
	int next = id & MEDIA_ENT_ID_FLAG_NEXT;

	id &= ~MEDIA_ENT_ID_FLAG_NEXT;

	media_device_for_each_entity(entity, mdev) {
		if (((media_entity_id(entity) == id) && !next) ||
		    ((media_entity_id(entity) > id) && next)) {
			return entity;
		}
	}

	return NULL;
}
```

**Conclusion**: Define `media_entity_id` as a resource. `pads` and `links` are pointers to arrays.

### Final output

```json
{
	"spec": [
		{"type": "include", "name": "uapi/linux/media.h", "code": "include <uapi/linux/media.h>"},
		{"type": "resource", "name": "media_entity_id", "code": "resource media_entity_id[int32]: MEDIA_ENT_ID_FLAG_NEXT"}
		{"type": "struct", "name": "media_links_enum", "code": "media_links_enum {\n\t\tentity\t\tmedia_entity_id\t(in)\n\tpads\t\tptr[out, array[media_pad_desc, 2]]\n\tlinks\t\tptr[out, array[media_link_desc, 2]]\n\treserved\tarray[const[0, int32], 4]\n}"}
	],
	"required": [
		{"type": "struct", "name": "media_pad_desc"},
		{"type": "struct", "name": "media_link_desc"}
	]
}
```

# Example of writing spec for media driver (4)

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

Please write specification for struct `media_pad_desc`.

## Output

### Thought

**Task**: Define `struct media_pad_desc`.

**Inspect Struct**: Get the definition.

(Calling tool to get source of `struct media_pad_desc`)

```c
struct media_pad_desc {
	__u32 entity;		/* entity ID */
	__u16 index;		/* pad index */
	__u32 flags;		/* pad flags */
	__u32 reserved[2];
};
```

**Analyze Fields**:
- **entity**: Uses `media_entity_id`.
- **index**: `__u16`. Standard integer.
- **flags**: Requires a flag set. I need to find macros starting with `MEDIA_PAD_FL_`.

(Calling tool to get source of macros whose prefix is `MEDIA_PAD_FL_`)

```c
#define MEDIA_PAD_FL_SINK			(1 << 0)
#define MEDIA_PAD_FL_SOURCE			(1 << 1)
#define MEDIA_PAD_FL_MUST_CONNECT		(1 << 2)
```

**Conclusion**: Create `media_pad_flags` with the found macros.

### Final output

```json
{
	"spec": [
		{"type": "include", "name": "uapi/linux/media.h", "code": "include <uapi/linux/media.h>"},
		{"type": "resource", "name": "media_entity_id", "code": "resource media_entity_id[int32]: MEDIA_ENT_ID_FLAG_NEXT"},
		{"type": "flag", "name": "media_pad_flags", "code": "media_pad_flags = MEDIA_PAD_FL_SINK, MEDIA_PAD_FL_SOURCE, MEDIA_PAD_FL_MUST_CONNECT"},
		{"type": "struct", "name": "media_pad_desc", "code": "media_pad_desc {\n\tentity\t\tmedia_entity_id\n\tindex\t\tmedia_pad_index\n\tflags\t\tflags[media_pad_flags, int32]\n\treserved\tarray[int32, 2]\n}"}
	],
	"required": []
}
```

# Example of writing spec for media driver (5)

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

Please write specification for struct `media_link_desc`.

## Output

### Thought

**Task**: Define struct `media_link_desc`.

**Inspect Struct**: Get the definition.

(Calling tool to get source of `struct media_link_desc`)

```c
struct media_link_desc {
	struct media_pad_desc source;
	struct media_pad_desc sink;
	__u32 flags;
	__u32 reserved[2];
};
```

**Analyze Fields**:
- `source`, `sink`: Use the already defined media_pad_desc.
- `flags`: Requires a flag set. Look for MEDIA_LNK_FL_.

(Calling tool to get source of macros whose prefix is `MEDIA_LNK_FL_`)

```c
#define MEDIA_LNK_FL_ENABLED			(1 << 0)
#define MEDIA_LNK_FL_IMMUTABLE			(1 << 1)
#define MEDIA_LNK_FL_DYNAMIC			(1 << 2)

#define MEDIA_LNK_FL_LINK_TYPE			(0xf << 28)
#  define MEDIA_LNK_FL_DATA_LINK		(0 << 28)
#  define MEDIA_LNK_FL_INTERFACE_LINK		(1 << 28)
```

**Analysis**:
- `MEDIA_LNK_FL_LINK_TYPE` is a mask and should not be included directly in the flags list.
- The specific values `MEDIA_LNK_FL_DATA_LINK` and `MEDIA_LNK_FL_INTERFACE_LINK` should be included.

### Final output

```json
{
	"spec": [
		{"type": "include", "name": "uapi/linux/media.h", "code": "include <uapi/linux/media.h>"},
		{"type": "flag", "name": "media_link_flags", "code": "media_link_flags = MEDIA_LNK_FL_ENABLED, MEDIA_LNK_FL_IMMUTABLE, MEDIA_LNK_FL_DYNAMIC, MEDIA_LNK_FL_DATA_LINK, MEDIA_LNK_FL_INTERFACE_LINK"},
		{"type": "struct", "name": "media_link_desc", "code": "media_link_desc {\n\tsource\t\tmedia_pad_desc\n\tsink\t\tmedia_pad_desc\n\tflags\t\tflags[media_link_flags, int32]\n\treserved\tarray[int32, 2]\n}"},
	],
	"required": []
}
```
