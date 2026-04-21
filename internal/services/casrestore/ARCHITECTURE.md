# CAS Restore Architecture

## Upload / restore invariant

For cloud189 CAS restore, **all upload-style restore flows must go through family cloud first**.

Required order:

1. Second-pass / instant restore into **family cloud**
2. Save / transfer the restored file from **family cloud** into **personal cloud**
3. Verify the file actually appears in the target **personal** folder

## Forbidden path

- Direct person upload / direct person instant-restore as the primary restore path

If a future implementation adds new adapters or optimizations, they must still preserve the family-first invariant.
