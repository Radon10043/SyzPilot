# Examples of writing spec for media driver

`(Tool call: <tool> <target>)` marks a tool call and the code block after it is the tool result. `/* ... */` marks lines omitted in this example only; real tool results are complete.

Every media example's input is this code block followed by its request line:

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

## Example 1

Request: Please write specification for init_syscall `syz_open_dev$media`

### Thought

Confirm the device node name instead of guessing.

(Tool call: get_func_code_by_name `media_devnode_register`)

```c
int __must_check media_devnode_register(struct media_device *mdev,
					struct media_devnode *devnode,
					struct module *owner)
{
	/* ... */
	minor = find_first_zero_bit(media_devnode_nums, MEDIA_NUM_DEVICES);
	/* ... */
	dev_set_name(&devnode->dev, "media%d", devnode->minor);
	device_initialize(&devnode->dev);

	/* Part 2: Initialize the character device */
	cdev_init(&devnode->cdev, &media_devnode_fops);
	/* ... */
}
```

- Name format `media%d` → path `/dev/media#`; `syz_open_dev` takes `dev` (string), `id` (index), `flags`.
- Define resource `fd_media` based on `fd`.

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

## Example 2

Request: Please write specification for syscall `ioctl$MEDIA_IOC_ENUM_LINKS`

### Thought

Trace `media_device_ioctl` to the handler of `MEDIA_IOC_ENUM_LINKS`.

(Tool call: get_func_code_by_name `media_device_ioctl`)

```c
static long media_device_ioctl(struct file *filp, unsigned int cmd,
			       unsigned long __arg)
{
	/* ... */
	info = &ioctl_info[_IOC_NR(cmd)];
	/* ... */
	if (info->arg_from_user) {
		ret = info->arg_from_user(karg, arg, cmd);
		if (ret)
			goto out_free;
	}
	/* ... */
	ret = info->fn(dev, karg);
	/* ... */
	if (!ret && info->arg_to_user)
		ret = info->arg_to_user(arg, karg, cmd);
	/* ... */
}
```

(Tool call: get_global_var_code_by_name `ioctl_info`)

```c
static const struct media_ioctl_info ioctl_info[] = {
	/* ... */
	MEDIA_IOC(ENUM_LINKS, media_device_enum_links, MEDIA_IOC_FL_GRAPH_MUTEX),
	/* ... */
};
```

(Tool call: get_func_code_by_name `media_device_enum_links`)

```c
static long media_device_enum_links(struct media_device *mdev, void *arg)
{
	struct media_links_enum *links = arg;
	struct media_entity *entity;

	entity = find_entity(mdev, links->entity);
	if (entity == NULL)
		return -EINVAL;

	if (links->pads) {
		/* ... */
			if (copy_to_user(&links->pads[p], &pad, sizeof(pad)))
				return -EFAULT;
		/* ... */
	}

	if (links->links) {
		/* ... */
			if (copy_to_user(ulink_desc, &klink_desc,
					 sizeof(*ulink_desc)))
				return -EFAULT;
		/* ... */
	}
	memset(links->reserved, 0, sizeof(links->reserved));

	return 0;
}
```

- Argument is `struct media_links_enum`: `links->entity` is read (`in`), `pads`/`links` are written back (`out`) → `ptr[inout, ...]`.
- `fd_media` (from `syz_open_dev$media`) and struct `media_links_enum` are not defined here → `required`.

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

## Example 3

Request: Please write specification for struct `media_links_enum`

### Thought

(Tool call: get_struct_code_by_name `media_links_enum`)

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

`entity` is passed to `find_entity`; verify it is treated as an ID.

(Tool call: get_func_code_by_name `find_entity`)

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

- `entity` → resource `media_entity_id` (special value `MEDIA_ENT_ID_FLAG_NEXT`), `in`.
- `pads`, `links` → kernel-filled arrays, `ptr[out, array[...]]`; element structs go to `required`.

### Final output

```json
{
	"spec": [
		{"type": "include", "name": "uapi/linux/media.h", "code": "include <uapi/linux/media.h>"},
		{"type": "resource", "name": "media_entity_id", "code": "resource media_entity_id[int32]: MEDIA_ENT_ID_FLAG_NEXT"},
		{"type": "struct", "name": "media_links_enum", "code": "media_links_enum {\n\tentity\t\tmedia_entity_id\t(in)\n\tpads\t\tptr[out, array[media_pad_desc, 2]]\n\tlinks\t\tptr[out, array[media_link_desc, 2]]\n\treserved\tarray[const[0, int32], 4]\n}"}
	],
	"required": [
		{"type": "struct", "name": "media_pad_desc"},
		{"type": "struct", "name": "media_link_desc"}
	]
}
```

## Example 4

Request: Please write specification for struct `media_pad_desc`

### Thought

(Tool call: get_struct_code_by_name `media_pad_desc`)

```c
struct media_pad_desc {
	__u32 entity;		/* entity ID */
	__u16 index;		/* pad index */
	__u32 flags;		/* pad flags */
	__u32 reserved[2];
};
```

`entity` → `media_entity_id`; `index` → `media_pad_index`; `flags` needs a flag set from `MEDIA_PAD_FL_*`.

(Tool call: get_macro_def_codes_by_pattern `MEDIA_PAD_FL_*`)

```c
#define MEDIA_PAD_FL_SINK			(1 << 0)
#define MEDIA_PAD_FL_SOURCE			(1 << 1)
#define MEDIA_PAD_FL_MUST_CONNECT		(1 << 2)
```

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

## Example 5

Request: Please write specification for struct `media_link_desc`

### Thought

(Tool call: get_struct_code_by_name `media_link_desc`)

```c
struct media_link_desc {
	struct media_pad_desc source;
	struct media_pad_desc sink;
	__u32 flags;
	__u32 reserved[2];
};
```

`source`/`sink` reuse `media_pad_desc`; `flags` needs a flag set from `MEDIA_LNK_FL_*`.

(Tool call: get_macro_def_codes_by_pattern `MEDIA_LNK_FL_*`)

```c
#define MEDIA_LNK_FL_ENABLED			(1 << 0)
#define MEDIA_LNK_FL_IMMUTABLE			(1 << 1)
#define MEDIA_LNK_FL_DYNAMIC			(1 << 2)

#define MEDIA_LNK_FL_LINK_TYPE			(0xf << 28)
#  define MEDIA_LNK_FL_DATA_LINK		(0 << 28)
#  define MEDIA_LNK_FL_INTERFACE_LINK		(1 << 28)
```

`MEDIA_LNK_FL_LINK_TYPE` is a mask → exclude it, but include its values `MEDIA_LNK_FL_DATA_LINK` and `MEDIA_LNK_FL_INTERFACE_LINK`.

### Final output

```json
{
	"spec": [
		{"type": "include", "name": "uapi/linux/media.h", "code": "include <uapi/linux/media.h>"},
		{"type": "flag", "name": "media_link_flags", "code": "media_link_flags = MEDIA_LNK_FL_ENABLED, MEDIA_LNK_FL_IMMUTABLE, MEDIA_LNK_FL_DYNAMIC, MEDIA_LNK_FL_DATA_LINK, MEDIA_LNK_FL_INTERFACE_LINK"},
		{"type": "struct", "name": "media_link_desc", "code": "media_link_desc {\n\tsource\t\tmedia_pad_desc\n\tsink\t\tmedia_pad_desc\n\tflags\t\tflags[media_link_flags, int32]\n\treserved\tarray[int32, 2]\n}"}
	],
	"required": []
}
```
