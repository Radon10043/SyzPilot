# Example of writing spec for media driver

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

## Output

### Thought

**Task:** Processing ioctl$MEDIA_IOC_ENUM_LINKS.

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

Check source of `ioctl_info`.

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

Found handler  `media_device_enum_links`.

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

Argument `arg` is cast to `struct media_links_enum`. Both read and write are performed to `arg`, implies `inout`.

Checking source of `struct media_links_enum`.

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

`struct media_links_enum` contains `struct media_pad_desc` and `struct media_link_desc`.

(Calling tool to get source of `struct media_pad_desc`)

```c
struct media_pad_desc {
	__u32 entity;		/* entity ID */
	__u16 index;		/* pad index */
	__u32 flags;		/* pad flags */
	__u32 reserved[2];
};
```

(Calling tool to get source of `struct media_link_desc`)

```c
struct media_link_desc {
	struct media_pad_desc source;
	struct media_pad_desc sink;
	__u32 flags;
	__u32 reserved[2];
};
```

`struct media_pad_desc` and `struct media_link_desc` contains field `flags`, maybe they have specified values. Based on my experience, I should get macros whose prefix is `MEDIA_PAD_FL_` and `MEDIA_LNK_FL_`.

(Calling tool to get source of macros whose prefix is `MEDIA_PAD_FL_`)

```c
#define MEDIA_PAD_FL_SINK			(1 << 0)
#define MEDIA_PAD_FL_SOURCE			(1 << 1)
#define MEDIA_PAD_FL_MUST_CONNECT		(1 << 2)
```

(Calling tool to get source of macros whose prefix is `MEDIA_LNK_FL_`)

```c
#define MEDIA_LNK_FL_ENABLED			(1 << 0)
#define MEDIA_LNK_FL_IMMUTABLE			(1 << 1)
#define MEDIA_LNK_FL_DYNAMIC			(1 << 2)

#define MEDIA_LNK_FL_LINK_TYPE			(0xf << 28)
#  define MEDIA_LNK_FL_DATA_LINK		(0 << 28)
#  define MEDIA_LNK_FL_INTERFACE_LINK		(1 << 28)
```

`MEDIA_LNK_FL_LINK_TYPE` is a mask, we shouldn't put it into flags. I also note that `entity` in `arg` are pssed to `find_entity`, check it source to ensure completeness.

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

`entity` in `struct media_pad_desc` should be a valid id of media entity.

### Final output

```json
{
    "include": [
        "include <uapi/linux/media.h>"
    ],
    "resource": [
        "resource fd_media[fd]",
        "resource media_entity_id[int32]: MEDIA_ENT_ID_FLAG_NEXT"
    ],
    "define": "define MEDIA_ENTITY_DESC_SIZE	sizeof(struct media_entity_desc)",
    "syscall": [
        "syz_open_dev$media(dev ptr[in, string[\"/dev/media#\"]], id intptr, flags flags[open_flags]) fd_media",
        "ioctl$MEDIA_IOC_ENUM_LINKS(fd fd_media, cmd const[MEDIA_IOC_ENUM_LINKS], arg ptr[inout, media_links_enum])",
    ],
    "flags": [
        "media_link_flags = MEDIA_LNK_FL_ENABLED, MEDIA_LNK_FL_IMMUTABLE, MEDIA_LNK_FL_DYNAMIC, MEDIA_LNK_FL_DATA_LINK, MEDIA_LNK_FL_INTERFACE_LINK",
        "media_pad_flags = MEDIA_PAD_FL_SINK, MEDIA_PAD_FL_SOURCE, MEDIA_PAD_FL_MUST_CONNECT"
    ],
    "struct": [
        "media_links_enum {\n\t\tentity\t\tmedia_entity_id\t(in)\n\tpads\t\tptr[out, array[media_pad_desc, 2]]\n\tlinks\t\tptr[out, array[media_link_desc, 2]]\n\treserved\tarray[const[0, int32], 4]\n}",
        "media_link_desc {\n\tsource\t\tmedia_pad_desc\n\tsink\t\tmedia_pad_desc\n\tflags\t\tflags[media_link_flags, int32]\n\treserved\tarray[int32, 2]\n}",
        "media_pad_desc {\n\tentity\t\tmedia_entity_id\n\tindex\t\tmedia_pad_index\n\tflags\t\tflags[media_pad_flags, int32]\n\treserved\tarray[int32, 2]\n}"
    ],
    "union": [],
    "type-alias": [],
    "type-template": [],
    "todo": [
        "ioctl$MEDIA_IOC_DEVICE_INFO",
        "ioctl$MEDIA_IOC_ENUM_ENTITIES",
        "ioctl$MEDIA_IOC_SETUP_LINK",
        "ioctl$MEDIA_IOC_G_TOPOLOGY",
        "ioctl$MEDIA_IOC_REQUEST_ALLOC",
    ]
}
```