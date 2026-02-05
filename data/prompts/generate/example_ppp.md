# Example of writing spec for ppp driver (1)

## Input

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

Please write specification for initialization syscall `openat$ppp`.

### Thought

**Task**: Define openat$ppp for the ppp driver.

**Identify Device Path**: I need to determine the file path associated with this driver. Based on the name `ppp`, it is likely a char device located at `/dev/ppp`. I will verify this by checking the `ppp_class` definition.

(Calling tool to get source code of variable `ppp_class`).

```c
static const struct class ppp_class = {
	.name = "ppp",
};
```

**Observation**: The class name is "ppp", confirming the device path is `/dev/ppp`.
**Dependencies**: To use `AT_FDCWD`, I need to include `linux/fcntl.h`.
**Resource Definition**: Since this opens a specific device, I should define a resource `fd_ppp` that depends on `fd`.

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

# Example of writing spec for ppp driver (2)

## Input

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

Please write specification for syscall `ioctl$PPPIOCSNPMODE`.

## Output

### Thought

**Task**: Create a specification for `ioctl$PPPIOCSNPMODE`.

**Analyze Handler**: I need to examine `ppp_ioctl` to understand how `PPPIOCSNPMODE` is handled and what arguments it expects.

(Calling tool to get source code of function `ppp_ioctl`).

```c
static long ppp_ioctl(struct file *file, unsigned int cmd, unsigned long arg)
{
	struct ppp_file *pf;
	struct ppp *ppp;
	int err = -EFAULT, val, val2, i;
	struct ppp_idle32 idle32;
	struct ppp_idle64 idle64;
	struct npioctl npi;
	int unit, cflags;
	struct slcompress *vj;
	void __user *argp = (void __user *)arg;
	int __user *p = argp;

	mutex_lock(&ppp_mutex);

	pf = file->private_data;
	if (!pf) {
		err = ppp_unattached_ioctl(current->nsproxy->net_ns,
					   pf, file, cmd, arg);
		goto out;
	}

	if (cmd == PPPIOCDETACH) {
		/*
		 * PPPIOCDETACH is no longer supported as it was heavily broken,
		 * and is only known to have been used by pppd older than
		 * ppp-2.4.2 (released November 2003).
		 */
		pr_warn_once("%s (%d) used obsolete PPPIOCDETACH ioctl\n",
			     current->comm, current->pid);
		err = -EINVAL;
		goto out;
	}

	if (pf->kind == CHANNEL) {
		struct channel *pch, *pchb;
		struct ppp_channel *chan;
		struct ppp_net *pn;

		pch = PF_TO_CHANNEL(pf);

		switch (cmd) {
		case PPPIOCCONNECT:
			if (get_user(unit, p))
				break;
			err = ppp_connect_channel(pch, unit);
			break;

		case PPPIOCDISCONN:
			err = ppp_disconnect_channel(pch);
			break;

		case PPPIOCBRIDGECHAN:
			if (get_user(unit, p))
				break;
			err = -ENXIO;
			pn = ppp_pernet(current->nsproxy->net_ns);
			spin_lock_bh(&pn->all_channels_lock);
			pchb = ppp_find_channel(pn, unit);
			/* Hold a reference to prevent pchb being freed while
			 * we establish the bridge.
			 */
			if (pchb)
				refcount_inc(&pchb->file.refcnt);
			spin_unlock_bh(&pn->all_channels_lock);
			if (!pchb)
				break;
			err = ppp_bridge_channels(pch, pchb);
			/* Drop earlier refcount now bridge establishment is complete */
			if (refcount_dec_and_test(&pchb->file.refcnt))
				ppp_destroy_channel(pchb);
			break;

		case PPPIOCUNBRIDGECHAN:
			err = ppp_unbridge_channels(pch);
			break;

		default:
			down_read(&pch->chan_sem);
			chan = pch->chan;
			err = -ENOTTY;
			if (chan && chan->ops->ioctl)
				err = chan->ops->ioctl(chan, cmd, arg);
			up_read(&pch->chan_sem);
		}
		goto out;
	}

	if (pf->kind != INTERFACE) {
		/* can't happen */
		pr_err("PPP: not interface or channel??\n");
		err = -EINVAL;
		goto out;
	}

	ppp = PF_TO_PPP(pf);
	switch (cmd) {
	case PPPIOCSMRU:
		if (get_user(val, p))
			break;
		ppp->mru = val;
		err = 0;
		break;

	case PPPIOCSFLAGS:
		if (get_user(val, p))
			break;
		ppp_lock(ppp);
		cflags = ppp->flags & ~val;
#ifdef CONFIG_PPP_MULTILINK
		if (!(ppp->flags & SC_MULTILINK) && (val & SC_MULTILINK))
			ppp->nextseq = 0;
#endif
		ppp->flags = val & SC_FLAG_BITS;
		ppp_unlock(ppp);
		if (cflags & SC_CCP_OPEN)
			ppp_ccp_closed(ppp);
		err = 0;
		break;

	case PPPIOCGFLAGS:
		val = ppp->flags | ppp->xstate | ppp->rstate;
		if (put_user(val, p))
			break;
		err = 0;
		break;

	case PPPIOCSCOMPRESS:
	{
		struct ppp_option_data data;
		if (copy_from_user(&data, argp, sizeof(data)))
			err = -EFAULT;
		else
			err = ppp_set_compress(ppp, &data);
		break;
	}
	case PPPIOCGUNIT:
		if (put_user(ppp->file.index, p))
			break;
		err = 0;
		break;

	case PPPIOCSDEBUG:
		if (get_user(val, p))
			break;
		ppp->debug = val;
		err = 0;
		break;

	case PPPIOCGDEBUG:
		if (put_user(ppp->debug, p))
			break;
		err = 0;
		break;

	case PPPIOCGIDLE32:
                idle32.xmit_idle = (jiffies - ppp->last_xmit) / HZ;
                idle32.recv_idle = (jiffies - ppp->last_recv) / HZ;
                if (copy_to_user(argp, &idle32, sizeof(idle32)))
			break;
		err = 0;
		break;

	case PPPIOCGIDLE64:
		idle64.xmit_idle = (jiffies - ppp->last_xmit) / HZ;
		idle64.recv_idle = (jiffies - ppp->last_recv) / HZ;
		if (copy_to_user(argp, &idle64, sizeof(idle64)))
			break;
		err = 0;
		break;

	case PPPIOCSMAXCID:
		if (get_user(val, p))
			break;
		val2 = 15;
		if ((val >> 16) != 0) {
			val2 = val >> 16;
			val &= 0xffff;
		}
		vj = slhc_init(val2+1, val+1);
		if (IS_ERR(vj)) {
			err = PTR_ERR(vj);
			break;
		}
		ppp_lock(ppp);
		if (ppp->vj)
			slhc_free(ppp->vj);
		ppp->vj = vj;
		ppp_unlock(ppp);
		err = 0;
		break;

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

#ifdef CONFIG_PPP_FILTER
	case PPPIOCSPASS:
	case PPPIOCSACTIVE:
	{
		struct bpf_prog *filter = ppp_get_filter(argp);
		struct bpf_prog **which;

		if (IS_ERR(filter)) {
			err = PTR_ERR(filter);
			break;
		}
		if (cmd == PPPIOCSPASS)
			which = &ppp->pass_filter;
		else
			which = &ppp->active_filter;
		ppp_lock(ppp);
		if (*which)
			bpf_prog_destroy(*which);
		*which = filter;
		ppp_unlock(ppp);
		err = 0;
		break;
	}
#endif /* CONFIG_PPP_FILTER */

#ifdef CONFIG_PPP_MULTILINK
	case PPPIOCSMRRU:
		if (get_user(val, p))
			break;
		ppp_recv_lock(ppp);
		ppp->mrru = val;
		ppp_recv_unlock(ppp);
		err = 0;
		break;
#endif /* CONFIG_PPP_MULTILINK */

	default:
		err = -ENOTTY;
	}

out:
	mutex_unlock(&ppp_mutex);

	return err;
}
```

**Analysis**:
- The handler uses `copy_from_user` for `PPPIOCSNPMODE`, indicating the argument direction is `in`.
- The data structure used is `npi`, which is of type `struct npioctl`.
- The fd argument requires the `fd_ppp` resource defined in the initialization step.

**Scope**:
- The definition of `struct npioctl` is required but not yet defined. I will add it to the `required` list.
- The initialization syscall `openat$ppp` is also a dependency.
- Headers `uapi/linux/ppp-ioctl.h` and `linux/ioctl.h` are needed for constants.

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
		{"type": "init_syscall", "name": "openat$ppp"},
	]
}
```

# Example of writing spec for pp driver (3)

## Input

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

Please write specification for struct `npioctl`.

## Output

### Thought

**Task**: Define `struct npioctl` and its field types.

**Inspect Struct**: I need to see the definition of `struct npioctl`.

(Calling tool to get code of `struct npioctl`)

```c
struct npioctl {
	int		protocol;	/* PPP protocol, e.g. PPP_IP */
	enum NPmode	mode;
};
```

**Inspect Enum**: I need to see the definition of `enum NPmode`.

(Calling tool to get code of `enum NPmode`)

```c
enum NPmode {
    NPMODE_PASS,		/* pass the packet through */
    NPMODE_DROP,		/* silently drop the packet */
    NPMODE_ERROR,		/* return an error */
    NPMODE_QUEUE		/* save it up for later. */
};
```

**Identify Protocol Values**: The `protocol` field is an `int`. Based on the comment "e.g. PPP_IP", I should search for macros matching the pattern `PPP_*` to define a flag set.

(Calling tool to get macro definitions whose pattern is `PPP_*`)

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
#define PPP_CONTROL(p)  (((__u8 *)(p))[1])
#define PPP_ESCAPE      0x7d
#define PPP_FCS(fcs, c) crc_ccitt_byte(fcs, c)
#define PPP_FCSLEN      2
#define PPP_FLAG        0x7e
#define PPP_GOODFCS     0xf0b8
#define PPP_HDRLEN      4
#define PPP_INITFCS     0xffff
#define PPP_IP          0x21
#define PPP_IPCP        0x8021
#define PPP_IPV6        0x57
#define PPP_IPV6CP      0x8057
#define PPP_IPX         0x2b
#define PPP_IPXCP       0x802b
#define PPP_LCP         0xc021
#define PPP_LCP_ECHOREP 0x0A
#define PPP_LCP_ECHOREQ 0x09
#define PPP_LCP_HDRLEN  4
#define PPP_LQR         0xc025
#define PPP_MAJOR       108
#define PPP_MAX_RQLEN   32
#define PPP_MP          0x3d
#define PPP_MPLSCP      0x80fd
#define PPP_MPLS_MC     0x0283
#define PPP_MPLS_UC     0x0281
#define PPP_MP_CB(skb)  ((struct ppp_mp_skb_parm *)((skb)->cb))
#define PPP_MP_MAX_QLEN 128
#define PPP_MRU         1500
#define PPP_PAP         0xc023
#define PPP_PROTO(skb)  get_unaligned_be16((skb)->data)
#define PPP_PROTOCOL(p) ((((__u8 *)(p))[2] << 8) + ((__u8 *)(p))[3])
#define PPP_PROTO_LEN   2
#define PPP_TRANS       0x20
#define PPP_UI          0x03
#define PPP_VERSION     "2.4.2"
#define PPP_VERSION     "2.4.2"
#define PPP_VERSION     "2.4.2"
#define PPP_VJC_COMP    0x2d
#define PPP_VJC_UNCOMP  0x2f
```

**Selection Strategy**: I will filter for macros that represent protocols (usually hex values) and exclude helper macros (like `PPP_ADDRESS`, `PPP_VERSION`).

**Conclusion**:
- `protocol` should use a flag set `ppp_proto`.
- `mode` should use a flag set `NPmode`.
- Header `uapi/linux/ppp_defs.h` is likely required for these definitions.

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
