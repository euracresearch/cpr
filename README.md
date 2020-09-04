cpr - ceph placement group raiser

# Notice

**DO NOT USE THIS ANY MORE**: It's better to set
`pg_num` and `pgp_num` straight to the right value
and restrict recovery and backfilling process to 1
or any acceptable value according to the cluster 
size and usage.

```
ceph tell osd.* injectargs '--osd-max-backfills 1'
ceph tell osd.* injectargs '--osd-recovery-max-active 1'
``` 

## Usage:

```
cpr -pool my_fancy_pool -target 512
cpr -pool my_fancy_pool -target 1024 -delta 5
cpr -pool my_fancy_pool -target 256 -verbose
```

`cpr` raises ceph placement groups of a given pool step
by step. It will first raise the 'pg_num' of the pool to
the given target and then waiting 30 seconds before
proceeding with the 'pgp_num'.

Before each raise it will be checked if the cluster is
in a healthy state for raising placement groups, if not
it will wait 10 seconds before retrying. After the raise
it will wait additional 40 seconds for Ceph to recognise
the change.
