 cpr - ceph placement group raiser

 Usage:

      cpr -pool my_fancy_pool -target 512
      cpr -pool my_fancy_pool -target 1024 -delta 5
      cpr -pool my_fancy_pool -target 256 -verbose

 `cpr` raises ceph placement groups of a given pool step
 by step. It will first raise the 'pg_num' of the pool to
 the given target and then waiting 30 seconds before
 proceeding with the 'pgp_num'.

 Before each raise it will be checked if the cluster is
 in a healthy state for raising placement groups, if not
 it will wait 10 seconds before retrying. After the raise
 it will wait additional 40 seconds for Ceph to recognise
 the change.
