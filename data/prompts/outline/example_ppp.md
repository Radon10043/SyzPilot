# Example: outlining todo tasks for ppp driver

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

Focus on `ppp_ioctl`.

(Calling tool to get source code of function `ppp_ioctl`)

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

Found switch cases for `PPPIOCCONNECT`, `PPPIOCDISCONN`, etc.

**Critical**: It calls `ppp_unattached_ioctl` if `file->private_data` is NULL.

(Calling tool to get source of function `ppp_unattached_ioctl` for recursive analysis)

```c
static int ppp_unattached_ioctl(struct net *net, struct ppp_file *pf,
			struct file *file, unsigned int cmd, unsigned long arg)
{
	int unit, err = -EFAULT;
	struct ppp *ppp;
	struct channel *chan;
	struct ppp_net *pn;
	int __user *p = (int __user *)arg;

	switch (cmd) {
	case PPPIOCNEWUNIT:
		/* Create a new ppp unit */
		if (get_user(unit, p))
			break;
		err = ppp_create_interface(net, file, &unit);
		if (err < 0)
			break;

		err = -EFAULT;
		if (put_user(unit, p))
			break;
		err = 0;
		break;

	case PPPIOCATTACH:
		/* Attach to an existing ppp unit */
		if (get_user(unit, p))
			break;
		err = -ENXIO;
		pn = ppp_pernet(net);
		mutex_lock(&pn->all_ppp_mutex);
		ppp = ppp_find_unit(pn, unit);
		if (ppp) {
			refcount_inc(&ppp->file.refcnt);
			file->private_data = &ppp->file;
			err = 0;
		}
		mutex_unlock(&pn->all_ppp_mutex);
		break;

	case PPPIOCATTCHAN:
		if (get_user(unit, p))
			break;
		err = -ENXIO;
		pn = ppp_pernet(net);
		spin_lock_bh(&pn->all_channels_lock);
		chan = ppp_find_channel(pn, unit);
		if (chan) {
			refcount_inc(&chan->file.refcnt);
			file->private_data = &chan->file;
			err = 0;
		}
		spin_unlock_bh(&pn->all_channels_lock);
		break;

	default:
		err = -ENOTTY;
	}

	return err;
}
```

Found additional commands: `PPPIOCNEWUNIT`, `PPPIOCATTACH`, `PPPIOCATTCHAN`.

PPP is generally treated as a single char device `/dev/ppp`, so `openat` is sufficient.

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
		"openat$ppp",
		"ioctl$PPPIOCNEWUNIT",
		"ioctl$PPPIOCATTACH",
		"ioctl$PPPIOCATTCHAN",
		"ioctl$PPPIOCCONNECT",
		"ioctl$PPPIOCDISCONN",
		"ioctl$PPPIOCBRIDGECHAN",
		"ioctl$PPPIOCUNBRIDGECHAN",
		"ioctl$PPPIOCSMRU",
		"ioctl$PPPIOCSFLAGS",
		"ioctl$PPPIOCGFLAGS",
		"ioctl$PPPIOCSCOMPRESS",
		"ioctl$PPPIOCGUNIT",
		"ioctl$PPPIOCSDEBUG",
		"ioctl$PPPIOCGDEBUG",
		"ioctl$PPPIOCGIDLE32",
		"ioctl$PPPIOCGIDLE64",
		"ioctl$PPPIOCSMAXCID",
		"ioctl$PPPIOCGNPMODE",
		"ioctl$PPPIOCSNPMODE",
		"ioctl$PPPIOCSPASS",
		"ioctl$PPPIOCSACTIVE",
		"ioctl$PPPIOCSMRRU"
    ]
}
```