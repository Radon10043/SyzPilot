# Examples of writing spec for ppp driver

`(Tool call: <tool> <target>)` marks a tool call and the code block after it is the tool result. `/* ... */` marks lines omitted in this example only; real tool results are complete.

Every ppp example's input is this code block followed by its request line:

```c
static const struct file_operations ppp_device_fops = {
	.owner		= THIS_MODULE,
	.read		= ppp_read,
	.write		= ppp_write,
	.poll		= ppp_poll,
	.unlocked_ioctl	= ppp_ioctl,
#ifdef CONFIG_COMPAT
	.compat_ioctl	= ppp_compat_ioctl,
#endif
	.open		= ppp_open,
	.release	= ppp_release,
	.llseek		= noop_llseek,
};
```

## Example 1

Request: Please write specification for init_syscall `openat$ppp`

### Thought

Confirm the device path via the driver's class instead of guessing.

(Tool call: get_global_var_code_by_name `ppp_class`)

```c
static const struct class ppp_class = {
	.name = "ppp",
};
```

- Class name `ppp` → single node `/dev/ppp`, opened with `openat`; `AT_FDCWD` needs `linux/fcntl.h`.
- Define resource `fd_ppp` based on `fd`.

### Final output

```json
{
	"spec": [
		{"type": "include", "name": "linux/fcntl.h", "code": "include <linux/fcntl.h>"},
		{"type": "resource", "name": "fd_ppp", "code": "resource fd_ppp[fd]"},
		{"type": "init_syscall", "name": "openat$ppp", "code": "openat$ppp(fd const[AT_FDCWD], file ptr[in, string[\"/dev/ppp\"]], flags flags[open_flags], mode const[0]) fd_ppp"}
	],
	"required": []
}
```

## Example 2

Request: Please write specification for syscall `ioctl$PPPIOCSNPMODE`

### Thought

(Tool call: get_func_code_by_name `ppp_ioctl`)

```c
static long ppp_ioctl(struct file *file, unsigned int cmd, unsigned long arg)
{
	/* ... */
	struct npioctl npi;
	/* ... */
	void __user *argp = (void __user *)arg;
	/* ... */
	ppp = PF_TO_PPP(pf);
	switch (cmd) {
	/* ... */
	case PPPIOCGNPMODE:
	case PPPIOCSNPMODE:
		if (copy_from_user(&npi, argp, sizeof(npi)))
			break;
		err = proto_to_npindex(npi.protocol);
		if (err < 0)
			break;
		i = err;
		if (cmd == PPPIOCGNPMODE) {
			err = -EFAULT;
			npi.mode = ppp->npmode[i];
			if (copy_to_user(argp, &npi, sizeof(npi)))
				break;
		} else {
			ppp->npmode[i] = npi.mode;
			/* we may be able to transmit more packets now (??) */
			netif_wake_queue(ppp->dev);
		}
		err = 0;
		break;
	/* ... */
	}
	/* ... */
}
```

- `PPPIOCSNPMODE` only does `copy_from_user` into `struct npioctl` → `ptr[in, npioctl]`; `fd` is `fd_ppp`.
- Constants need `uapi/linux/ppp-ioctl.h` and `linux/ioctl.h`.
- Struct `npioctl` and init_syscall `openat$ppp` are not defined here → `required`.

### Final output

```json
{
	"spec": [
		{"type": "include", "name": "uapi/linux/ppp-ioctl.h", "code": "include <uapi/linux/ppp-ioctl.h>"},
		{"type": "include", "name": "linux/ioctl.h", "code": "include <linux/ioctl.h>"},
		{"type": "syscall", "name": "ioctl$PPPIOCSNPMODE", "code": "ioctl$PPPIOCSNPMODE(fd fd_ppp, cmd const[PPPIOCSNPMODE], arg ptr[in, npioctl])"}
	],
	"required": [
		{"type": "struct", "name": "npioctl"},
		{"type": "init_syscall", "name": "openat$ppp"}
	]
}
```

## Example 3

Request: Please write specification for struct `npioctl`

### Thought

(Tool call: get_struct_code_by_name `npioctl`)

```c
struct npioctl {
	int		protocol;	/* PPP protocol, e.g. PPP_IP */
	enum NPmode	mode;
};
```

(Tool call: get_enum_code_by_specifier `NPmode`)

```c
enum NPmode {
    NPMODE_PASS,		/* pass the packet through */
    NPMODE_DROP,		/* silently drop the packet */
    NPMODE_ERROR,		/* return an error */
    NPMODE_QUEUE		/* save it up for later. */
};
```

`protocol` is an `int`; the comment "e.g. PPP_IP" points to the `PPP_*` macros.

(Tool call: get_macro_def_codes_by_pattern `PPP_*`)

```c
#define PPP_ADDRESS(p)  (((__u8 *)(p))[0])
#define PPP_ALLSTATIONS 0xff
#define PPP_AT          0x29
#define PPP_ATCP        0x8029
#define PPP_CBCP        0xc029
#define PPP_CCP         0x80fd
#define PPP_CCPFRAG     0x80fb
#define PPP_CHAP        0xc223
#define PPP_COMP        0xfd
#define PPP_COMPFRAG    0xfb
#define PPP_FLAG        0x7e
#define PPP_IP          0x21
#define PPP_IPCP        0x8021
#define PPP_IPV6        0x57
#define PPP_IPV6CP      0x8057
#define PPP_IPX         0x2b
#define PPP_IPXCP       0x802b
#define PPP_LCP         0xc021
#define PPP_LQR         0xc025
#define PPP_MP          0x3d
#define PPP_MPLSCP      0x80fd
#define PPP_MPLS_MC     0x0283
#define PPP_MPLS_UC     0x0281
#define PPP_MRU         1500
#define PPP_PAP         0xc023
#define PPP_VERSION     "2.4.2"
#define PPP_VJC_COMP    0x2d
#define PPP_VJC_UNCOMP  0x2f
/* ... more non-protocol macros ... */
```

- Keep only protocol numbers; drop function-like macros (`PPP_ADDRESS`), framing/size constants (`PPP_ALLSTATIONS`, `PPP_FLAG`, `PPP_MRU`) and strings (`PPP_VERSION`).
- `protocol` → `flags[ppp_proto, int32]`; `mode` → `flags[NPmode, int32]`; include `uapi/linux/ppp_defs.h`.

### Final output

```json
{
	"spec": [
		{"type": "include", "name": "uapi/linux/ppp_defs.h", "code": "include <uapi/linux/ppp_defs.h>"},
		{"type": "struct", "name": "npioctl", "code": "npioctl {\n\tprotocol\tflags[ppp_proto, int32]\n\tmode\t\tflags[NPmode, int32]\n}"},
		{"type": "flag", "name": "ppp_proto", "code": "ppp_proto = PPP_IP, PPP_AT, PPP_IPX, PPP_VJC_COMP, PPP_VJC_UNCOMP, PPP_MP, PPP_IPV6, PPP_COMPFRAG, PPP_COMP, PPP_MPLS_UC, PPP_MPLS_MC, PPP_IPCP, PPP_ATCP, PPP_IPXCP, PPP_IPV6CP, PPP_CCPFRAG, PPP_CCP, PPP_MPLSCP, PPP_LCP, PPP_PAP, PPP_LQR, PPP_CHAP, PPP_CBCP"},
		{"type": "flag", "name": "NPmode", "code": "NPmode = NPMODE_PASS, NPMODE_DROP, NPMODE_ERROR, NPMODE_QUEUE"}
	],
	"required": []
}
```
