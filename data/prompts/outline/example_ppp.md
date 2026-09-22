# Example: outlining todo tasks for ppp driver

`(Tool call: <tool> <target>)` marks a tool call and the code block after it is the tool result. `/* ... */` marks lines omitted in this example only; real tool results are complete.

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

## Output

### Thought

`.unlocked_ioctl = ppp_ioctl` is the command entry point.

(Tool call: get_func_code_by_name `ppp_ioctl`)

```c
static long ppp_ioctl(struct file *file, unsigned int cmd, unsigned long arg)
{
	/* ... */
	pf = file->private_data;
	if (!pf) {
		err = ppp_unattached_ioctl(current->nsproxy->net_ns,
					   pf, file, cmd, arg);
		goto out;
	}

	if (cmd == PPPIOCDETACH) {
		/* PPPIOCDETACH is no longer supported ... */
		err = -EINVAL;
		goto out;
	}

	if (pf->kind == CHANNEL) {
		/* ... */
		switch (cmd) {
		case PPPIOCCONNECT:	/* ... */
		case PPPIOCDISCONN:	/* ... */
		case PPPIOCBRIDGECHAN:	/* ... */
		case PPPIOCUNBRIDGECHAN:	/* ... */
		default:
			/* ... */
			if (chan && chan->ops->ioctl)
				err = chan->ops->ioctl(chan, cmd, arg);
			/* ... */
		}
		goto out;
	}
	/* ... */
	ppp = PF_TO_PPP(pf);
	switch (cmd) {
	case PPPIOCSMRU:	/* ... */
	case PPPIOCSFLAGS:	/* ... */
	case PPPIOCGFLAGS:	/* ... */
	case PPPIOCSCOMPRESS:	/* ... */
	case PPPIOCGUNIT:	/* ... */
	case PPPIOCSDEBUG:	/* ... */
	case PPPIOCGDEBUG:	/* ... */
	case PPPIOCGIDLE32:	/* ... */
	case PPPIOCGIDLE64:	/* ... */
	case PPPIOCSMAXCID:	/* ... */
	case PPPIOCGNPMODE:
	case PPPIOCSNPMODE:	/* ... */
#ifdef CONFIG_PPP_FILTER
	case PPPIOCSPASS:
	case PPPIOCSACTIVE:	/* ... */
#endif /* CONFIG_PPP_FILTER */
#ifdef CONFIG_PPP_MULTILINK
	case PPPIOCSMRRU:	/* ... */
#endif /* CONFIG_PPP_MULTILINK */
	default:
		err = -ENOTTY;
	}
	/* ... */
}
```

- Channel commands: `PPPIOCCONNECT`, `PPPIOCDISCONN`, `PPPIOCBRIDGECHAN`, `PPPIOCUNBRIDGECHAN` (`default` goes to another driver's `chan->ops->ioctl`, not traced).
- Interface commands: `PPPIOCSMRU` ... `PPPIOCSMRRU`, including cases under `#ifdef`.
- `PPPIOCDETACH` is always rejected (obsolete) → omit.
- **Crucial**: when `file->private_data` is NULL it calls `ppp_unattached_ioctl`, which likely creates/attaches units → trace it.

(Tool call: get_func_code_by_name `ppp_unattached_ioctl`)

```c
static int ppp_unattached_ioctl(struct net *net, struct ppp_file *pf,
			struct file *file, unsigned int cmd, unsigned long arg)
{
	/* ... */
	switch (cmd) {
	case PPPIOCNEWUNIT:
		/* Create a new ppp unit */
		/* ... */
	case PPPIOCATTACH:
		/* Attach to an existing ppp unit */
		/* ... */
			file->private_data = &ppp->file;
		/* ... */
	case PPPIOCATTCHAN:
		/* ... */
			file->private_data = &chan->file;
		/* ... */
	default:
		err = -ENOTTY;
	}

	return err;
}
```

Setup commands: `PPPIOCNEWUNIT`, `PPPIOCATTACH`, `PPPIOCATTCHAN`.

`init_syscall`: single char device `/dev/ppp` → `openat$ppp`.

### Final output

```json
[
	{"type": "init_syscall", "name": "openat$ppp"},
	{"type": "syscall", "name": "ioctl$PPPIOCNEWUNIT"},
	{"type": "syscall", "name": "ioctl$PPPIOCATTACH"},
	{"type": "syscall", "name": "ioctl$PPPIOCATTCHAN"},
	{"type": "syscall", "name": "ioctl$PPPIOCCONNECT"},
	{"type": "syscall", "name": "ioctl$PPPIOCDISCONN"},
	{"type": "syscall", "name": "ioctl$PPPIOCBRIDGECHAN"},
	{"type": "syscall", "name": "ioctl$PPPIOCUNBRIDGECHAN"},
	{"type": "syscall", "name": "ioctl$PPPIOCSMRU"},
	{"type": "syscall", "name": "ioctl$PPPIOCSFLAGS"},
	{"type": "syscall", "name": "ioctl$PPPIOCGFLAGS"},
	{"type": "syscall", "name": "ioctl$PPPIOCSCOMPRESS"},
	{"type": "syscall", "name": "ioctl$PPPIOCGUNIT"},
	{"type": "syscall", "name": "ioctl$PPPIOCSDEBUG"},
	{"type": "syscall", "name": "ioctl$PPPIOCGDEBUG"},
	{"type": "syscall", "name": "ioctl$PPPIOCGIDLE32"},
	{"type": "syscall", "name": "ioctl$PPPIOCGIDLE64"},
	{"type": "syscall", "name": "ioctl$PPPIOCSMAXCID"},
	{"type": "syscall", "name": "ioctl$PPPIOCGNPMODE"},
	{"type": "syscall", "name": "ioctl$PPPIOCSNPMODE"},
	{"type": "syscall", "name": "ioctl$PPPIOCSPASS"},
	{"type": "syscall", "name": "ioctl$PPPIOCSACTIVE"},
	{"type": "syscall", "name": "ioctl$PPPIOCSMRRU"}
]
```
